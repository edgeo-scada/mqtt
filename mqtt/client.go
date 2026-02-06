// Copyright 2025 Edgeo SCADA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mqtt

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// writeRequest wraps a packet with an optional completion signal.
type writeRequest struct {
	packet Packet
	done   chan error // signaled when write completes (nil = success)
}

// Client is an MQTT 5.0 client.
type Client struct {
	opts    *ClientOptions
	conn    net.Conn
	wsConn  *websocket.Conn
	state   atomic.Int32
	mu      sync.RWMutex
	wg      sync.WaitGroup
	done    chan struct{}
	writeCh chan *writeRequest
	metrics *Metrics
	logger  *slog.Logger

	// Packet ID management
	packetID     uint16
	packetIDLock sync.Mutex

	// Pending operations
	pendingPub     map[uint16]*PublishToken
	pendingPubLock sync.RWMutex
	pendingSub     map[uint16]*SubscribeToken
	pendingSubLock sync.RWMutex
	pendingUnsub   map[uint16]*UnsubscribeToken
	pendingUnsubLock sync.RWMutex

	// QoS 2 state
	pubRecv     map[uint16]struct{}
	pubRecvLock sync.Mutex

	// Subscriptions
	subscriptions     map[string]MessageHandler
	subscriptionsLock sync.RWMutex

	// Server properties
	serverProps *Properties

	// Topic aliases
	topicAliases     map[uint16]string
	topicAliasesLock sync.RWMutex
	outgoingAliases  map[string]uint16
	outgoingAliasSeq uint16
	aliasLock        sync.Mutex

	// Keep-alive
	lastPacketSent time.Time
	lastPacketRecv time.Time
	pingOutstanding bool
	pingLock        sync.Mutex
}

// NewClient creates a new MQTT client.
func NewClient(opts ...Option) *Client {
	options := NewClientOptions()
	for _, opt := range opts {
		opt(options)
	}

	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	c := &Client{
		opts:            options,
		done:            make(chan struct{}),
		writeCh:         make(chan *writeRequest, 100),
		metrics:         NewMetrics(),
		logger:          logger,
		pendingPub:      make(map[uint16]*PublishToken),
		pendingSub:      make(map[uint16]*SubscribeToken),
		pendingUnsub:    make(map[uint16]*UnsubscribeToken),
		pubRecv:         make(map[uint16]struct{}),
		subscriptions:   make(map[string]MessageHandler),
		topicAliases:    make(map[uint16]string),
		outgoingAliases: make(map[string]uint16),
	}

	return c
}

// Connect establishes a connection to the MQTT broker.
func (c *Client) Connect(ctx context.Context) error {
	if !c.state.CompareAndSwap(int32(StateDisconnected), int32(StateConnecting)) {
		return ErrAlreadyConnected
	}

	if len(c.opts.Servers) == 0 {
		c.state.Store(int32(StateDisconnected))
		return errors.New("mqtt: no servers configured")
	}

	c.metrics.ConnectionAttempts.Add(1)

	// Try each server
	var lastErr error
	for _, server := range c.opts.Servers {
		if err := c.connectToServer(ctx, server); err != nil {
			lastErr = err
			c.logger.Warn("failed to connect to server",
				"server", server.String(),
				"error", err)
			continue
		}
		return nil
	}

	c.state.Store(int32(StateDisconnected))
	return fmt.Errorf("mqtt: failed to connect to any server: %w", lastErr)
}

func (c *Client) connectToServer(ctx context.Context, server *url.URL) error {
	// Establish connection based on scheme
	var err error
	switch server.Scheme {
	case "mqtt", "tcp":
		err = c.connectTCP(ctx, server)
	case "mqtts", "ssl", "tls":
		err = c.connectTLS(ctx, server)
	case "ws":
		err = c.connectWS(ctx, server)
	case "wss":
		err = c.connectWSS(ctx, server)
	default:
		return fmt.Errorf("unsupported scheme: %s", server.Scheme)
	}

	if err != nil {
		return err
	}

	// Send CONNECT packet
	if err := c.sendConnect(); err != nil {
		c.closeConn()
		return err
	}

	// Wait for CONNACK
	connAck, err := c.waitConnAck(ctx)
	if err != nil {
		c.closeConn()
		return err
	}

	if connAck.ReasonCode != ReasonSuccess {
		c.closeConn()
		return &ConnectError{
			Code:       connAck.ReasonCode,
			Properties: connAck.Properties,
		}
	}

	// Store server properties
	c.serverProps = connAck.Properties

	c.state.Store(int32(StateConnected))
	c.metrics.ActiveConnections.Add(1)

	// Reset channels
	c.done = make(chan struct{})
	c.writeCh = make(chan *writeRequest, 100)

	// Start background goroutines
	c.wg.Add(3)
	go c.readLoop()
	go c.writeLoop()
	go c.keepAliveLoop()

	// Call OnConnect callback
	if c.opts.OnConnect != nil {
		go c.opts.OnConnect(c)
	}

	c.logger.Info("connected to broker",
		"server", server.String(),
		"session_present", connAck.SessionPresent)

	return nil
}

