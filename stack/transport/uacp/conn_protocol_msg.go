package uacp

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/searis/guma/stack/uatype"
)

// msgStringMaxLen is the maximum length in characters legal for strings visible
// to the UACP layer. I.e. strings in HEL, ACK, ERR and RHE message bodies.
const msgStingMaxLen = 4096

// handshakeMaxSize describes the maximum buffer size needed to buffer any
// messages that may occur during an UACP handshake.
const handshakeMaxSize = msgHeaderSize + helloMsgMaxSize

// msgType is a three byte ASCII code that identifies the message type at the
// UACP layer.
type msgType string

// Message types that are understood at the UACP layer.
const (
	msgTypeHel msgType = "HEL"
	msgTypeAck msgType = "ACK"
	msgTypeErr msgType = "ERR"
	msgTypeRhe msgType = "RHE" // Introduced in OPC UA 1.04, not yet supported.
)

// Message types that are not understood for the UACP layer, but should be
// passed on to secure channel.
const (
	msgTypeOpn msgType = "OPN"
	msgTypeClo msgType = "CLO"
	msgTypeMsg msgType = "MSG"
)

// msgHeader implements OPC UA Connection Protocol Message Header and is
// compatible with the first 8 bytes of the OPC UA Secure Conversation Message
// Header (secureMsgHeader).
//
// As the implemented is a byte-slice, it is possible to cast the head of a
// message buffer to a msgHeader, and then read or write values directly from
// the underlying buffer.
//
// Always assure a buffer casted to msgHeader, is more than 8 bytes, or panics
// and silent errors are likely to occur.
//
// Memory layout:
// - Type        [3]byte
// - Reserved    byte      // Always `F` for messags at this layer.
// - Size        uint32
type msgHeader []byte

const msgHeaderSize = 8
const msgHeaderFmt = `--- Message Header ---
Type:              %3s
Reserved:          %3c
Size:           %6d
-- /Message Header ---`

// getMsgHeader validates that b is large enough to contain a header, and then
// casts the head of the buffer into a msgHeader. Any changes to msgHeader will
// appear in b and wise versa.
func getMsgHeader(b []byte) (msgHeader, error) {
	if len(b) < msgHeaderSize {
		return nil, errors.New("getMsgHeader: not enough data")
	}
	return msgHeader(b[0:msgHeaderSize]), nil
}

// String returns a nicely formatted representation of the data in h.
func (h msgHeader) String() string {
	if len(h) < msgHeaderSize {
		return fmt.Sprintf("<Invalid length msgHeader>")
	}
	return fmt.Sprintf(msgHeaderFmt, h.Type(), h[3], h.Size())
}

// Type returns the message type from h.
func (h msgHeader) Type() msgType {
	return msgType(h[0:3])
}

// Size returns the message size from h.
func (h msgHeader) Size() uint32 {
	return binary.LittleEndian.Uint32(h[4:8])
}

// SetHelloHeader sets h to an OPC UA Connection Protocol Message Header for the
// Hello Message. size should always be set to include both the size og h and
// the size of the message.
func (h msgHeader) SetHelloHeader(size uint32) {
	copy(h, []byte{'H', 'E', 'L', 'F', 0, 0, 0, 0})
	h.SetSize(size)
}

// SetAckHeader sets h to an OPC UA Connection Protocol Message Header for the
// Acknowledge Message. The message size, which is fixed, is also set in the
// header.
func (h msgHeader) SetAckHeader() {
	// the ACK message size is fixed and less than 255 (0xFF), so we can set it
	// directly in the byte-stream.
	copy(h, []byte{'A', 'C', 'K', 'F', msgHeaderSize + ackMsgSize, 0, 0, 0})
}

// SetErrHeader sets h to an OPC UA Connection Protocol Message Header for the
// Error Message. size should always be set to include both the size og h and
// the size of the message.
func (h msgHeader) SetErrHeader(size uint32) {
	copy(h, []byte{'E', 'R', 'R', 'F', 0, 0, 0, 0})
	h.SetSize(size)
}

// SetSize overwrite the message size section in h. size should always be set to
// include both the size og h and the size of the message.
func (h msgHeader) SetSize(size uint32) {
	binary.LittleEndian.PutUint32(h[4:8], size)
}

type helloMsg struct {
	Version     uint32
	MsgChunking MsgChunking
	EndpointURL string
}

func helloMsgSize(endpointURL string) uint32 {
	return uint32(6*4 + len([]byte(endpointURL)))
}

const helloMsgMaxSize = 6*4 + msgStingMaxLen

type revHelloMsg struct {
	ServerURI   string
	EndpointURL string
}

func revHelloMsgSize(serverURI, endpointURL string) uint32 {
	return uint32(2*4 + len([]byte(serverURI)) + len([]byte(endpointURL)))
}

const revHelloMsgMaxSize = 2*4 + 2*msgStingMaxLen

type ackMsg struct {
	Version     uint32
	MsgChunking MsgChunking
}

const ackMsgSize = 5 * 4

type errMsg struct {
	Error  uatype.StatusCode
	Reason string
}

const errMsgMaxSize = 2*4 + msgStingMaxLen
