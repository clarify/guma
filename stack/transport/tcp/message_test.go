package tcp_test

import (
	"testing"

	"github.com/searis/guma/internal/testutil"
	"github.com/searis/guma/stack/transport/tcp"
)

func TestMessage(t *testing.T) {
	cases := []testutil.TranscoderTest{
		{
			SubTests: testutil.TestEncode | testutil.TestDecode,
			Name:     `"HELLO MESSAGE"`,
			Unmarshaled: tcp.HelloMessage{
				Version:           tcp.DefaultVersion,
				ReceiveBufferSize: tcp.DefaultReceiveBufferSize,
				SendBufferSize:    tcp.DefaultSendBufferSize,
				MaxMessageSize:    tcp.DefaultMaxMessageSize,
				MaxChunkCount:     tcp.DefaultMaxChunkCount,
				EndpointURL:       "opc.tcp://T15C-069E89:4840",
			},
			DecodeTarget: new(tcp.HelloMessage),
			Marshaled: []byte{
				0x00, 0x00, 0x00, 0x00, // Version 0
				0x00, 0x00, 0x0A, 0x00, // DefaultReceiveBufferSize: 1024 * 64 * 10
				0x00, 0x00, 0x0A, 0x00, // DefaultSendBufferSize: 1024 * 64 * 10
				0x00, 0x00, 0x00, 0x00, // DefaultMaxMessageSize: 0
				0x00, 0x00, 0x00, 0x00, // DefaultMaxChunkCount: 0
				0x1A, 0x00, 0x00, 0x00, // String length: 26
				0x6f, 0x70, /* op */
				0x63, 0x2e, 0x74, 0x63, 0x70, 0x3a, 0x2f, 0x2f, /* c.tcp:// */
				0x54, 0x31, 0x35, 0x43, 0x2d, 0x30, 0x36, 0x39, /* T15C-069 */
				0x45, 0x38, 0x39, 0x3a, 0x34, 0x38, 0x34, 0x30, /* E89:4840 */
			},
		},
	}

	for i := range cases {
		cases[i].Run(t)
	}
}
