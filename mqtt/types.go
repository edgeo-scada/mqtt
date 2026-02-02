package mqtt

import (
	"sync"
	"time"
)

// PacketType represents MQTT control packet types.
type PacketType byte

const (
	// CONNECT - Client request to connect to Server.
	PacketConnect PacketType = 1
	// CONNACK - Connect acknowledgment.
	PacketConnAck PacketType = 2
	// PUBLISH - Publish message.
	PacketPublish PacketType = 3
	// PUBACK - Publish acknowledgment (QoS 1).
	PacketPubAck PacketType = 4
	// PUBREC - Publish received (QoS 2 delivery part 1).
	PacketPubRec PacketType = 5
	// PUBREL - Publish release (QoS 2 delivery part 2).
	PacketPubRel PacketType = 6
	// PUBCOMP - Publish complete (QoS 2 delivery part 3).
	PacketPubComp PacketType = 7
	// SUBSCRIBE - Subscribe request.
	PacketSubscribe PacketType = 8
	// SUBACK - Subscribe acknowledgment.
	PacketSubAck PacketType = 9
	// UNSUBSCRIBE - Unsubscribe request.
	PacketUnsubscribe PacketType = 10
	// UNSUBACK - Unsubscribe acknowledgment.
	PacketUnsubAck PacketType = 11
	// PINGREQ - PING request.
	PacketPingReq PacketType = 12
	// PINGRESP - PING response.
	PacketPingResp PacketType = 13
	// DISCONNECT - Disconnect notification.
	PacketDisconnect PacketType = 14
	// AUTH - Authentication exchange.
	PacketAuth PacketType = 15
)

// String returns the string representation of the packet type.
func (p PacketType) String() string {
	switch p {
	case PacketConnect:
		return "CONNECT"
	case PacketConnAck:
		return "CONNACK"
	case PacketPublish:
		return "PUBLISH"
	case PacketPubAck:
		return "PUBACK"
	case PacketPubRec:
		return "PUBREC"
	case PacketPubRel:
		return "PUBREL"
	case PacketPubComp:
		return "PUBCOMP"
	case PacketSubscribe:
		return "SUBSCRIBE"
	case PacketSubAck:
		return "SUBACK"
	case PacketUnsubscribe:
		return "UNSUBSCRIBE"
	case PacketUnsubAck:
		return "UNSUBACK"
	case PacketPingReq:
		return "PINGREQ"
	case PacketPingResp:
		return "PINGRESP"
	case PacketDisconnect:
		return "DISCONNECT"
	case PacketAuth:
		return "AUTH"
	default:
		return "UNKNOWN"
	}
}

// QoS represents MQTT Quality of Service levels.
type QoS byte

const (
	// QoS0 - At most once delivery.
	QoS0 QoS = 0
	// QoS1 - At least once delivery.
	QoS1 QoS = 1
	// QoS2 - Exactly once delivery.
	QoS2 QoS = 2
)

// String returns the string representation of the QoS level.
func (q QoS) String() string {
	switch q {
	case QoS0:
		return "QoS0 (At most once)"
	case QoS1:
		return "QoS1 (At least once)"
	case QoS2:
		return "QoS2 (Exactly once)"
	default:
		return "Unknown QoS"
	}
}

// ConnectionState represents the state of a client connection.
type ConnectionState int

const (
	// StateDisconnected indicates the client is not connected.
	StateDisconnected ConnectionState = iota
	// StateConnecting indicates the client is attempting to connect.
	StateConnecting
	// StateConnected indicates the client is connected and ready.
	StateConnected
	// StateDisconnecting indicates the client is gracefully disconnecting.
	StateDisconnecting
)

// String returns the string representation of the connection state.
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateDisconnecting:
		return "Disconnecting"
	default:
		return "Unknown"
	}
}

// RetainHandling specifies how retained messages are handled on subscribe.
type RetainHandling byte