func (c *Client) connectTCP(ctx context.Context, server *url.URL) error {
	host := server.Host
	if server.Port() == "" {
		host = net.JoinHostPort(server.Hostname(), fmt.Sprintf("%d", DefaultPort))
	}

	dialer := &net.Dialer{Timeout: c.opts.ConnectTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

func (c *Client) connectTLS(ctx context.Context, server *url.URL) error {
	host := server.Host
	if server.Port() == "" {
		host = net.JoinHostPort(server.Hostname(), fmt.Sprintf("%d", DefaultTLSPort))
	}

	tlsConfig := c.opts.TLSConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{}
	}
	if tlsConfig.ServerName == "" {
		tlsConfig.ServerName = server.Hostname()
	}

	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: c.opts.ConnectTimeout},
		Config:    tlsConfig,
	}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

func (c *Client) connectWS(ctx context.Context, server *url.URL) error {
	wsURL := fmt.Sprintf("ws://%s%s", server.Host, c.opts.WebSocketPath)
	if server.Port() == "" {
		wsURL = fmt.Sprintf("ws://%s:%d%s", server.Hostname(), DefaultWSPort, c.opts.WebSocketPath)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: c.opts.ConnectTimeout,
		Subprotocols:     []string{"mqtt"},
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return err
	}

	c.wsConn = conn
	return nil
}

func (c *Client) connectWSS(ctx context.Context, server *url.URL) error {
	wsURL := fmt.Sprintf("wss://%s%s", server.Host, c.opts.WebSocketPath)
	if server.Port() == "" {
		wsURL = fmt.Sprintf("wss://%s:%d%s", server.Hostname(), DefaultWSSPort, c.opts.WebSocketPath)
	}

	tlsConfig := c.opts.TLSConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{}
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: c.opts.ConnectTimeout,
		TLSClientConfig:  tlsConfig,
		Subprotocols:     []string{"mqtt"},
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return err
	}

	c.wsConn = conn
	return nil
}

func (c *Client) sendConnect() error {
	packet := &ConnectPacket{
		ProtocolName:    ProtocolName,
		ProtocolVersion: ProtocolVersion,
		CleanStart:      c.opts.CleanStart,
		KeepAlive:       uint16(c.opts.KeepAlive.Seconds()),
		ClientID:        c.opts.ClientID,
		Properties: &Properties{
			SessionExpiryInterval: &c.opts.SessionExpiryInterval,
			ReceiveMaximum:        &c.opts.ReceiveMaximum,
			MaximumPacketSize:     &c.opts.MaxPacketSize,
			TopicAliasMaximum:     &c.opts.TopicAliasMaximum,
			UserProperties:        c.opts.UserProperties,
		},
	}

	if c.opts.RequestResponseInfo {
		reqRespInfo := byte(1)
		packet.Properties.RequestResponseInfo = &reqRespInfo
	}

	if c.opts.RequestProblemInfo {
		reqProbInfo := byte(1)
		packet.Properties.RequestProblemInfo = &reqProbInfo
	}

	if c.opts.Username != "" {
		packet.UsernameFlag = true
		packet.Username = c.opts.Username
	}

	if c.opts.Password != nil {
		packet.PasswordFlag = true
		packet.Password = c.opts.Password
	}

	if c.opts.WillEnabled {
		packet.WillFlag = true
		packet.WillQoS = c.opts.WillQoS
		packet.WillRetain = c.opts.WillRetain
		packet.WillTopic = c.opts.WillTopic
		packet.WillPayload = c.opts.WillPayload
		packet.WillProperties = c.opts.WillProperties
	}

	return c.writePacket(packet)
}

