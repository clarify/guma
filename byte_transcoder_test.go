package guma_test

import (
	"testing"
)

func TestByteTranscoder(t *testing.T) {
	type invalidInt32 struct {
		Data int32 `opcua:"bits=9"`
	}
	type shrinkUint32 struct {
		Data uint32 `opcua:"bits=16"`
	}
	type signedInts struct {
		Data0 int8
		Data1 int16
		Data2 int32
		Data3 int64
		Data4 [2]byte
		Data5 [2]int32
	}

	cases := []TranscoderTest{
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "true",
			Unmarshaled:  bool(true),
			DecodeTarget: new(bool),
			Marshaled:    []byte{0x01},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "false",
			Unmarshaled:  bool(false),
			DecodeTarget: new(bool),
			Marshaled:    []byte{0x00},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "uint8(0xCA)",
			Unmarshaled:  uint8(0xCA),
			DecodeTarget: new(uint8),
			Marshaled:    []byte{0xCA},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "int8(-1)",
			Unmarshaled:  int8(-1),
			DecodeTarget: new(int8),
			Marshaled:    []byte{0xFF},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "uint16(0xCAFE)",
			Unmarshaled:  uint16(0xCAFE),
			DecodeTarget: new(uint16),
			Marshaled:    []byte{0xFE, 0xCA},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "int16(-1)",
			Unmarshaled:  int16(-1),
			DecodeTarget: new(int16),
			Marshaled:    []byte{0xFF, 0xFF},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "uint32(0xCAFE1337)",
			Unmarshaled:  uint32(0xCAFE1337),
			DecodeTarget: new(uint32),
			Marshaled:    []byte{0x37, 0x13, 0xFE, 0xCA},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "int32(-1)",
			Unmarshaled:  int32(-1),
			DecodeTarget: new(int32),
			Marshaled:    []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "uint64(0x01000000CAFE1337)",
			Unmarshaled:  uint64(0x01000000CAFE1337),
			DecodeTarget: new(uint64),
			Marshaled:    []byte{0x37, 0x13, 0xFE, 0xCA, 0x00, 0x00, 0x00, 0x01},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "int64(-1)",
			Unmarshaled:  int64(-1),
			DecodeTarget: new(int64),
			Marshaled:    []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			// Documenting current behavior: We raise an ugly panic error for
			// when using a non-aligned bit-size as a struct-tag.
			SubTests:    TestEncode,
			Name:        "invalidInt32{0x01FF}",
			Unmarshaled: invalidInt32{0x01FF},
			EncodeError: "recovered from panic: 'interface conversion: interface {} is int32, not uint8'",
		},
		{
			// Documenting current behavior: We don't allow integer
			// shrinking/growing based on struct tags atm.
			SubTests:    TestEncode,
			Name:        "shrinkUint32{0x0000FFFF}",
			Unmarshaled: shrinkUint32{0x0000FFFF},
			EncodeError: "recovered from panic: 'interface conversion: interface {} is uint32, not uint8'",
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "signedInts{0x13,0x37,0xCA,0xFE,{0x07,0x11},{0x13, 0x37}}",
			Unmarshaled:  signedInts{0x13, 0x37, 0xCA, 0xFE, [2]byte{0x07, 0x11}, [2]int32{0x13, 0x37}},
			DecodeTarget: &signedInts{},
			Marshaled: []byte{
				0x13,
				0x37, 0,
				0xCA, 0, 0, 0,
				0xFE, 0, 0, 0, 0, 0, 0, 0,
				0x07, 0x11,
				0x13, 0, 0, 0, 0x37, 0, 0, 0,
			},
		},
	}

	for i := range cases {
		cases[i].Run(t)
	}
}