const (
	// RetainSendAtSubscribe - Send retained messages at subscribe time.
	RetainSendAtSubscribe RetainHandling = 0
	// RetainSendIfNew - Send retained only if subscription is new.
	RetainSendIfNew RetainHandling = 1
	// RetainDoNotSend - Do not send retained messages.
	RetainDoNotSend RetainHandling = 2
)

// PropertyID represents MQTT 5.0 property identifiers.
type PropertyID byte

const (
	PropPayloadFormatIndicator   PropertyID = 0x01
	PropMessageExpiryInterval    PropertyID = 0x02
	PropContentType              PropertyID = 0x03
	PropResponseTopic            PropertyID = 0x08
	PropCorrelationData          PropertyID = 0x09
	PropSubscriptionIdentifier   PropertyID = 0x0B
	PropSessionExpiryInterval    PropertyID = 0x11
	PropAssignedClientIdentifier PropertyID = 0x12
	PropServerKeepAlive          PropertyID = 0x13
	PropAuthenticationMethod     PropertyID = 0x15
	PropAuthenticationData       PropertyID = 0x16
	PropRequestProblemInfo       PropertyID = 0x17
	PropWillDelayInterval        PropertyID = 0x18
	PropRequestResponseInfo      PropertyID = 0x19
	PropResponseInfo             PropertyID = 0x1A
	PropServerReference          PropertyID = 0x1C
	PropReasonString             PropertyID = 0x1F
	PropReceiveMaximum           PropertyID = 0x21
	PropTopicAliasMaximum        PropertyID = 0x22
	PropTopicAlias               PropertyID = 0x23
	PropMaximumQoS               PropertyID = 0x24
	PropRetainAvailable          PropertyID = 0x25
	PropUserProperty             PropertyID = 0x26
	PropMaximumPacketSize        PropertyID = 0x27
	PropWildcardSubAvailable     PropertyID = 0x28
	PropSubIdentifierAvailable   PropertyID = 0x29
	PropSharedSubAvailable       PropertyID = 0x2A
)

// Properties represents MQTT 5.0 properties.
type Properties struct {
	// Payload format: 0 = unspecified bytes, 1 = UTF-8 encoded
	PayloadFormat *byte
	// Message expiry interval in seconds
	MessageExpiry *uint32
	// Content type (MIME type)
	ContentType string
	// Response topic for request/response pattern
	ResponseTopic string
	// Correlation data for request/response pattern
	CorrelationData []byte
	// Subscription identifiers
	SubscriptionIdentifier []uint32
	// Session expiry interval in seconds
	SessionExpiryInterval *uint32
	// Client identifier assigned by server
	AssignedClientID string
	// Server keep alive override
	ServerKeepAlive *uint16
	// Authentication method
	AuthMethod string
	// Authentication data
	AuthData []byte
	// Request problem information
	RequestProblemInfo *byte
	// Will delay interval
	WillDelayInterval *uint32
	// Request response information
	RequestResponseInfo *byte
	// Response information
	ResponseInfo string
	// Server reference for redirect
	ServerReference string
	// Reason string for diagnostics
	ReasonString string
	// Receive maximum (server flow control)
	ReceiveMaximum *uint16
	// Topic alias maximum
	TopicAliasMaximum *uint16
	// Topic alias
	TopicAlias *uint16
	// Maximum QoS supported
	MaximumQoS *byte
	// Retain available flag
	RetainAvailable *byte
	// User properties (key-value pairs)
	UserProperties []UserProperty
	// Maximum packet size
	MaximumPacketSize *uint32
	// Wildcard subscription available
	WildcardSubAvailable *byte
	// Subscription identifier available
	SubIDAvailable *byte
	// Shared subscription available
	SharedSubAvailable *byte
}

// UserProperty represents a user-defined property.
type UserProperty struct {
	Key   string
	Value string
}

// Message represents a published MQTT message.
type Message struct {
	// Topic is the message topic.
	Topic string
	// Payload is the message payload.
	Payload []byte
	// QoS is the quality of service level.
	QoS QoS
	// Retain indicates if the message should be retained.
	Retain bool
	// PacketID is the packet identifier (QoS > 0).
	PacketID uint16
	// Properties contains MQTT 5.0 properties.
	Properties *Properties
	// Duplicate indicates if this is a re-delivery.
	Duplicate bool
}

