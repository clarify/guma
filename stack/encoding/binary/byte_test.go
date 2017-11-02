package binary_test

import (
	"testing"

	"github.com/searis/guma/internal/testutil"
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

	cases := []testutil.TranscoderTest{
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "true",
			Unmarshaled:  bool(true),
			DecodeTarget: new(bool),
			Marshaled:    []byte{0x01},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "false",
			Unmarshaled:  bool(false),
			DecodeTarget: new(bool),
			Marshaled:    []byte{0x00},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "uint8(0xCA)",
			Unmarshaled:  uint8(0xCA),
			DecodeTarget: new(uint8),
			Marshaled:    []byte{0xCA},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "int8(-1)",
			Unmarshaled:  int8(-1),
			DecodeTarget: new(int8),
			Marshaled:    []byte{0xFF},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "uint16(0xCAFE)",
			Unmarshaled:  uint16(0xCAFE),
			DecodeTarget: new(uint16),
			Marshaled:    []byte{0xFE, 0xCA},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "int16(-1)",
			Unmarshaled:  int16(-1),
			DecodeTarget: new(int16),
			Marshaled:    []byte{0xFF, 0xFF},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "uint32(0xCAFE1337)",
			Unmarshaled:  uint32(0xCAFE1337),
			DecodeTarget: new(uint32),
			Marshaled:    []byte{0x37, 0x13, 0xFE, 0xCA},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "int32(-1)",
			Unmarshaled:  int32(-1),
			DecodeTarget: new(int32),
			Marshaled:    []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "uint64(0x01000000CAFE1337)",
			Unmarshaled:  uint64(0x01000000CAFE1337),
			DecodeTarget: new(uint64),
			Marshaled:    []byte{0x37, 0x13, 0xFE, 0xCA, 0x00, 0x00, 0x00, 0x01},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "int64(-1)",
			Unmarshaled:  int64(-1),
			DecodeTarget: new(int64),
			Marshaled:    []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "[]byte{0xF0,0x0C}",
			Unmarshaled:  []byte{0xF0, 0x0C},
			DecodeTarget: &[]byte{0x00, 0x00},
			Marshaled:    []byte{0xF0, 0x0C},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "[2]byte{0xF0,0x0C}",
			Unmarshaled:  [2]byte{0xF0, 0x0C},
			DecodeTarget: &[2]byte{0x00, 0x00},
			Marshaled:    []byte{0xF0, 0x0C},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "[]int8{0xF0,0x0C}",
			Unmarshaled:  []int8{-16, 0x0C},
			DecodeTarget: &[]int8{0x00, 0x00},
			Marshaled:    []byte{0xF0, 0x0C},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "[2]int8{-16,0x0C}",
			Unmarshaled:  [2]int8{-16, 0x0C},
			DecodeTarget: &[2]int8{0x00, 0x00},
			Marshaled:    []byte{0xF0, 0x0C},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "[]uint16{0xCAFE,0x1337}",
			Unmarshaled:  []uint16{0xCAFE, 0x1337},
			DecodeTarget: &[]uint16{0xCAFE, 0x1337},
			Marshaled:    []byte{0xFE, 0xCA, 0x37, 0x13},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "[2]uint16{0xCADE,0x1337}",
			Unmarshaled:  [2]uint16{0xCAFE, 0x1337},
			DecodeTarget: &[2]uint16{0xCAFE, 0x1337},
			Marshaled:    []byte{0xFE, 0xCA, 0x37, 0x13},
		},
		{
			// Documenting current behavior where non-aligned bit-lengths >8 are
			// not supported. In this sexample, for struct-field tags.
			SubTests:    testutil.TestEncode,
			Name:        "invalidInt32{0x01FF}",
			Unmarshaled: invalidInt32{0x01FF},
			EncodeError: "EncoderError invalidInt32.Data: bit length not in range 1-8",
		},
		{
			// Documenting current behavior where >8 bit-lengths are not
			// supported for struct field tags.
			SubTests:    testutil.TestEncode,
			Name:        "shrinkUint32{0x0000FFFF}",
			Unmarshaled: shrinkUint32{0x0000FFFF},
			EncodeError: "EncoderError shrinkUint32.Data: bit length not in range 1-8",
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "signedInts{0x13,0x37,0xCA,0xFE,{0x07,0x11},{0x13, 0x37}}",
			Unmarshaled:  signedInts{0x13, 0x37, 0xCA, 0xFE, [2]byte{0x07, 0x11}, [2]int32{0x13, 0x37}},
			DecodeTarget: new(signedInts),
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
