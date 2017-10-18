package tcp

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/searis/guma/stack/transport"
)

// ChunkType is a one byte ASCII code that indicates whether the MessageChunk is the final chunk in a Message.
type ChunkType byte

/*
The following values are defined for ChunkTypes:
C An intermediate chunk.
F The final chunk.
A The final chunk (used when an error occurred and the Message is aborted).
This field is only meaningful for MessageType of ‘MSG’ This field is always ‘F’ for other MessageTypes.
*/
const (
	ChunkTypeIntermediate ChunkType = 'C'
	ChunkTypeFinal        ChunkType = 'F'
	ChunkTypeFinalAborted ChunkType = 'A'
	MessageHeaderByteSize int       = 12
)

// MessageHeader is what starts every OPC UA TCP Message
type MessageHeader struct {
	Type            transport.MessageType
	ChunkType       ChunkType
	Size            uint32
	SecureChannelID uint32
}

func (m MessageHeader) String() string {
	return fmt.Sprintf(`
		---- MessageHeader ----
		Type:		%s
		ChunkType:	%c
		Size:		%d
		SecChanID:	%d
		---- /MessageHeader ----`, m.Type, m.ChunkType, m.Size, m.SecureChannelID)
}

// UnmarshalBinary decodes an OPC UA MessageHeader from byte array
// data into MessageHeader struct m.
func (m *MessageHeader) UnmarshalBinary(data []byte) error {

	err := m.Type.UnmarshalBinary(data)
	if err != nil {
		return err
	}
	m.ChunkType = ChunkType(data[3])
	m.Size = binary.LittleEndian.Uint32(data[4:])
	m.SecureChannelID = binary.LittleEndian.Uint32(data[8:])
	return nil
}

// MarshalBinary encodes a MessageHeader struct into a OPC UA MessageHeader
// representation byte array.
func (m MessageHeader) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	mt, err := m.Type.MarshalBinary()
	if err != nil {
		return nil, err
	}
	binary.Write(&buf, binary.LittleEndian, mt)
	binary.Write(&buf, binary.LittleEndian, m.ChunkType)
	binary.Write(&buf, binary.LittleEndian, m.Size)
	binary.Write(&buf, binary.LittleEndian, m.SecureChannelID)
	return buf.Bytes(), nil
}

// BitLength returnes the total bit size of the MessageHeader
// It implements the Bitlenghter interface used for binary encoding / decoding
func (m *MessageHeader) BitLength() int {
	return int(8 * MessageHeaderByteSize)
}
