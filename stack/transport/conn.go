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

// Conn is a OPCUA specific request/reponse driven interface.
type Conn interface {
	// Send transmits a request via the connection. If a message cannot be
	// successfully sent over the connection, Send returns an error.
	Send(r *Request) (*Response, error)

	// Close terminates the connection after performing necessary clean up.
	Close() error
}
