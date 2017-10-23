package tcp_test

import (
	"testing"

	"github.com/searis/guma/internal/testutil"
	"github.com/searis/guma/stack/transport"
	"github.com/searis/guma/stack/transport/tcp"
)

func TestMessageHeader(t *testing.T) {
	cases := []testutil.TranscoderTest{
		{
			SubTests: testutil.TestEncode | testutil.TestDecode,
			Name:     `MessageHeader(Type:"HEL",ChunkType:"F",Size:58)`,
			Unmarshaled: tcp.MessageHeader{
				Type:      transport.MessageTypeHello,
				ChunkType: tcp.ChunkTypeFinal,
				Size:      58,
			},
			DecodeTarget: new(tcp.MessageHeader),
			Marshaled: []byte{
				0x48, 0x45, 0x4c, // MessageType: HEL (Hello)
				0x46,                   // ChunkType: F (Final)
				0x3a, 0x00, 0x00, 0x00, // Size: 58
			},
		},
	}

	for i := range cases {
		cases[i].Run(t)
	}
}
func TestMessageType(t *testing.T) {
	type oneStringStruct struct {
		Data0 string
	}
	cases := []testutil.TranscoderTest{
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         `"HEL"`,
			Unmarshaled:  transport.MessageTypeHello,
			DecodeTarget: new(transport.MessageType),
			Marshaled:    []byte{'H', 'E', 'L'},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         `"ACK"`,
			Unmarshaled:  transport.MessageTypeAck,
			DecodeTarget: new(transport.MessageType),
			Marshaled:    []byte{'A', 'C', 'K'},
		},
	}

	for i := range cases {
		cases[i].Run(t)
	}
}
