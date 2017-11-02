package transport

import (
	"io"
	"time"

	"github.com/searis/guma/stack/uatype"
)

// Request should contain a matching NodeID and encoded Body, and is used to
// communicate against a server.
type Request struct {
	NodeID uatype.ExpandedNodeId
	Body   io.Reader
}

// Response should contain a matching NodeID and encoded Body, and is returned
// from a server to a client.
type Response struct {
	NodeID uatype.ExpandedNodeId
	Body   io.Reader
}

// SecureChannel is an abstract interface for the (client-side) OPC UA Secure
// Conversation concept. Different implementations may be given on top of
// different types of connections, e.g. UACP v.s. HTTP.
type SecureChannel interface {
	// Send transmits a request via the channel. If a message cannot be
	// successfully sent over the channel, Send returns an error. To send
	// without a deadline, use the zero time.
	Send(req Request, deadline time.Time) (*Response, error)

	// Close closes the channel after performing necessary clean-up.
	Close() error
}
