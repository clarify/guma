package binary_test

import (
	"testing"

	"github.com/searis/guma/internal/testutil"
	"github.com/searis/guma/stack/uatype"
)

func TestByteString(t *testing.T) {
	type oneByteStringStruct struct {
		Data0 uatype.ByteString
	}
	cases := []testutil.TranscoderTest{
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         `oneByteStringStruct{ByteString("foobar")}`,
			Unmarshaled:  oneByteStringStruct{uatype.ByteString{'f', 'o', 'o', 'b', 'a', 'r'}},
			DecodeTarget: new(oneByteStringStruct),
			Marshaled: []byte{
				6, 0, 0, 0,
				'f',
				'o',
				'o',
				'b',
				'a',
				'r',
			},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         `ByteString(nil)`,
			Unmarshaled:  uatype.ByteString(nil),
			DecodeTarget: new(uatype.ByteString),
			Marshaled: []byte{
				0xFF, 0xFF, 0xFF, 0xFF,
			},
		},
		{
			// At the moment we don't distinguish between null and empty values
			// for ByteString, so testing Encode only.
			SubTests:     testutil.TestEncode,
			Name:         `ByteString("")`,
			Unmarshaled:  uatype.ByteString(""),
			DecodeTarget: new(uatype.ByteString),
			Marshaled: []byte{
				0xFF, 0xFF, 0xFF, 0xFF,
			},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         `ByteString("foobar")`,
			Unmarshaled:  uatype.ByteString{'f', 'o', 'o', 'b', 'a', 'r'},
			DecodeTarget: new(uatype.ByteString),
			Marshaled: []byte{
				6, 0, 0, 0,
				'f',
				'o',
				'o',
				'b',
				'a',
				'r',
			},
		},
	}

	for i := range cases {
		cases[i].Run(t)
	}
}
