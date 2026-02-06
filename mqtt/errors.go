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
	"errors"
	"fmt"
)

// Standard errors.
var (
	ErrNotConnected        = errors.New("mqtt: not connected")
	ErrAlreadyConnected    = errors.New("mqtt: already connected")
	ErrConnectionLost      = errors.New("mqtt: connection lost")
	ErrTimeout             = errors.New("mqtt: operation timed out")
	ErrInvalidQoS          = errors.New("mqtt: invalid QoS level")
	ErrInvalidTopic        = errors.New("mqtt: invalid topic")
	ErrInvalidPacketID     = errors.New("mqtt: invalid packet ID")
	ErrInvalidProtocol     = errors.New("mqtt: invalid protocol")
	ErrPacketTooLarge      = errors.New("mqtt: packet too large")
	ErrMalformedPacket     = errors.New("mqtt: malformed packet")
	ErrProtocolError       = errors.New("mqtt: protocol error")
	ErrClientClosed        = errors.New("mqtt: client closed")
	ErrInvalidClientID     = errors.New("mqtt: invalid client ID")
	ErrBadUsernamePassword = errors.New("mqtt: bad username or password")
	ErrNotAuthorized       = errors.New("mqtt: not authorized")
	ErrServerUnavailable   = errors.New("mqtt: server unavailable")
	ErrServerBusy          = errors.New("mqtt: server busy")
	ErrBanned              = errors.New("mqtt: banned")
	ErrBadAuthMethod       = errors.New("mqtt: bad authentication method")
	ErrTopicNameInvalid    = errors.New("mqtt: topic name invalid")
	ErrReceiveMaxExceeded  = errors.New("mqtt: receive maximum exceeded")
	ErrTopicAliasInvalid   = errors.New("mqtt: topic alias invalid")
	ErrPacketIDInUse       = errors.New("mqtt: packet identifier in use")
	ErrQuotaExceeded       = errors.New("mqtt: quota exceeded")
	ErrPayloadFormatInvalid = errors.New("mqtt: payload format invalid")
	ErrRetainNotSupported  = errors.New("mqtt: retain not supported")
	ErrQoSNotSupported     = errors.New("mqtt: QoS not supported")
	ErrUseAnotherServer    = errors.New("mqtt: use another server")
	ErrServerMoved         = errors.New("mqtt: server moved")
	ErrSharedSubsNotSupported = errors.New("mqtt: shared subscriptions not supported")
	ErrSubIDNotSupported   = errors.New("mqtt: subscription identifiers not supported")
	ErrWildcardSubsNotSupported = errors.New("mqtt: wildcard subscriptions not supported")
)

// ReasonCode represents MQTT 5.0 reason codes.
type ReasonCode byte

