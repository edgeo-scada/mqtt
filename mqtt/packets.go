package mqtt

import (
	"bytes"
	"encoding/binary"
	"io"
)

// ConnectPacket represents a CONNECT packet.
type ConnectPacket struct {
	// Protocol name (MQTT)
	ProtocolName string
	// Protocol version (5 for MQTT 5.0)
	ProtocolVersion byte
	// Clean start flag
	CleanStart bool
	// Will flag
	WillFlag bool
	// Will QoS
	WillQoS QoS
	// Will retain flag
	WillRetain bool
	// Username flag
	UsernameFlag bool
	// Password flag
	PasswordFlag bool
	// Keep alive interval in seconds
	KeepAlive uint16
	// Properties
	Properties *Properties
	// Client identifier
	ClientID string
	// Will properties
	WillProperties *Properties
	// Will topic
	WillTopic string
	// Will payload
	WillPayload []byte
	// Username
	Username string
	// Password
	Password []byte
}

// Type returns the packet type.
func (p *ConnectPacket) Type() PacketType {
	return PacketConnect
}

// Encode encodes the CONNECT packet.
func (p *ConnectPacket) Encode() ([]byte, error) {
	var buf bytes.Buffer

	// Variable header
	// Protocol name
	buf.Write(encodeString(ProtocolName))
	// Protocol version
	buf.WriteByte(ProtocolVersion)

	// Connect flags
	var flags byte
	if p.CleanStart {
		flags |= 0x02
	}
	if p.WillFlag {
		flags |= 0x04
		flags |= byte(p.WillQoS) << 3
		if p.WillRetain {
			flags |= 0x20
		}
	}
	if p.PasswordFlag {
		flags |= 0x40
	}
	if p.UsernameFlag {
		flags |= 0x80
	}
	buf.WriteByte(flags)

	// Keep alive
	binary.Write(&buf, binary.BigEndian, p.KeepAlive)

	// Properties
	buf.Write(encodeProperties(p.Properties))

	// Payload
	// Client ID
	buf.Write(encodeString(p.ClientID))

	// Will properties and message
	if p.WillFlag {
		buf.Write(encodeProperties(p.WillProperties))
		buf.Write(encodeString(p.WillTopic))
		buf.Write(encodeBinary(p.WillPayload))
	}

	// Username
	if p.UsernameFlag {
		buf.Write(encodeString(p.Username))
	}

	// Password
	if p.PasswordFlag {
		buf.Write(encodeBinary(p.Password))
	}

	// Build complete packet
	payload := buf.Bytes()
	header := &FixedHeader{
		Type:      PacketConnect,
		Remaining: len(payload),
	}

	return append(header.Encode(), payload...), nil
}

// ConnAckPacket represents a CONNACK packet.
type ConnAckPacket struct {
	// Session present flag
	SessionPresent bool
	// Reason code
	ReasonCode ReasonCode
	// Properties
	Properties *Properties
}

// Type returns the packet type.
func (p *ConnAckPacket) Type() PacketType {
	return PacketConnAck
}

// Encode encodes the CONNACK packet.
func (p *ConnAckPacket) Encode() ([]byte, error) {
	var buf bytes.Buffer

	// Connect acknowledge flags
	var flags byte
	if p.SessionPresent {
		flags |= 0x01
	}
	buf.WriteByte(flags)

	// Reason code
	buf.WriteByte(byte(p.ReasonCode))

	// Properties
	buf.Write(encodeProperties(p.Properties))

	payload := buf.Bytes()
	header := &FixedHeader{
		Type:      PacketConnAck,
		Remaining: len(payload),
	}

	return append(header.Encode(), payload...), nil
}