func (c *Client) waitConnAck(ctx context.Context) (*ConnAckPacket, error) {
	deadline := time.Now().Add(c.opts.ConnectTimeout)
	if err := c.setReadDeadline(deadline); err != nil {
		return nil, err
	}

	packet, err := c.readPacket()
	if err != nil {
		return nil, err
	}

	connAck, ok := packet.(*ConnAckPacket)
	if !ok {
		return nil, fmt.Errorf("expected CONNACK, got %T", packet)
	}

	return connAck, nil
}

func (c *Client) readPacket() (Packet, error) {
	if c.wsConn != nil {
		_, data, err := c.wsConn.ReadMessage()
		if err != nil {
			return nil, err
		}
		return ReadPacket(newBytesReader(data))
	}
	return ReadPacket(c.conn)
}

func (c *Client) writePacket(p Packet) error {
	data, err := p.Encode()
	if err != nil {
		return err
	}

	if c.wsConn != nil {
		return c.wsConn.WriteMessage(websocket.BinaryMessage, data)
	}

	if err := c.setWriteDeadline(time.Now().Add(c.opts.WriteTimeout)); err != nil {
		return err
	}

	_, err = c.conn.Write(data)
	c.pingLock.Lock()
	c.lastPacketSent = time.Now()
	c.pingLock.Unlock()
	return err
}

func (c *Client) setReadDeadline(t time.Time) error {
	if c.conn != nil {
		return c.conn.SetReadDeadline(t)
	}
	if c.wsConn != nil {
		return c.wsConn.SetReadDeadline(t)
	}
	return nil
}

func (c *Client) setWriteDeadline(t time.Time) error {
	if c.conn != nil {
		return c.conn.SetWriteDeadline(t)
	}
	if c.wsConn != nil {
		return c.wsConn.SetWriteDeadline(t)
	}
	return nil
}

func (c *Client) readLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.done:
			return
		default:
		}

		// Set read deadline for keep-alive
		keepAlive := c.opts.KeepAlive
		if keepAlive > 0 {
			c.setReadDeadline(time.Now().Add(keepAlive + (keepAlive / 2)))
		}

		packet, err := c.readPacket()
		if err != nil {
			select {
			case <-c.done:
				return
			default:
				c.handleConnectionLost(err)
				return
			}
		}

		c.pingLock.Lock()
		c.lastPacketRecv = time.Now()
		c.pingLock.Unlock()

		c.handlePacket(packet)
	}
}

func (c *Client) writeLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.done:
			return
		case req := <-c.writeCh:
			err := c.writePacket(req.packet)

			// Signal completion if requested
			if req.done != nil {
				req.done <- err
				close(req.done)
			}

			if err != nil {
				select {
				case <-c.done:
					return
				default:
					c.handleConnectionLost(err)
					return
				}
			}
			c.metrics.PacketsSent.Add(1)
		}
	}
}

func (c *Client) keepAliveLoop() {
	defer c.wg.Done()

	keepAlive := c.opts.KeepAlive
	if keepAlive == 0 {
		return
	}

	ticker := time.NewTicker(keepAlive / 2)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.pingLock.Lock()
			if c.pingOutstanding {
				c.pingLock.Unlock()
				c.handleConnectionLost(ErrTimeout)
				return
			}

			if time.Since(c.lastPacketSent) >= keepAlive/2 {
				c.pingOutstanding = true
				c.pingLock.Unlock()
				c.sendPing()
			} else {
				c.pingLock.Unlock()
			}
		}
	}
}

func (c *Client) sendPing() {
	select {
	case c.writeCh <- &writeRequest{packet: &PingReqPacket{}}:
	default:
	}
}

// queuePacket queues a packet for async sending (fire-and-forget).
func (c *Client) queuePacket(p Packet) {
	select {
	case c.writeCh <- &writeRequest{packet: p}:
	default:
	}
}

// sendPacketSync sends a packet and waits for it to be written.
func (c *Client) sendPacketSync(ctx context.Context, p Packet) error {
	done := make(chan error, 1)
	req := &writeRequest{packet: p, done: done}

	select {
	case c.writeCh <- req:
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return ErrNotConnected
	}

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return ErrNotConnected
	}
}