// MQTT 5.0 Reason Codes.
const (
	ReasonSuccess                     ReasonCode = 0x00
	ReasonNormalDisconnection         ReasonCode = 0x00
	ReasonGrantedQoS0                 ReasonCode = 0x00
	ReasonGrantedQoS1                 ReasonCode = 0x01
	ReasonGrantedQoS2                 ReasonCode = 0x02
	ReasonDisconnectWithWill          ReasonCode = 0x04
	ReasonNoMatchingSubscribers       ReasonCode = 0x10
	ReasonNoSubscriptionExisted       ReasonCode = 0x11
	ReasonContinueAuthentication      ReasonCode = 0x18
	ReasonReAuthenticate              ReasonCode = 0x19
	ReasonUnspecifiedError            ReasonCode = 0x80
	ReasonMalformedPacket             ReasonCode = 0x81
	ReasonProtocolError               ReasonCode = 0x82
	ReasonImplementationError         ReasonCode = 0x83
	ReasonUnsupportedProtocolVersion  ReasonCode = 0x84
	ReasonClientIDNotValid            ReasonCode = 0x85
	ReasonBadUsernameOrPassword       ReasonCode = 0x86
	ReasonNotAuthorized               ReasonCode = 0x87
	ReasonServerUnavailable           ReasonCode = 0x88
	ReasonServerBusy                  ReasonCode = 0x89
	ReasonBanned                      ReasonCode = 0x8A
	ReasonServerShuttingDown          ReasonCode = 0x8B
	ReasonBadAuthenticationMethod     ReasonCode = 0x8C
	ReasonKeepAliveTimeout            ReasonCode = 0x8D
	ReasonSessionTakenOver            ReasonCode = 0x8E
	ReasonTopicFilterInvalid          ReasonCode = 0x8F
	ReasonTopicNameInvalid            ReasonCode = 0x90
	ReasonPacketIDInUse               ReasonCode = 0x91
	ReasonPacketIDNotFound            ReasonCode = 0x92
	ReasonReceiveMaximumExceeded      ReasonCode = 0x93
	ReasonTopicAliasInvalid           ReasonCode = 0x94
	ReasonPacketTooLarge              ReasonCode = 0x95
	ReasonMessageRateTooHigh          ReasonCode = 0x96
	ReasonQuotaExceeded               ReasonCode = 0x97
	ReasonAdministrativeAction        ReasonCode = 0x98
	ReasonPayloadFormatInvalid        ReasonCode = 0x99
	ReasonRetainNotSupported          ReasonCode = 0x9A
	ReasonQoSNotSupported             ReasonCode = 0x9B
	ReasonUseAnotherServer            ReasonCode = 0x9C
	ReasonServerMoved                 ReasonCode = 0x9D
	ReasonSharedSubsNotSupported      ReasonCode = 0x9E
	ReasonConnectionRateExceeded      ReasonCode = 0x9F
	ReasonMaxConnectTime              ReasonCode = 0xA0
	ReasonSubIDNotSupported           ReasonCode = 0xA1
	ReasonWildcardSubsNotSupported    ReasonCode = 0xA2
)

// String returns the string representation of the reason code.
func (r ReasonCode) String() string {
	switch r {
	case ReasonSuccess:
		return "Success"
	case ReasonGrantedQoS1:
		return "Granted QoS 1"
	case ReasonGrantedQoS2:
		return "Granted QoS 2"
	case ReasonDisconnectWithWill:
		return "Disconnect with Will Message"
	case ReasonNoMatchingSubscribers:
		return "No matching subscribers"
	case ReasonNoSubscriptionExisted:
		return "No subscription existed"
	case ReasonContinueAuthentication:
		return "Continue authentication"
	case ReasonReAuthenticate:
		return "Re-authenticate"
	case ReasonUnspecifiedError:
		return "Unspecified error"
	case ReasonMalformedPacket:
		return "Malformed Packet"
	case ReasonProtocolError:
		return "Protocol Error"
	case ReasonImplementationError:
		return "Implementation specific error"
	case ReasonUnsupportedProtocolVersion:
		return "Unsupported Protocol Version"
	case ReasonClientIDNotValid:
		return "Client Identifier not valid"
	case ReasonBadUsernameOrPassword:
		return "Bad User Name or Password"
	case ReasonNotAuthorized:
		return "Not authorized"
	case ReasonServerUnavailable:
		return "Server unavailable"
	case ReasonServerBusy:
		return "Server busy"
	case ReasonBanned:
		return "Banned"
	case ReasonServerShuttingDown:
		return "Server shutting down"
	case ReasonBadAuthenticationMethod:
		return "Bad authentication method"
	case ReasonKeepAliveTimeout:
		return "Keep Alive timeout"
	case ReasonSessionTakenOver:
		return "Session taken over"
	case ReasonTopicFilterInvalid:
		return "Topic Filter invalid"
	case ReasonTopicNameInvalid:
		return "Topic Name invalid"
	case ReasonPacketIDInUse:
		return "Packet Identifier in use"
	case ReasonPacketIDNotFound:
		return "Packet Identifier not found"
	case ReasonReceiveMaximumExceeded:
		return "Receive Maximum exceeded"
	case ReasonTopicAliasInvalid:
		return "Topic Alias invalid"
	case ReasonPacketTooLarge:
		return "Packet too large"
	case ReasonMessageRateTooHigh:
		return "Message rate too high"
	case ReasonQuotaExceeded:
		return "Quota exceeded"
	case ReasonAdministrativeAction:
		return "Administrative action"
	case ReasonPayloadFormatInvalid:
		return "Payload format invalid"
	case ReasonRetainNotSupported:
		return "Retain not supported"
	case ReasonQoSNotSupported:
		return "QoS not supported"
	case ReasonUseAnotherServer:
		return "Use another server"
	case ReasonServerMoved:
		return "Server moved"
	case ReasonSharedSubsNotSupported:
		return "Shared Subscriptions not supported"
	case ReasonConnectionRateExceeded:
		return "Connection rate exceeded"
	case ReasonMaxConnectTime:
		return "Maximum connect time"
	case ReasonSubIDNotSupported:
		return "Subscription Identifiers not supported"
	case ReasonWildcardSubsNotSupported:
		return "Wildcard Subscriptions not supported"
	default:
		return fmt.Sprintf("Unknown reason code: 0x%02X", byte(r))
	}
}

