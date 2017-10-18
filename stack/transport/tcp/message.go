package tcp

import (
	"bytes"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/searis/guma/stack/encoding/binary"
)

var typeFmt = spew.ConfigState{
	Indent:   "\t",
	SortKeys: true,
}

// HelloMessage is the first message sent on the socket to create a connection
// It contains connection parameters supported by the client. These might be
// adjusted in the AckResponse
type HelloMessage struct {
	Version           uint32
	ReceiveBufferSize uint32
	SendBufferSize    uint32
	MaxMessageSize    uint32
	MaxChunkCount     uint32
	EndpointURL       string
}

// AckResponse is the response to a HelloMessage and contains the parameters that
// the server can accept.
type AckResponse struct {
	Version           uint32
	ReceiveBufferSize uint32
	SendBufferSize    uint32
	MaxMessageSize    uint32
	MaxChunkCount     uint32
}

func (h HelloMessage) bytes() ([]byte, error) {
	var buf bytes.Buffer
	enc := binary.NewEncoder(&buf)

	err := enc.Encode(h)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (a AckResponse) bytes() ([]byte, error) {
	var buf bytes.Buffer
	enc := binary.NewEncoder(&buf)

	err := enc.Encode(a)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (a AckResponse) String() string {
	return fmt.Sprintf(`
		---- AckResponse ----
		Version:		%d
		ReceiveBufferSize:	%d
		SendBufferSize:		%d
		MaxMessageSize:		%d
		MaxChunkCount		%d
		---- /AckResponse ----`, a.Version, a.ReceiveBufferSize, a.SendBufferSize, a.MaxMessageSize, a.MaxChunkCount)
}

func (h HelloMessage) String() string {
	return fmt.Sprintf(`
		---- HelloMessage ----
		Version:		%d
		ReceiveBufferSize:	%d
		SendBufferSize:		%d
		MaxMessageSize:		%d
		MaxChunkCount		%d
		---- /HelloMessage ----`, h.Version, h.ReceiveBufferSize, h.SendBufferSize, h.MaxMessageSize, h.MaxChunkCount)
}