func decodeConnAck(r io.Reader) (*ConnAckPacket, error) {
	p := &ConnAckPacket{}

	// Read flags
	flags := make([]byte, 1)
	if _, err := io.ReadFull(r, flags); err != nil {
		return nil, err
	}
	p.SessionPresent = flags[0]&0x01 != 0

	// Read reason code
	rc := make([]byte, 1)
	if _, err := io.ReadFull(r, rc); err != nil {
		return nil, err
	}
	p.ReasonCode = ReasonCode(rc[0])

	// Read properties
	props, err := decodeProperties(r)
	if err != nil {
		return nil, err
	}
	p.Properties = props

	return p, nil
}

// PublishPacket represents a PUBLISH packet.
type PublishPacket struct {
	// Duplicate delivery flag
	Duplicate bool
	// QoS level
	QoS QoS
	// Retain flag
	Retain bool
	// Topic name
	Topic string
	// Packet identifier (QoS > 0)
	PacketID uint16
	// Properties
	Properties *Properties
	// Payload
	Payload []byte
}

// Type returns the packet type.
func (p *PublishPacket) Type() PacketType {
	return PacketPublish
}

// Encode encodes the PUBLISH packet.
func (p *PublishPacket) Encode() ([]byte, error) {
	var buf bytes.Buffer

	// Topic name
	buf.Write(encodeString(p.Topic))

	// Packet identifier (QoS > 0)
	if p.QoS > QoS0 {
		binary.Write(&buf, binary.BigEndian, p.PacketID)
	}

	// Properties
	buf.Write(encodeProperties(p.Properties))

	// Payload
	buf.Write(p.Payload)

	// Build flags
	var flags byte
	if p.Duplicate {
		flags |= 0x08
	}
	flags |= byte(p.QoS) << 1
	if p.Retain {
		flags |= 0x01
	}

	payload := buf.Bytes()
	header := &FixedHeader{
		Type:      PacketPublish,
		Flags:     flags,
		Remaining: len(payload),
	}

	return append(header.Encode(), payload...), nil
}

func decodePublish(r io.Reader, flags byte, remaining int) (*PublishPacket, error) {
	p := &PublishPacket{
		Duplicate: flags&0x08 != 0,
		QoS:       QoS((flags >> 1) & 0x03),
		Retain:    flags&0x01 != 0,
	}

	// Read topic
	topic, err := decodeString(r)
	if err != nil {
		return nil, err
	}
	p.Topic = topic
	remaining -= 2 + len(topic)

	// Read packet ID (QoS > 0)
	if p.QoS > QoS0 {
		var packetID uint16
		if err := binary.Read(r, binary.BigEndian, &packetID); err != nil {
			return nil, err
		}
		p.PacketID = packetID
		remaining -= 2
	}

	// Read properties
	props, err := decodeProperties(r)
	if err != nil {
		return nil, err
	}
	p.Properties = props

	// Calculate remaining payload size (approximate)
	// Read remaining as payload
	payload := make([]byte, 0, remaining)
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			payload = append(payload, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	p.Payload = payload

	return p, nil
}

// PubAckPacket represents a PUBACK packet.
type PubAckPacket struct {
	PacketID   uint16
	ReasonCode ReasonCode
	Properties *Properties
}

// Type returns the packet type.
func (p *PubAckPacket) Type() PacketType {
	return PacketPubAck
}

// Encode encodes the PUBACK packet.
func (p *PubAckPacket) Encode() ([]byte, error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, p.PacketID)

	// Only include reason code and properties if not success
	if p.ReasonCode != ReasonSuccess || p.Properties != nil {
		buf.WriteByte(byte(p.ReasonCode))
		buf.Write(encodeProperties(p.Properties))
	}

	payload := buf.Bytes()
	header := &FixedHeader{
		Type:      PacketPubAck,
		Remaining: len(payload),
	}

	return append(header.Encode(), payload...), nil
}