func (c *Client) handlePacket(packet Packet) {
	c.metrics.PacketsReceived.Add(1)

	switch p := packet.(type) {
	case *ConnAckPacket:
		// Already handled in Connect
	case *PublishPacket:
		c.handlePublish(p)
	case *PubAckPacket:
		c.handlePubAck(p)
	case *PubRecPacket:
		c.handlePubRec(p)
	case *PubRelPacket:
		c.handlePubRel(p)
	case *PubCompPacket:
		c.handlePubComp(p)
	case *SubAckPacket:
		c.handleSubAck(p)
	case *UnsubAckPacket:
		c.handleUnsubAck(p)
	case *PingRespPacket:
		c.pingLock.Lock()
		c.pingOutstanding = false
		c.pingLock.Unlock()
	case *DisconnectPacket:
		c.handleDisconnect(p)
	case *AuthPacket:
		c.handleAuth(p)
	}
}

func (c *Client) handlePublish(p *PublishPacket) {
	c.metrics.MessagesReceived.Add(1)

	// Handle topic alias
	topic := p.Topic
	if p.Properties != nil && p.Properties.TopicAlias != nil {
		alias := *p.Properties.TopicAlias
		if topic != "" {
			// Store the alias
			c.topicAliasesLock.Lock()
			c.topicAliases[alias] = topic
			c.topicAliasesLock.Unlock()
		} else {
			// Use stored alias
			c.topicAliasesLock.RLock()
			topic = c.topicAliases[alias]
			c.topicAliasesLock.RUnlock()
		}
	}

	msg := &Message{
		Topic:      topic,
		Payload:    p.Payload,
		QoS:        p.QoS,
		Retain:     p.Retain,
		PacketID:   p.PacketID,
		Properties: p.Properties,
		Duplicate:  p.Duplicate,
	}

	// QoS handling
	switch p.QoS {
	case QoS1:
		// Send PUBACK
		c.queuePacket(&PubAckPacket{
			PacketID:   p.PacketID,
			ReasonCode: ReasonSuccess,
		})
	case QoS2:
		// Check if we already received this
		c.pubRecvLock.Lock()
		if _, exists := c.pubRecv[p.PacketID]; exists {
			c.pubRecvLock.Unlock()
			// Already received, send PUBREC again
			c.queuePacket(&PubRecPacket{
				PacketID:   p.PacketID,
				ReasonCode: ReasonSuccess,
			})
			return
		}
		c.pubRecv[p.PacketID] = struct{}{}
		c.pubRecvLock.Unlock()

		// Send PUBREC
		c.queuePacket(&PubRecPacket{
			PacketID:   p.PacketID,
			ReasonCode: ReasonSuccess,
		})
	}

	// Deliver message
	c.deliverMessage(msg)
}

func (c *Client) deliverMessage(msg *Message) {
	// Find matching handler
	c.subscriptionsLock.RLock()
	handler := c.findHandler(msg.Topic)
	c.subscriptionsLock.RUnlock()

	if handler != nil {
		go handler(c, msg)
	} else if c.opts.DefaultMsgHandler != nil {
		go c.opts.DefaultMsgHandler(c, msg)
	}
}

func (c *Client) findHandler(topic string) MessageHandler {
	// Exact match first
	if handler, ok := c.subscriptions[topic]; ok {
		return handler
	}

	// Try wildcard matching
	for filter, handler := range c.subscriptions {
		if topicMatches(filter, topic) {
			return handler
		}
	}

	return nil
}

// topicMatches checks if a topic matches a filter with wildcards.
func topicMatches(filter, topic string) bool {
	if filter == topic {
		return true
	}

	filterParts := splitTopic(filter)
	topicParts := splitTopic(topic)

	for i, fp := range filterParts {
		if fp == "#" {
			return true
		}
		if i >= len(topicParts) {
			return false
		}
		if fp != "+" && fp != topicParts[i] {
			return false
		}
	}

	return len(filterParts) == len(topicParts)
}

func splitTopic(topic string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(topic); i++ {
		if i == len(topic) || topic[i] == '/' {
			parts = append(parts, topic[start:i])
			start = i + 1
		}
	}
	return parts
}

