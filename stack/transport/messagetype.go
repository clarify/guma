package transport

import (
	"bytes"

	"encoding/binary"
)

// MessageType is a three byte ASCII code that identifies the Message type
type MessageType string

// HEL, ACK & ERR are used on the socket level while OPN,CLO & MSG are used
// by the secureChannel
const (
	MessageTypeHello MessageType = "HEL"
	MessageTypeAck   MessageType = "ACK"
	MessageTypeErr   MessageType = "ERR"
	MessageTypeMsg   MessageType = "MSG"
	MessageTypeClose MessageType = "CLO"
	MessageTypeOpen  MessageType = "OPN"
)

// UnmarshalBinary decodes an OPC UA MessageType from binary data into m.
func (m *MessageType) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data[0:3])
	ar := make([]byte, 3)

	if err := binary.Read(buf, binary.LittleEndian, ar); err != nil {
		return err
	}
	*m = MessageType(ar)

	return nil
}

// MarshalBinary encodes an OPC UA MessageType to its binary representation.
func (m MessageType) MarshalBinary() ([]byte, error) {
	return []byte(m), nil
}

// BitLength returnes the total bit size of the MessageType
// It implements the Bitlenghter interface used for binary encoding / decoding
func (m *MessageType) BitLength() int {
	return 8 * 3
}
