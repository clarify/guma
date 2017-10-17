package binary_test

import (
	"testing"
)

func TestNumbers(t *testing.T) {
	type allNumbers struct {
		I8  int8
		U8  uint8
		I16 int16
		U16 uint16
		I32 int32
		U32 uint32
		F32 float32
		I64 int64
		U64 uint64
		F64 float64
	}

	cases := []TranscoderTest{
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "uint8(0xFF)",
			Unmarshaled:  uint8(0xFF),
			DecodeTarget: new(uint8),
			Marshaled:    []byte{0xFF},
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
			Name:         "int16(0x0042)",
			Unmarshaled:  int16(0x0042),
			DecodeTarget: new(int16),
			Marshaled:    []byte{0x42, 0x00},
		},
		{
			SubTests:     TestDecode,
			Name:         "ErrNotEnoughData",
			DecodeTarget: new(int16),
			DecodeError:  "not enough data",
			Marshaled:    []byte{0x42},
		},
		{
			SubTests: TestEncode | TestDecode,
			Name:     "allNumbers",
			Unmarshaled: allNumbers{
				I8:  -0x12,
				U8:  0x12,
				I16: -0x1234,
				U16: 0x1234,
				I32: -0x12345678,
				U32: 0x12345678,
				F32: 3.14,
				I64: -0x1234567801020304,
				U64: 0x1234567801020304,
				F64: -3.14,
			},
			DecodeTarget: new(allNumbers),
			Marshaled: []byte{
				0xEE,
				0x12,
				0xCC, 0xED,
				0x34, 0x12,
				0x88, 0xA9, 0xCB, 0xED,
				0x78, 0x56, 0x34, 0x12,
				0xC3, 0xF5, 0x48, 0x40,
				0xFC, 0xFC, 0xFD, 0xFE, 0x87, 0xA9, 0xCB, 0xED,
				0x04, 0x03, 0x02, 0x01, 0x78, 0x56, 0x34, 0x12,
				0x1F, 0x85, 0xEB, 0x51, 0xB8, 0x1E, 0x09, 0xC0,
			},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "[]int8{1,-2,3}",
			Unmarshaled:  []int8{1, -2, 3},
			DecodeTarget: newInt8Slice(3),
			Marshaled: []byte{
				0x01,
				0xFE,
				0x03,
			},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "[]int16{1,-2,3}",
			Unmarshaled:  []int16{1, -2, 3},
			DecodeTarget: newInt16Slice(3),
			Marshaled: []byte{
				0x01, 0x00,
				0xFE, 0xFF,
				0x03, 0x00,
			},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         "[]int32{1,-2,3}",
			Unmarshaled:  []int32{1, -2, 3},
			DecodeTarget: newInt32Slice(3),
			Marshaled: []byte{
				0x01, 0x00, 0x00, 0x00,
				0xFE, 0xFF, 0xFF, 0xFF,
				0x03, 0x00, 0x00, 0x00,
			},
		},
	}

	for i := range cases {
		cases[i].Run(t)
	}
}

func newInt8Slice(len int) *[]int8 {
	l := make([]int8, len)
	return &l
}

func newInt16Slice(len int) *[]int16 {
	l := make([]int16, len)
	return &l
}

func newInt32Slice(len int) *[]int32 {
	l := make([]int32, len)
	return &l
}