func decodePubAck(r *bytes.Reader) (*PubAckPacket, error) {
	p := &PubAckPacket{}

	if err := binary.Read(r, binary.BigEndian, &p.PacketID); err != nil {
		return nil, err
	}

	if r.Len() > 0 {
		rc, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		p.ReasonCode = ReasonCode(rc)

		if r.Len() > 0 {
			props, err := decodeProperties(r)
			if err != nil {
				return nil, err
			}
			p.Properties = props
		}
	}

	return p, nil
}

// PubRecPacket represents a PUBREC packet.
type PubRecPacket struct {
	PacketID   uint16
	ReasonCode ReasonCode
	Properties *Properties
}

// Type returns the packet type.
func (p *PubRecPacket) Type() PacketType {
	return PacketPubRec
}

// Encode encodes the PUBREC packet.
func (p *PubRecPacket) Encode() ([]byte, error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, p.PacketID)

	if p.ReasonCode != ReasonSuccess || p.Properties != nil {
		buf.WriteByte(byte(p.ReasonCode))
		buf.Write(encodeProperties(p.Properties))
	}

	payload := buf.Bytes()
	header := &FixedHeader{
		Type:      PacketPubRec,
		Remaining: len(payload),
	}

	return append(header.Encode(), payload...), nil
}

func decodePubRec(r *bytes.Reader) (*PubRecPacket, error) {
	p := &PubRecPacket{}

	if err := binary.Read(r, binary.BigEndian, &p.PacketID); err != nil {
		return nil, err
	}

	if r.Len() > 0 {
		rc, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		p.ReasonCode = ReasonCode(rc)

		if r.Len() > 0 {
			props, err := decodeProperties(r)
			if err != nil {
				return nil, err
			}
			p.Properties = props
		}
	}

	return p, nil
}

// PubRelPacket represents a PUBREL packet.
type PubRelPacket struct {
	PacketID   uint16
	ReasonCode ReasonCode
	Properties *Properties
}

// Type returns the packet type.
func (p *PubRelPacket) Type() PacketType {
	return PacketPubRel
}

// Encode encodes the PUBREL packet.
func (p *PubRelPacket) Encode() ([]byte, error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, p.PacketID)

	if p.ReasonCode != ReasonSuccess || p.Properties != nil {
		buf.WriteByte(byte(p.ReasonCode))
		buf.Write(encodeProperties(p.Properties))
	}

	payload := buf.Bytes()
	header := &FixedHeader{
		Type:      PacketPubRel,
		Flags:     0x02, // Required flag
		Remaining: len(payload),
	}

	return append(header.Encode(), payload...), nil
}

func decodePubRel(r *bytes.Reader) (*PubRelPacket, error) {
	p := &PubRelPacket{}

	if err := binary.Read(r, binary.BigEndian, &p.PacketID); err != nil {
		return nil, err
	}

	if r.Len() > 0 {
		rc, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		p.ReasonCode = ReasonCode(rc)

		if r.Len() > 0 {
			props, err := decodeProperties(r)
			if err != nil {
				return nil, err
			}
			p.Properties = props
		}
	}

	return p, nil
}

// PubCompPacket represents a PUBCOMP packet.
type PubCompPacket struct {
	PacketID   uint16
	ReasonCode ReasonCode
	Properties *Properties
}

// Type returns the packet type.
func (p *PubCompPacket) Type() PacketType {
	return PacketPubComp
}

// Encode encodes the PUBCOMP packet.
func (p *PubCompPacket) Encode() ([]byte, error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, p.PacketID)

	if p.ReasonCode != ReasonSuccess || p.Properties != nil {
		buf.WriteByte(byte(p.ReasonCode))
		buf.Write(encodeProperties(p.Properties))
	}

	payload := buf.Bytes()
	header := &FixedHeader{
		Type:      PacketPubComp,
		Remaining: len(payload),
	}

	return append(header.Encode(), payload...), nil
}

