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
			Name:     `"HELLO MESSAGE"`,
			Unmarshaled: tcp.MessageHeader{
				Type:            transport.MessageTypeHello,
				ChunkType:       tcp.ChunkTypeFinal,
				Size:            58,
				SecureChannelID: 01,
			},
			DecodeTarget: new(tcp.MessageHeader),
			Marshaled: []byte{
				0x48, 0x45, 0x4c, // HEL
				0x46,                   // ChunkTypeFinal (F)
				0x3a, 0x00, 0x00, 0x00, // Size: 58
				0x01, 0x00, 0x00, 0x00, // SecureChannelID: 01
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
			Marshaled: []byte{
				'H',
				'E',
				'L',
			},
		},
	}

	for i := range cases {
		cases[i].Run(t)
	}
}