func (c *Client) handlePubAck(p *PubAckPacket) {
	c.pendingPubLock.Lock()
	token, ok := c.pendingPub[p.PacketID]
	if ok {
		delete(c.pendingPub, p.PacketID)
	}
	c.pendingPubLock.Unlock()

	if ok {
		if p.ReasonCode.IsError() {
			token.complete(p.ReasonCode.ToError())
		} else {
			token.complete(nil)
		}
	}
}

func (c *Client) handlePubRec(p *PubRecPacket) {
	if p.ReasonCode.IsError() {
		c.pendingPubLock.Lock()
		token, ok := c.pendingPub[p.PacketID]
		if ok {
			delete(c.pendingPub, p.PacketID)
		}
		c.pendingPubLock.Unlock()

		if ok {
			token.complete(p.ReasonCode.ToError())
		}
		return
	}

	// Send PUBREL
	c.queuePacket(&PubRelPacket{
		PacketID:   p.PacketID,
		ReasonCode: ReasonSuccess,
	})
}

func (c *Client) handlePubRel(p *PubRelPacket) {
	c.pubRecvLock.Lock()
	delete(c.pubRecv, p.PacketID)
	c.pubRecvLock.Unlock()

	// Send PUBCOMP
	c.queuePacket(&PubCompPacket{
		PacketID:   p.PacketID,
		ReasonCode: ReasonSuccess,
	})
}

func (c *Client) handlePubComp(p *PubCompPacket) {
	c.pendingPubLock.Lock()
	token, ok := c.pendingPub[p.PacketID]
	if ok {
		delete(c.pendingPub, p.PacketID)
	}
	c.pendingPubLock.Unlock()

	if ok {
		if p.ReasonCode.IsError() {
			token.complete(p.ReasonCode.ToError())
		} else {
			token.complete(nil)
		}
	}
}

func (c *Client) handleSubAck(p *SubAckPacket) {
	c.pendingSubLock.Lock()
	token, ok := c.pendingSub[p.PacketID]
	if ok {
		delete(c.pendingSub, p.PacketID)
	}
	c.pendingSubLock.Unlock()

	if ok {
		token.Properties = p.Properties
		grantedQoS := make([]QoS, len(p.ReasonCodes))
		for i, rc := range p.ReasonCodes {
			if rc.IsError() {
				token.complete(rc.ToError())
				return
			}
			grantedQoS[i] = QoS(rc)
		}
		token.GrantedQoS = grantedQoS
		token.complete(nil)
	}
}

func (c *Client) handleUnsubAck(p *UnsubAckPacket) {
	c.pendingUnsubLock.Lock()
	token, ok := c.pendingUnsub[p.PacketID]
	if ok {
		delete(c.pendingUnsub, p.PacketID)
	}
	c.pendingUnsubLock.Unlock()

	if ok {
		token.Properties = p.Properties
		for _, rc := range p.ReasonCodes {
			if rc.IsError() {
				token.complete(rc.ToError())
				return
			}
		}
		token.complete(nil)
	}
}

func (c *Client) handleDisconnect(p *DisconnectPacket) {
	c.handleConnectionLost(&DisconnectError{
		Code:       p.ReasonCode,
		Properties: p.Properties,
	})
}

func (c *Client) handleAuth(_ *AuthPacket) {
	// Enhanced authentication not implemented
	c.logger.Warn("received AUTH packet, enhanced authentication not supported")
}

func (c *Client) handleConnectionLost(err error) {
	if !c.state.CompareAndSwap(int32(StateConnected), int32(StateDisconnected)) {
		return
	}

	c.metrics.ActiveConnections.Add(-1)
	close(c.done)
	c.closeConn()

	c.logger.Info("connection lost", "error", err)

	if c.opts.OnConnectionLost != nil {
		go c.opts.OnConnectionLost(c, err)
	}

	// Fail pending operations
	c.failPending(err)

	// Auto-reconnect
	if c.opts.AutoReconnect {
		go c.reconnect()
	}
}