// Subscription represents a topic subscription.
type Subscription struct {
	// Topic is the subscription topic filter.
	Topic string
	// QoS is the maximum QoS level.
	QoS QoS
	// NoLocal prevents messages from being sent to the publisher.
	NoLocal bool
	// RetainAsPublished keeps the retain flag.
	RetainAsPublished bool
	// RetainHandling specifies retained message behavior.
	RetainHandling RetainHandling
}

// Token represents an asynchronous operation result.
type Token interface {
	// Wait blocks until the operation completes.
	Wait() error
	// WaitTimeout blocks until completion or timeout.
	WaitTimeout(timeout time.Duration) error
	// Done returns a channel that closes when complete.
	Done() <-chan struct{}
	// Error returns the error, if any.
	Error() error
}

// token implements the Token interface.
type token struct {
	done chan struct{}
	err  error
	mu   sync.Mutex
}

// newToken creates a new token.
func newToken() *token {
	return &token{
		done: make(chan struct{}),
	}
}

// Wait blocks until the operation completes.
func (t *token) Wait() error {
	<-t.done
	return t.err
}

// WaitTimeout blocks until completion or timeout.
func (t *token) WaitTimeout(timeout time.Duration) error {
	select {
	case <-t.done:
		return t.err
	case <-time.After(timeout):
		return ErrTimeout
	}
}

// Done returns a channel that closes when complete.
func (t *token) Done() <-chan struct{} {
	return t.done
}

// Error returns the error, if any.
func (t *token) Error() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.err
}

// complete marks the token as complete.
func (t *token) complete(err error) {
	t.mu.Lock()
	t.err = err
	t.mu.Unlock()
	close(t.done)
}

// ConnectToken is returned from Connect operations.
type ConnectToken struct {
	*token
	SessionPresent bool
	Properties     *Properties
}

// newConnectToken creates a new connect token.
func newConnectToken() *ConnectToken {
	return &ConnectToken{
		token: newToken(),
	}
}

// PublishToken is returned from Publish operations.
type PublishToken struct {
	*token
	PacketID uint16
}

// newPublishToken creates a new publish token.
func newPublishToken() *PublishToken {
	return &PublishToken{
		token: newToken(),
	}
}

// SubscribeToken is returned from Subscribe operations.
type SubscribeToken struct {
	*token
	GrantedQoS []QoS
	Properties *Properties
}

// newSubscribeToken creates a new subscribe token.
func newSubscribeToken() *SubscribeToken {
	return &SubscribeToken{
		token: newToken(),
	}
}

// UnsubscribeToken is returned from Unsubscribe operations.
type UnsubscribeToken struct {
	*token
	Properties *Properties
}

// newUnsubscribeToken creates a new unsubscribe token.
func newUnsubscribeToken() *UnsubscribeToken {
	return &UnsubscribeToken{
		token: newToken(),
	}
}

// MessageHandler is a callback for received messages.
type MessageHandler func(client *Client, msg *Message)

// ConnectionLostHandler is a callback for connection loss.
type ConnectionLostHandler func(client *Client, err error)

// OnConnectHandler is a callback for successful connection.
type OnConnectHandler func(client *Client)

// ReconnectHandler is a callback for reconnection attempts.
type ReconnectHandler func(client *Client, opts *ClientOptions)

// Default values.
const (
	DefaultKeepAlive         = 60 * time.Second
	DefaultConnectTimeout    = 30 * time.Second
	DefaultWriteTimeout      = 10 * time.Second
	DefaultPingTimeout       = 10 * time.Second
	DefaultMaxReconnectDelay = 2 * time.Minute
	DefaultReceiveMaximum    = 65535
	DefaultMaxPacketSize     = 268435456 // 256 MB
	DefaultTopicAliasMaximum = 0
	DefaultPort              = 1883
	DefaultTLSPort           = 8883
	DefaultWSPort            = 80
	DefaultWSSPort           = 443
)
