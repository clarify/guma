package uacp

import (
	"time"

	"github.com/searis/guma/stack/uatype"
)

// ChunkType is a one byte ASCII code that indicates whether the MessageChunk is
// the final chunk in a message at the OPC UA Secure Conversation layer.
type chunkType byte

func (chunkType) BitLength() int {
	return 8
}

const (
	// chunkTypeIntermediate indicates that a mesage has more chunks.
	chunkTypeIntermediate chunkType = 'C'

	// chunkTypeFinal indicates that this is the final chunk fo the message.
	chunkTypeFinal chunkType = 'F'

	// chunkTypeFinalAborted indicates that the messsage sending was aborted
	// before it could finish.
	chunkTypeFinalAborted chunkType = 'A'
)

// secureMsgType defines the message types that are understood at the
// SecureChannelLevel.
var (
	secureMsgTypeOpn = [3]byte{'O', 'P', 'N'}
	secureMsgTypeClo = [3]byte{'C', 'L', 'O'}
	secureMsgTypeMsg = [3]byte{'M', 'S', 'G'}
)

// secureMsgHeader implements OPC UA OPC UA Secure Conversation Message Header.
type secureMsgHeader struct {
	Type            [3]byte
	ChunkType       chunkType
	Size            uint32
	SecureChannelID uint32
}

// setChunkType can be called on a send buffer instance that starts with a
// secureMsgHeader. It will panic with an index out of range if b is not 4 bytes
// or more.
func setChunkType(b []byte, t chunkType) {
	b[3] = byte(t)
}

// msgType returns h.Type as a msgType, useful for switch cases.
func (h secureMsgHeader) msgType() msgType {
	return msgType(h.Type[0:3])
}

const secureMsgHeaderSize = 12

type symmetricAlgorithmSecurityHeader struct {
	TokenID uint32
}

const symmetricAlgorithmSecurityHeaderSize = 4

type sequenceHeader struct {
	SequenceNumber uint32
	RequestID      uint32
}

const sequenceHeaderSize = 8

type secureAbortBody struct {
	Status uatype.StatusCode
	Reason string
}

func encodeUnsignedDuration(t time.Duration) uint32 {
	return uint32(t / time.Millisecond)

}

func decodeUnsignedDuration(i uint32) time.Duration {
	return time.Duration(i) * time.Millisecond

}