func (c *Client) failPending(err error) {
	c.pendingPubLock.Lock()
	for _, token := range c.pendingPub {
		token.complete(err)
	}
	c.pendingPub = make(map[uint16]*PublishToken)
	c.pendingPubLock.Unlock()

	c.pendingSubLock.Lock()
	for _, token := range c.pendingSub {
		token.complete(err)
	}
	c.pendingSub = make(map[uint16]*SubscribeToken)
	c.pendingSubLock.Unlock()

	c.pendingUnsubLock.Lock()
	for _, token := range c.pendingUnsub {
		token.complete(err)
	}
	c.pendingUnsub = make(map[uint16]*UnsubscribeToken)
	c.pendingUnsubLock.Unlock()
}

func (c *Client) reconnect() {
	backoff := c.opts.ConnectRetryInterval
	retries := 0

	for {
		if c.opts.OnReconnecting != nil {
			c.opts.OnReconnecting(c, c.opts)
		}

		c.metrics.ReconnectAttempts.Add(1)

		ctx, cancel := context.WithTimeout(context.Background(), c.opts.ConnectTimeout)
		err := c.Connect(ctx)
		cancel()

		if err == nil {
			// Resubscribe
			c.resubscribe()
			return
		}

		c.logger.Warn("reconnection failed", "error", err, "retry_in", backoff)

		retries++
		if c.opts.MaxRetries > 0 && retries >= c.opts.MaxRetries {
			c.logger.Error("max reconnection attempts reached")
			return
		}

		time.Sleep(backoff)

		// Exponential backoff with jitter
		backoff = time.Duration(float64(backoff) * (1.5 + rand.Float64()*0.5))
		if backoff > c.opts.MaxReconnectInterval {
			backoff = c.opts.MaxReconnectInterval
		}
	}
}

func (c *Client) resubscribe() {
	c.subscriptionsLock.RLock()
	topics := make([]string, 0, len(c.subscriptions))
	for topic := range c.subscriptions {
		topics = append(topics, topic)
	}
	c.subscriptionsLock.RUnlock()

	for _, topic := range topics {
		c.Subscribe(context.Background(), topic, QoS1, nil)
	}
}

func (c *Client) closeConn() {
	if c.wsConn != nil {
		c.wsConn.Close()
		c.wsConn = nil
	}
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) nextPacketID() uint16 {
	c.packetIDLock.Lock()
	defer c.packetIDLock.Unlock()

	c.packetID++
	if c.packetID == 0 {
		c.packetID = 1
	}
	return c.packetID
}

// Disconnect gracefully disconnects from the broker.
func (c *Client) Disconnect(ctx context.Context) error {
	if !c.state.CompareAndSwap(int32(StateConnected), int32(StateDisconnecting)) {
		return ErrNotConnected
	}

	// Drain pending writes before disconnecting
	c.drainWriteChannel()

	// Send DISCONNECT packet
	packet := &DisconnectPacket{
		ReasonCode: ReasonNormalDisconnection,
	}
	c.writePacket(packet)

	c.state.Store(int32(StateDisconnected))
	c.metrics.ActiveConnections.Add(-1)

	close(c.done)
	c.wg.Wait()
	c.closeConn()

	c.logger.Info("disconnected from broker")
	return nil
}

// drainWriteChannel sends all pending packets before disconnect.
func (c *Client) drainWriteChannel() {
	for {
		select {
		case req := <-c.writeCh:
			err := c.writePacket(req.packet)
			if req.done != nil {
				req.done <- err
				close(req.done)
			}
		default:
			return
		}
	}
}

// Publish publishes a message to a topic.
func (c *Client) Publish(ctx context.Context, topic string, payload []byte, qos QoS, retain bool) *PublishToken {
	token := newPublishToken()

	if c.State() != StateConnected {
		token.complete(ErrNotConnected)
		return token
	}

	packet := &PublishPacket{
		Topic:   topic,
		Payload: payload,
		QoS:     qos,
		Retain:  retain,
	}

	if qos > QoS0 {
		packet.PacketID = c.nextPacketID()
		token.PacketID = packet.PacketID

		c.pendingPubLock.Lock()
		c.pendingPub[packet.PacketID] = token
		c.pendingPubLock.Unlock()

		select {
		case c.writeCh <- &writeRequest{packet: packet}:
			c.metrics.MessagesSent.Add(1)
		case <-ctx.Done():
			token.complete(ctx.Err())
		}
	} else {
		// QoS 0: send synchronously to ensure delivery before disconnect
		if err := c.sendPacketSync(ctx, packet); err != nil {
			token.complete(err)
			return token
		}
		c.metrics.MessagesSent.Add(1)
		token.complete(nil)
	}

	return token
}

