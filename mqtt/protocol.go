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
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Packet interface for all MQTT packets.
type Packet interface {
	// Type returns the packet type.
	Type() PacketType
	// Encode encodes the packet to bytes.
	Encode() ([]byte, error)
}

// FixedHeader represents the MQTT fixed header.
type FixedHeader struct {
	Type      PacketType
	Flags     byte
	Remaining int
}

// Encode encodes the fixed header to bytes.
func (h *FixedHeader) Encode() []byte {
	buf := make([]byte, 1, 5)
	buf[0] = byte(h.Type)<<4 | (h.Flags & 0x0F)
	buf = append(buf, encodeVarInt(h.Remaining)...)
	return buf
}

// encodeVarInt encodes an integer using MQTT variable-length encoding.
func encodeVarInt(value int) []byte {
	var buf []byte
	for {
		b := byte(value % 128)
		value /= 128
		if value > 0 {
			b |= 0x80
		}
		buf = append(buf, b)
		if value == 0 {
			break
		}
	}
	return buf
}

// decodeVarInt decodes a variable-length integer from a reader.
func decodeVarInt(r io.Reader) (int, int, error) {
	var value int
	var multiplier = 1
	var bytesRead int
	b := make([]byte, 1)

	for {
		if bytesRead >= 4 {
			return 0, bytesRead, ErrMalformedPacket
		}
		_, err := r.Read(b)
		if err != nil {
			return 0, bytesRead, err
		}
		bytesRead++
		value += int(b[0]&0x7F) * multiplier
		if b[0]&0x80 == 0 {
			break
		}
		multiplier *= 128
	}

	return value, bytesRead, nil
}

// encodeString encodes a UTF-8 string with length prefix.
func encodeString(s string) []byte {
	buf := make([]byte, 2+len(s))
	binary.BigEndian.PutUint16(buf, uint16(len(s)))
	copy(buf[2:], s)
	return buf
}

// decodeString decodes a UTF-8 string from a reader.
func decodeString(r io.Reader) (string, error) {
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return "", err
	}
	length := binary.BigEndian.Uint16(lenBuf)
	if length == 0 {
		return "", nil
	}
	strBuf := make([]byte, length)
	if _, err := io.ReadFull(r, strBuf); err != nil {
		return "", err
	}
	return string(strBuf), nil
}

// encodeBinary encodes binary data with length prefix.
func encodeBinary(data []byte) []byte {
	buf := make([]byte, 2+len(data))
	binary.BigEndian.PutUint16(buf, uint16(len(data)))
	copy(buf[2:], data)
	return buf
}