func decodePubComp(r *bytes.Reader) (*PubCompPacket, error) {
	p := &PubCompPacket{}

	if err := binary.Read(r, binary.BigEndian, &p.PacketID); err != nil {
		return nil, err
	}

	if r.Len() > 0 {
		rc, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		p.ReasonCode = ReasonCode(rc)

		if r.Len() > 0 {
			props, err := decodeProperties(r)
			if err != nil {
				return nil, err
			}
			p.Properties = props
		}
	}

	return p, nil
}

// SubscribePacket represents a SUBSCRIBE packet.
type SubscribePacket struct {
	PacketID      uint16
	Properties    *Properties
	Subscriptions []Subscription
}

// Type returns the packet type.
func (p *SubscribePacket) Type() PacketType {
	return PacketSubscribe
}

// Encode encodes the SUBSCRIBE packet.
func (p *SubscribePacket) Encode() ([]byte, error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, p.PacketID)
	buf.Write(encodeProperties(p.Properties))

	for _, sub := range p.Subscriptions {
		buf.Write(encodeString(sub.Topic))
		// Subscription options
		var opts byte
		opts |= byte(sub.QoS)
		if sub.NoLocal {
			opts |= 0x04
		}
		if sub.RetainAsPublished {
			opts |= 0x08
		}
		opts |= byte(sub.RetainHandling) << 4
		buf.WriteByte(opts)
	}

	payload := buf.Bytes()
	header := &FixedHeader{
		Type:      PacketSubscribe,
		Flags:     0x02, // Required flag
		Remaining: len(payload),
	}

	return append(header.Encode(), payload...), nil
}

// SubAckPacket represents a SUBACK packet.
type SubAckPacket struct {
	PacketID    uint16
	Properties  *Properties
	ReasonCodes []ReasonCode
}

// Type returns the packet type.
func (p *SubAckPacket) Type() PacketType {
	return PacketSubAck
}

// Encode encodes the SUBACK packet.
func (p *SubAckPacket) Encode() ([]byte, error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, p.PacketID)
	buf.Write(encodeProperties(p.Properties))

	for _, rc := range p.ReasonCodes {
		buf.WriteByte(byte(rc))
	}

	payload := buf.Bytes()
	header := &FixedHeader{
		Type:      PacketSubAck,
		Remaining: len(payload),
	}

	return append(header.Encode(), payload...), nil
}

func decodeSubAck(r *bytes.Reader) (*SubAckPacket, error) {
	p := &SubAckPacket{}

	if err := binary.Read(r, binary.BigEndian, &p.PacketID); err != nil {
		return nil, err
	}

	props, err := decodeProperties(r)
	if err != nil {
		return nil, err
	}
	p.Properties = props

	for r.Len() > 0 {
		rc, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		p.ReasonCodes = append(p.ReasonCodes, ReasonCode(rc))
	}

	return p, nil
}

// UnsubscribePacket represents an UNSUBSCRIBE packet.
type UnsubscribePacket struct {
	PacketID   uint16
	Properties *Properties
	Topics     []string
}

// Type returns the packet type.
func (p *UnsubscribePacket) Type() PacketType {
	return PacketUnsubscribe
}

// Encode encodes the UNSUBSCRIBE packet.
func (p *UnsubscribePacket) Encode() ([]byte, error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, p.PacketID)
	buf.Write(encodeProperties(p.Properties))

	for _, topic := range p.Topics {
		buf.Write(encodeString(topic))
	}

	payload := buf.Bytes()
	header := &FixedHeader{
		Type:      PacketUnsubscribe,
		Flags:     0x02, // Required flag
		Remaining: len(payload),
	}

	return append(header.Encode(), payload...), nil
}

// UnsubAckPacket represents an UNSUBACK packet.
type UnsubAckPacket struct {
	PacketID    uint16
	Properties  *Properties
	ReasonCodes []ReasonCode
}

// Type returns the packet type.
func (p *UnsubAckPacket) Type() PacketType {
	return PacketUnsubAck
}