// PublishWithProperties publishes a message with MQTT 5.0 properties.
func (c *Client) PublishWithProperties(ctx context.Context, topic string, payload []byte, qos QoS, retain bool, props *Properties) *PublishToken {
	token := newPublishToken()

	if c.State() != StateConnected {
		token.complete(ErrNotConnected)
		return token
	}

	packet := &PublishPacket{
		Topic:      topic,
		Payload:    payload,
		QoS:        qos,
		Retain:     retain,
		Properties: props,
	}

	if qos > QoS0 {
		packet.PacketID = c.nextPacketID()
		token.PacketID = packet.PacketID

		c.pendingPubLock.Lock()
		c.pendingPub[packet.PacketID] = token
		c.pendingPubLock.Unlock()

		select {
		case c.writeCh <- &writeRequest{packet: packet}:
			c.metrics.MessagesSent.Add(1)
		case <-ctx.Done():
			token.complete(ctx.Err())
		}
	} else {
		// QoS 0: send synchronously to ensure delivery before disconnect
		if err := c.sendPacketSync(ctx, packet); err != nil {
			token.complete(err)
			return token
		}
		c.metrics.MessagesSent.Add(1)
		token.complete(nil)
	}

	return token
}

// Subscribe subscribes to a topic.
func (c *Client) Subscribe(ctx context.Context, topic string, qos QoS, handler MessageHandler) *SubscribeToken {
	return c.SubscribeMultiple(ctx, []Subscription{{Topic: topic, QoS: qos}}, handler)
}

// SubscribeMultiple subscribes to multiple topics.
func (c *Client) SubscribeMultiple(ctx context.Context, subs []Subscription, handler MessageHandler) *SubscribeToken {
	token := newSubscribeToken()

	if c.State() != StateConnected {
		token.complete(ErrNotConnected)
		return token
	}

	packetID := c.nextPacketID()
	packet := &SubscribePacket{
		PacketID:      packetID,
		Subscriptions: subs,
	}

	c.pendingSubLock.Lock()
	c.pendingSub[packetID] = token
	c.pendingSubLock.Unlock()

	// Store handlers
	if handler != nil {
		c.subscriptionsLock.Lock()
		for _, sub := range subs {
			c.subscriptions[sub.Topic] = handler
		}
		c.subscriptionsLock.Unlock()
	}

	select {
	case c.writeCh <- &writeRequest{packet: packet}:
	case <-ctx.Done():
		c.pendingSubLock.Lock()
		delete(c.pendingSub, packetID)
		c.pendingSubLock.Unlock()
		token.complete(ctx.Err())
	}

	return token
}

// Unsubscribe unsubscribes from a topic.
func (c *Client) Unsubscribe(ctx context.Context, topics ...string) *UnsubscribeToken {
	token := newUnsubscribeToken()

	if c.State() != StateConnected {
		token.complete(ErrNotConnected)
		return token
	}

	packetID := c.nextPacketID()
	packet := &UnsubscribePacket{
		PacketID: packetID,
		Topics:   topics,
	}

	c.pendingUnsubLock.Lock()
	c.pendingUnsub[packetID] = token
	c.pendingUnsubLock.Unlock()

	// Remove handlers
	c.subscriptionsLock.Lock()
	for _, topic := range topics {
		delete(c.subscriptions, topic)
	}
	c.subscriptionsLock.Unlock()

	select {
	case c.writeCh <- &writeRequest{packet: packet}:
	case <-ctx.Done():
		c.pendingUnsubLock.Lock()
		delete(c.pendingUnsub, packetID)
		c.pendingUnsubLock.Unlock()
		token.complete(ctx.Err())
	}

	return token
}

// State returns the current connection state.
func (c *Client) State() ConnectionState {
	return ConnectionState(c.state.Load())
}

// IsConnected returns true if connected.
func (c *Client) IsConnected() bool {
	return c.State() == StateConnected
}

// Metrics returns the client metrics.
func (c *Client) Metrics() *Metrics {
	return c.metrics
}

// ServerProperties returns the server's properties from CONNACK.
func (c *Client) ServerProperties() *Properties {
	return c.serverProps
}

// bytesReader wraps a byte slice as an io.Reader.
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, errors.New("EOF")
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
