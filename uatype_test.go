package guma_test

import (
	"testing"

	"github.com/searis/guma/uatype"
)

func TestString(t *testing.T) {
	type oneStringStruct struct {
		Data0 uatype.String
	}
	cases := []TranscoderTest{
		{
			SubTests:     TestEncode | TestDecode,
			Name:         `String("")`,
			Unmarshaled:  uatype.String(""),
			DecodeTarget: new(uatype.String),
			Marshaled: []byte{
				0, 0, 0, 0,
			},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         `String("foobar")`,
			Unmarshaled:  uatype.String("foobar"),
			DecodeTarget: new(uatype.String),
			Marshaled: []byte{
				'f', 0, 0, 0,
				'o', 0, 0, 0,
				'o', 0, 0, 0,
				'b', 0, 0, 0,
				'a', 0, 0, 0,
				'r', 0, 0, 0,
				0, 0, 0, 0,
			},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         `oneStringStruct{String("foobar")}`,
			Unmarshaled:  oneStringStruct{uatype.String("foobar")},
			DecodeTarget: new(oneStringStruct),
			Marshaled: []byte{
				'f', 0, 0, 0,
				'o', 0, 0, 0,
				'o', 0, 0, 0,
				'b', 0, 0, 0,
				'a', 0, 0, 0,
				'r', 0, 0, 0,
				0, 0, 0, 0,
			},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         `ByteString("")`,
			Unmarshaled:  uatype.ByteString(""),
			DecodeTarget: new(uatype.ByteString),
			Marshaled: []byte{
				0xFF, 0xFF, 0xFF, 0xFF,
			},
		},
		{
			SubTests:    TestEncode | TestDecode,
			Name:        `ByteString("foobar")`,
			Unmarshaled: uatype.ByteString{'f', 'o', 'o', 'b', 'a', 'r'},
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