// decodeBinary decodes binary data from a reader.
func decodeBinary(r io.Reader) ([]byte, error) {
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint16(lenBuf)
	if length == 0 {
		return nil, nil
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}

// encodeProperties encodes MQTT 5.0 properties.
func encodeProperties(props *Properties) []byte {
	if props == nil {
		return []byte{0}
	}

	var buf bytes.Buffer

	if props.PayloadFormat != nil {
		buf.WriteByte(byte(PropPayloadFormatIndicator))
		buf.WriteByte(*props.PayloadFormat)
	}

	if props.MessageExpiry != nil {
		buf.WriteByte(byte(PropMessageExpiryInterval))
		binary.Write(&buf, binary.BigEndian, *props.MessageExpiry)
	}

	if props.ContentType != "" {
		buf.WriteByte(byte(PropContentType))
		buf.Write(encodeString(props.ContentType))
	}

	if props.ResponseTopic != "" {
		buf.WriteByte(byte(PropResponseTopic))
		buf.Write(encodeString(props.ResponseTopic))
	}

	if props.CorrelationData != nil {
		buf.WriteByte(byte(PropCorrelationData))
		buf.Write(encodeBinary(props.CorrelationData))
	}

	for _, subID := range props.SubscriptionIdentifier {
		buf.WriteByte(byte(PropSubscriptionIdentifier))
		buf.Write(encodeVarInt(int(subID)))
	}

	if props.SessionExpiryInterval != nil {
		buf.WriteByte(byte(PropSessionExpiryInterval))
		binary.Write(&buf, binary.BigEndian, *props.SessionExpiryInterval)
	}

	if props.AssignedClientID != "" {
		buf.WriteByte(byte(PropAssignedClientIdentifier))
		buf.Write(encodeString(props.AssignedClientID))
	}

	if props.ServerKeepAlive != nil {
		buf.WriteByte(byte(PropServerKeepAlive))
		binary.Write(&buf, binary.BigEndian, *props.ServerKeepAlive)
	}

	if props.AuthMethod != "" {
		buf.WriteByte(byte(PropAuthenticationMethod))
		buf.Write(encodeString(props.AuthMethod))
	}

	if props.AuthData != nil {
		buf.WriteByte(byte(PropAuthenticationData))
		buf.Write(encodeBinary(props.AuthData))
	}

	if props.RequestProblemInfo != nil {
		buf.WriteByte(byte(PropRequestProblemInfo))
		buf.WriteByte(*props.RequestProblemInfo)
	}

	if props.WillDelayInterval != nil {
		buf.WriteByte(byte(PropWillDelayInterval))
		binary.Write(&buf, binary.BigEndian, *props.WillDelayInterval)
	}

	if props.RequestResponseInfo != nil {
		buf.WriteByte(byte(PropRequestResponseInfo))
		buf.WriteByte(*props.RequestResponseInfo)
	}

	if props.ResponseInfo != "" {
		buf.WriteByte(byte(PropResponseInfo))
		buf.Write(encodeString(props.ResponseInfo))
	}

	if props.ServerReference != "" {
		buf.WriteByte(byte(PropServerReference))
		buf.Write(encodeString(props.ServerReference))
	}

	if props.ReasonString != "" {
		buf.WriteByte(byte(PropReasonString))
		buf.Write(encodeString(props.ReasonString))
	}

	if props.ReceiveMaximum != nil {
		buf.WriteByte(byte(PropReceiveMaximum))
		binary.Write(&buf, binary.BigEndian, *props.ReceiveMaximum)
	}

	if props.TopicAliasMaximum != nil {
		buf.WriteByte(byte(PropTopicAliasMaximum))
		binary.Write(&buf, binary.BigEndian, *props.TopicAliasMaximum)
	}

	if props.TopicAlias != nil {
		buf.WriteByte(byte(PropTopicAlias))
		binary.Write(&buf, binary.BigEndian, *props.TopicAlias)
	}

	if props.MaximumQoS != nil {
		buf.WriteByte(byte(PropMaximumQoS))
		buf.WriteByte(*props.MaximumQoS)
	}

	if props.RetainAvailable != nil {
		buf.WriteByte(byte(PropRetainAvailable))
		buf.WriteByte(*props.RetainAvailable)
	}

	for _, up := range props.UserProperties {
		buf.WriteByte(byte(PropUserProperty))
		buf.Write(encodeString(up.Key))
		buf.Write(encodeString(up.Value))
	}

	if props.MaximumPacketSize != nil {
		buf.WriteByte(byte(PropMaximumPacketSize))
		binary.Write(&buf, binary.BigEndian, *props.MaximumPacketSize)
	}

	if props.WildcardSubAvailable != nil {
		buf.WriteByte(byte(PropWildcardSubAvailable))
		buf.WriteByte(*props.WildcardSubAvailable)
	}

	if props.SubIDAvailable != nil {
		buf.WriteByte(byte(PropSubIdentifierAvailable))
		buf.WriteByte(*props.SubIDAvailable)
	}

	if props.SharedSubAvailable != nil {
		buf.WriteByte(byte(PropSharedSubAvailable))
		buf.WriteByte(*props.SharedSubAvailable)
	}

	propBytes := buf.Bytes()
	result := encodeVarInt(len(propBytes))
	return append(result, propBytes...)
}

// decodeProperties decodes MQTT 5.0 properties from a reader.
func decodeProperties(r io.Reader) (*Properties, error) {
	length, _, err := decodeVarInt(r)
	if err != nil {
		return nil, err
	}

	if length == 0 {
		return nil, nil
	}

	propBytes := make([]byte, length)
	if _, err := io.ReadFull(r, propBytes); err != nil {
		return nil, err
	}

	props := &Properties{}
	pr := bytes.NewReader(propBytes)

	for pr.Len() > 0 {
		propID, err := pr.ReadByte()
		if err != nil {
			return nil, err
		}

		switch PropertyID(propID) {
		case PropPayloadFormatIndicator:
			b, err := pr.ReadByte()
			if err != nil {
				return nil, err
			}
			props.PayloadFormat = &b

		case PropMessageExpiryInterval:
			var v uint32
			if err := binary.Read(pr, binary.BigEndian, &v); err != nil {
				return nil, err
			}
			props.MessageExpiry = &v

		case PropContentType:
			s, err := decodeString(pr)
			if err != nil {
				return nil, err
			}
			props.ContentType = s

		case PropResponseTopic:
			s, err := decodeString(pr)
			if err != nil {
				return nil, err
			}
			props.ResponseTopic = s

		case PropCorrelationData:
			data, err := decodeBinary(pr)
			if err != nil {
				return nil, err
			}
			props.CorrelationData = data

		case PropSubscriptionIdentifier:
			v, _, err := decodeVarInt(pr)
			if err != nil {
				return nil, err
			}
			props.SubscriptionIdentifier = append(props.SubscriptionIdentifier, uint32(v))

		case PropSessionExpiryInterval:
			var v uint32
			if err := binary.Read(pr, binary.BigEndian, &v); err != nil {
				return nil, err
			}
			props.SessionExpiryInterval = &v

		case PropAssignedClientIdentifier:
			s, err := decodeString(pr)
			if err != nil {
				return nil, err
			}
			props.AssignedClientID = s

		case PropServerKeepAlive:
			var v uint16
			if err := binary.Read(pr, binary.BigEndian, &v); err != nil {
				return nil, err
			}
			props.ServerKeepAlive = &v

		case PropAuthenticationMethod:
			s, err := decodeString(pr)
			if err != nil {
				return nil, err
			}
			props.AuthMethod = s

		case PropAuthenticationData:
			data, err := decodeBinary(pr)
			if err != nil {
				return nil, err
			}
			props.AuthData = data

		case PropRequestProblemInfo:
			b, err := pr.ReadByte()
			if err != nil {
				return nil, err
			}
			props.RequestProblemInfo = &b

		case PropWillDelayInterval:
			var v uint32
			if err := binary.Read(pr, binary.BigEndian, &v); err != nil {
				return nil, err
			}
			props.WillDelayInterval = &v

		case PropRequestResponseInfo:
			b, err := pr.ReadByte()
			if err != nil {
				return nil, err
			}
			props.RequestResponseInfo = &b

		case PropResponseInfo:
			s, err := decodeString(pr)
			if err != nil {
				return nil, err
			}
			props.ResponseInfo = s

		case PropServerReference:
			s, err := decodeString(pr)
			if err != nil {
				return nil, err
			}
			props.ServerReference = s

		case PropReasonString:
			s, err := decodeString(pr)
			if err != nil {
				return nil, err
			}
			props.ReasonString = s

		case PropReceiveMaximum:
			var v uint16
			if err := binary.Read(pr, binary.BigEndian, &v); err != nil {
				return nil, err
			}
			props.ReceiveMaximum = &v

		case PropTopicAliasMaximum:
			var v uint16
			if err := binary.Read(pr, binary.BigEndian, &v); err != nil {
				return nil, err
			}
			props.TopicAliasMaximum = &v

		case PropTopicAlias:
			var v uint16
			if err := binary.Read(pr, binary.BigEndian, &v); err != nil {
				return nil, err
			}
			props.TopicAlias = &v

		case PropMaximumQoS:
			b, err := pr.ReadByte()
			if err != nil {
				return nil, err
			}
			props.MaximumQoS = &b

		case PropRetainAvailable:
			b, err := pr.ReadByte()
			if err != nil {
				return nil, err
			}
			props.RetainAvailable = &b

		case PropUserProperty:
			key, err := decodeString(pr)
			if err != nil {
				return nil, err
			}
			value, err := decodeString(pr)
			if err != nil {
				return nil, err
			}
			props.UserProperties = append(props.UserProperties, UserProperty{Key: key, Value: value})

		case PropMaximumPacketSize:
			var v uint32
			if err := binary.Read(pr, binary.BigEndian, &v); err != nil {
				return nil, err
			}
			props.MaximumPacketSize = &v

		case PropWildcardSubAvailable:
			b, err := pr.ReadByte()
			if err != nil {
				return nil, err
			}
			props.WildcardSubAvailable = &b

		case PropSubIdentifierAvailable:
			b, err := pr.ReadByte()
			if err != nil {
				return nil, err
			}
			props.SubIDAvailable = &b

		case PropSharedSubAvailable:
			b, err := pr.ReadByte()
			if err != nil {
				return nil, err
			}
			props.SharedSubAvailable = &b

		default:
			return nil, fmt.Errorf("unknown property ID: 0x%02X", propID)
		}
	}

	return props, nil
}

// ReadPacket reads and decodes an MQTT packet from a reader.
func ReadPacket(r io.Reader) (Packet, error) {
	// Read fixed header first byte
	headerByte := make([]byte, 1)
	if _, err := io.ReadFull(r, headerByte); err != nil {
		return nil, err
	}

	packetType := PacketType(headerByte[0] >> 4)
	flags := headerByte[0] & 0x0F

	// Read remaining length
	remaining, _, err := decodeVarInt(r)
	if err != nil {
		return nil, err
	}

	// Read payload
	payload := make([]byte, remaining)
	if remaining > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
	}

	// Decode packet based on type
	pr := bytes.NewReader(payload)

	switch packetType {
	case PacketConnAck:
		return decodeConnAck(pr)
	case PacketPublish:
		return decodePublish(pr, flags, remaining)
	case PacketPubAck:
		return decodePubAck(pr)
	case PacketPubRec:
		return decodePubRec(pr)
	case PacketPubRel:
		return decodePubRel(pr)
	case PacketPubComp:
		return decodePubComp(pr)
	case PacketSubAck:
		return decodeSubAck(pr)
	case PacketUnsubAck:
		return decodeUnsubAck(pr)
	case PacketPingResp:
		return &PingRespPacket{}, nil
	case PacketDisconnect:
		return decodeDisconnect(pr)
	case PacketAuth:
		return decodeAuth(pr)
	default:
		return nil, fmt.Errorf("unexpected packet type: %s", packetType)
	}
}

// WritePacket encodes and writes an MQTT packet to a writer.
func WritePacket(w io.Writer, p Packet) error {
	data, err := p.Encode()
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
