package transport

import (
	"io"

	"github.com/searis/guma/stack/uatype"
)

// Request represents an OPCUA request received by a server or to be sent by a
// client.
type Request struct {
	NodeID uatype.NodeId
	Body   io.Reader
}

// Response represents an OPCUA response sent by a server or received by a
// client.
type Response struct {
	NodeID uatype.NodeId
	Body   io.Reader
}

// Channel is a OPCUA specific request/reponse driven interface.
type Channel interface {
	// Send transmits a request via the channel. If a message cannot be
	// successfully sent over the channel, Send returns an error.
	Send(r *Request) (*Response, error)

	// Close closes the channel after performing necessary clean up.
	// Underlying sockets must still be closed by provider
	Close() error
}

// Conn is a OPCUA transport layer interface.
type Conn interface {
	Send(mt MessageType, r io.Reader, secChanID uint32) error
	Receive() (io.Reader, error)
}
