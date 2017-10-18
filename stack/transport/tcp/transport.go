package tcp

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"github.com/searis/guma/stack/transport"

	"github.com/searis/guma/stack/encoding/binary"
)

// Defaults for tcp Conn, these are used when no user defined
// parameters are provided.
const (
	DefaultReceiveBufferSize uint32        = 1024 * 64 * 10
	DefaultSendBufferSize    uint32        = 1024 * 64 * 10
	DefaultMaxMessageSize    uint32        = 0
	DefaultMaxChunkCount     uint32        = 0
	DefaultVersion           uint32        = 0
	DefaultDialTimeout       time.Duration = 0
)

// Conn is a tcp implementation of the transport.Conn interface.
// If no values are provided when creating the Conn struct sane defaults will be used.
type Conn struct {
	conn              net.Conn
	ReceiveBufferSize uint32
	SendBufferSize    uint32
	MaxMessageSize    uint32
	MaxChunkCount     uint32
	DialTimeout       time.Duration
	rcvBuffer         *bytes.Buffer
}

// Connect establishes a connection to the given endpoint.
// Parameters defined in the Conn struct will be negotiated with the server during the
// HEL <-> ACK handshake.
func (t *Conn) Connect(endpoint string) error {
	//todo parse endpoint, split addr and port

	conn, err := net.DialTimeout("tcp", endpoint, t.dialTimeout())
	if err != nil {
		return err
	}
	t.conn = conn

	binary.SetDebugLogger(log.New(os.Stderr, "", 0))

	// Initiate connection by sending a Hello Message
	err = t.sendHello(endpoint)
	if err != nil {
		t.conn.Close()
		return err
	}

	// Receive and process ACK Message
	err = t.receiveAck()
	if err != nil {
		t.conn.Close()
		return err
	}

	return nil
}

// Send transmits a message of type mt to the connected endpoint. All bytes available in the io.Reader
// will be transmitted before this method returns. If no secChanID is given the value 0 will be used.
func (t *Conn) Send(mt transport.MessageType, r io.Reader, secChanID uint32) error {
	if t.conn == nil {
		return fmt.Errorf("not connected")
	}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	//Todo: implement chunking if len(data) > t.ReceiveBufferSize
	fmt.Printf("About to send %s size: %d bytes on secID: %d\n\n", mt, len(data), secChanID)
	header := MessageHeader{
		Type:            mt,
		ChunkType:       ChunkTypeFinal,
		Size:            uint32(MessageHeaderByteSize + len(data)),
		SecureChannelID: secChanID,
	}
	var buf bytes.Buffer
	enc := binary.NewEncoder(&buf)
	enc.Encode(header)
	enc.Encode(data)
	n, err := t.conn.Write(buf.Bytes())
	if err != nil {
		return err
	}
	fmt.Println(header)
	fmt.Printf(">> Sent data:\n%s", typeFmt.Sdump(buf.Bytes()))

	if n < 0 {
		// todo return correct error
		return fmt.Errorf("negative send")
	}

	return nil
}

// Receive will receive a message from the connected endpoint and return a io.Reader where a complete
// message can be read. If the message is divided into multiple chunks this method will receive and
// assemble them all before returning.
func (t *Conn) Receive() (io.Reader, error) {
	if t.conn == nil {
		return nil, fmt.Errorf("not connected")
	}
	// Read header
	header, err := t.readHeader()
	if err != nil {
		return nil, err
	}
	fmt.Println(header)

	hbuf := make([]byte, header.Size)
	//TODO: Implement timeout
	_, err = t.conn.Read(hbuf)
	if err != nil && err != io.EOF {
		return nil, err
	}

	switch header.ChunkType {
	case ChunkTypeFinal:
		if t.rcvBuffer == nil {
			fmt.Printf("<< Received data:\n%s", typeFmt.Sdump(hbuf))
			return bytes.NewReader(hbuf), nil

		}
		t.rcvBuffer.Write(hbuf)
		fmt.Printf("<< Received data:\n%s", typeFmt.Sdump(t.rcvBuffer.Bytes()))
		return t.rcvBuffer, nil
	case ChunkTypeFinalAborted:
	case ChunkTypeIntermediate:
		if t.rcvBuffer == nil {
			t.rcvBuffer = bytes.NewBuffer(hbuf)
		} else {
			t.rcvBuffer.Write(hbuf)
			return t.Receive()
		}

	}

	return nil, nil
}