// IsError returns true if the reason code indicates an error.
func (r ReasonCode) IsError() bool {
	return r >= 0x80
}

// ToError converts a reason code to an error.
func (r ReasonCode) ToError() error {
	if !r.IsError() {
		return nil
	}

	switch r {
	case ReasonMalformedPacket:
		return ErrMalformedPacket
	case ReasonProtocolError:
		return ErrProtocolError
	case ReasonClientIDNotValid:
		return ErrInvalidClientID
	case ReasonBadUsernameOrPassword:
		return ErrBadUsernamePassword
	case ReasonNotAuthorized:
		return ErrNotAuthorized
	case ReasonServerUnavailable:
		return ErrServerUnavailable
	case ReasonServerBusy:
		return ErrServerBusy
	case ReasonBanned:
		return ErrBanned
	case ReasonBadAuthenticationMethod:
		return ErrBadAuthMethod
	case ReasonTopicNameInvalid:
		return ErrTopicNameInvalid
	case ReasonPacketIDInUse:
		return ErrPacketIDInUse
	case ReasonReceiveMaximumExceeded:
		return ErrReceiveMaxExceeded
	case ReasonTopicAliasInvalid:
		return ErrTopicAliasInvalid
	case ReasonPacketTooLarge:
		return ErrPacketTooLarge
	case ReasonQuotaExceeded:
		return ErrQuotaExceeded
	case ReasonPayloadFormatInvalid:
		return ErrPayloadFormatInvalid
	case ReasonRetainNotSupported:
		return ErrRetainNotSupported
	case ReasonQoSNotSupported:
		return ErrQoSNotSupported
	case ReasonUseAnotherServer:
		return ErrUseAnotherServer
	case ReasonServerMoved:
		return ErrServerMoved
	case ReasonSharedSubsNotSupported:
		return ErrSharedSubsNotSupported
	case ReasonSubIDNotSupported:
		return ErrSubIDNotSupported
	case ReasonWildcardSubsNotSupported:
		return ErrWildcardSubsNotSupported
	default:
		return &MQTTError{Code: r, Message: r.String()}
	}
}

// MQTTError represents an MQTT protocol error.
type MQTTError struct {
	Code       ReasonCode
	Message    string
	Properties *Properties
}

// Error implements the error interface.
func (e *MQTTError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("mqtt: %s (0x%02X): %s", e.Code.String(), byte(e.Code), e.Message)
	}
	return fmt.Sprintf("mqtt: %s (0x%02X)", e.Code.String(), byte(e.Code))
}

// NewMQTTError creates a new MQTT error.
func NewMQTTError(code ReasonCode, message string) *MQTTError {
	return &MQTTError{
		Code:    code,
		Message: message,
	}
}

// ConnectError represents a connection error with additional details.
type ConnectError struct {
	Code       ReasonCode
	Properties *Properties
}

// Error implements the error interface.
func (e *ConnectError) Error() string {
	msg := fmt.Sprintf("mqtt: connection refused: %s (0x%02X)", e.Code.String(), byte(e.Code))
	if e.Properties != nil && e.Properties.ReasonString != "" {
		msg += ": " + e.Properties.ReasonString
	}
	return msg
}

// DisconnectError represents a disconnection from the server.
type DisconnectError struct {
	Code       ReasonCode
	Properties *Properties
}

// Error implements the error interface.
func (e *DisconnectError) Error() string {
	msg := fmt.Sprintf("mqtt: disconnected: %s (0x%02X)", e.Code.String(), byte(e.Code))
	if e.Properties != nil && e.Properties.ReasonString != "" {
		msg += ": " + e.Properties.ReasonString
	}
	return msg
}