// Encode encodes the UNSUBACK packet.
func (p *UnsubAckPacket) Encode() ([]byte, error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, p.PacketID)
	buf.Write(encodeProperties(p.Properties))

	for _, rc := range p.ReasonCodes {
		buf.WriteByte(byte(rc))
	}

	payload := buf.Bytes()
	header := &FixedHeader{
		Type:      PacketUnsubAck,
		Remaining: len(payload),
	}

	return append(header.Encode(), payload...), nil
}

func decodeUnsubAck(r *bytes.Reader) (*UnsubAckPacket, error) {
	p := &UnsubAckPacket{}

	if err := binary.Read(r, binary.BigEndian, &p.PacketID); err != nil {
		return nil, err
	}

	props, err := decodeProperties(r)
	if err != nil {
		return nil, err
	}
	p.Properties = props

	for r.Len() > 0 {
		rc, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		p.ReasonCodes = append(p.ReasonCodes, ReasonCode(rc))
	}

	return p, nil
}

// PingReqPacket represents a PINGREQ packet.
type PingReqPacket struct{}

// Type returns the packet type.
func (p *PingReqPacket) Type() PacketType {
	return PacketPingReq
}

// Encode encodes the PINGREQ packet.
func (p *PingReqPacket) Encode() ([]byte, error) {
	return []byte{byte(PacketPingReq) << 4, 0}, nil
}

// PingRespPacket represents a PINGRESP packet.
type PingRespPacket struct{}

// Type returns the packet type.
func (p *PingRespPacket) Type() PacketType {
	return PacketPingResp
}

// Encode encodes the PINGRESP packet.
func (p *PingRespPacket) Encode() ([]byte, error) {
	return []byte{byte(PacketPingResp) << 4, 0}, nil
}

// DisconnectPacket represents a DISCONNECT packet.
type DisconnectPacket struct {
	ReasonCode ReasonCode
	Properties *Properties
}

// Type returns the packet type.
func (p *DisconnectPacket) Type() PacketType {
	return PacketDisconnect
}

// Encode encodes the DISCONNECT packet.
func (p *DisconnectPacket) Encode() ([]byte, error) {
	var buf bytes.Buffer

	if p.ReasonCode != ReasonSuccess || p.Properties != nil {
		buf.WriteByte(byte(p.ReasonCode))
		buf.Write(encodeProperties(p.Properties))
	}

	payload := buf.Bytes()
	header := &FixedHeader{
		Type:      PacketDisconnect,
		Remaining: len(payload),
	}

	return append(header.Encode(), payload...), nil
}

func decodeDisconnect(r *bytes.Reader) (*DisconnectPacket, error) {
	p := &DisconnectPacket{}

	if r.Len() > 0 {
		rc, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		p.ReasonCode = ReasonCode(rc)

		if r.Len() > 0 {
			props, err := decodeProperties(r)
			if err != nil {
				return nil, err
			}
			p.Properties = props
		}
	}

	return p, nil
}

// AuthPacket represents an AUTH packet.
type AuthPacket struct {
	ReasonCode ReasonCode
	Properties *Properties
}

// Type returns the packet type.
func (p *AuthPacket) Type() PacketType {
	return PacketAuth
}

// Encode encodes the AUTH packet.
func (p *AuthPacket) Encode() ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteByte(byte(p.ReasonCode))
	buf.Write(encodeProperties(p.Properties))

	payload := buf.Bytes()
	header := &FixedHeader{
		Type:      PacketAuth,
		Remaining: len(payload),
	}

	return append(header.Encode(), payload...), nil
}

func decodeAuth(r *bytes.Reader) (*AuthPacket, error) {
	p := &AuthPacket{}

	if r.Len() > 0 {
		rc, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		p.ReasonCode = ReasonCode(rc)

		if r.Len() > 0 {
			props, err := decodeProperties(r)
			if err != nil {
				return nil, err
			}
			p.Properties = props
		}
	}

	return p, nil
}