func (t *Conn) readHeader() (MessageHeader, error) {
	res := MessageHeader{}

	hbuf := make([]byte, MessageHeaderByteSize)
	n, err := t.conn.Read(hbuf)
	if err != nil {
		return res, err
	}

	if n != MessageHeaderByteSize {
		return res, fmt.Errorf("Invalid header size")
	}

	err = res.UnmarshalBinary(hbuf)
	if err != nil {
		return res, err
	}
	return res, nil

}

func (t *Conn) sendHello(endpoint string) error {
	hello := HelloMessage{
		Version:           t.version(),
		ReceiveBufferSize: t.receiveBufferSize(),
		SendBufferSize:    t.sendBufferSize(),
		MaxMessageSize:    t.maxMessageSize(),
		MaxChunkCount:     t.maxChunkCount(),
		EndpointURL:       endpoint,
	}

	data, err := hello.bytes()
	if err != nil {
		return err
	}
	t.Send(transport.MessageTypeHello, bytes.NewReader(data), 0)
	return nil
}

func (t *Conn) receiveAck() error {

	header, err := t.readHeader()
	if err != nil {
		return err
	}
	if header.ChunkType != ChunkTypeFinal {
		return fmt.Errorf("invalid chunk type")
	}

	if header.Type != transport.MessageTypeAck {
		return fmt.Errorf("invalid message type type")
	}

	if header.Size > t.receiveBufferSize() {
		return fmt.Errorf("invalid message size (size > receiveBufferSize")
	}

	databuf := make([]byte, header.Size)
	_, err = t.conn.Read(databuf)
	if err != nil {
		return err
	}

	var ack AckResponse
	dec := binary.NewDecoder(bytes.NewBuffer(databuf))
	err = dec.Decode(&ack)
	if err != nil {
		return err
	}

	// check the Receive and Send BufferSizes
	// The values should not be larger than what the Client requested in the Hello Message
	// The values should be greater or equal than 8 192 bytes (see 1.03 Errata)
	if ack.ReceiveBufferSize > t.ReceiveBufferSize || ack.ReceiveBufferSize < 8192 {
		//OpcUa_BadConnectionRejected
	}
	t.ReceiveBufferSize = ack.ReceiveBufferSize

	if ack.SendBufferSize > t.SendBufferSize || ack.SendBufferSize < 8192 {
		//OpcUa_BadConnectionRejected
	}
	t.SendBufferSize = ack.SendBufferSize

	// Check the MaxMessageSize
	// If the size received from the server is != 0 we can accept smaller message sizes
	if ack.MaxMessageSize != 0 && (t.MaxMessageSize == 0 || t.MaxMessageSize > ack.MaxMessageSize) {
		// accept smaller messages
		t.MaxMessageSize = ack.MaxMessageSize
	}

	// Check the MaxChunkCount
	// If the size received from the server is != 0 we can accept smaller chunk counts
	if ack.MaxChunkCount != 0 && (t.MaxChunkCount == 0 || t.MaxChunkCount > ack.MaxChunkCount) {
		// accept less chunks
		t.MaxChunkCount = ack.MaxChunkCount
	}
	return nil
}

/* --- Methods to provide default values --- */

func (t *Conn) receiveBufferSize() uint32 {
	if t.ReceiveBufferSize > 0 {
		return t.ReceiveBufferSize
	}
	return DefaultReceiveBufferSize
}

func (t *Conn) sendBufferSize() uint32 {
	if t.SendBufferSize > 0 {
		return t.SendBufferSize
	}
	return DefaultSendBufferSize
}

func (t *Conn) maxMessageSize() uint32 {
	if t.MaxMessageSize > 0 {
		return t.MaxMessageSize
	}
	return DefaultMaxMessageSize
}

func (t *Conn) maxChunkCount() uint32 {
	if t.MaxChunkCount > 0 {
		return t.MaxChunkCount
	}
	return DefaultMaxChunkCount
}

func (t *Conn) version() uint32 {
	return DefaultVersion
}

func (t *Conn) dialTimeout() time.Duration {
	if t.DialTimeout > 0 {
		return t.DialTimeout
	}
	return DefaultDialTimeout
}
