package stack

import "github.com/searis/guma/stack/transport"

// A Client is an OPCUA client.
type Client struct {
	Channel transport.Channel
}
