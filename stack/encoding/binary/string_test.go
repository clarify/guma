package binary_test

import (
	"testing"
)

func TestString(t *testing.T) {
	type oneStringStruct struct {
		Data0 string
	}
	cases := []TranscoderTest{
		{
			SubTests:     TestEncode | TestDecode,
			Name:         `""`,
			Unmarshaled:  "",
			DecodeTarget: new(string),
			Marshaled: []byte{
				0xFF, 0xFF, 0xFF, 0xFF, // -1
			},
		},
		{
			SubTests:     TestDecode,
			Name:         "ErrNotEnoughData",
			DecodeTarget: new(string),
			DecodeError:  "not enough data",
			Marshaled:    []byte{0x42},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         `"A世界"`,
			Unmarshaled:  "A世界",
			DecodeTarget: new(string),
			Marshaled: []byte{
				7, 0, 0, 0,
				65,
				228,
				184,
				150,
				231,
				149,
				140,
			},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         `oneStringStruct{"foobar"}`,
			Unmarshaled:  oneStringStruct{"foobar"},
			DecodeTarget: new(oneStringStruct),
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
			SubTests:     TestEncode | TestDecode,
			Name:         `["","A世界","foobar"]`,
			Unmarshaled:  []string{"", "A世界", "foobar"},
			DecodeTarget: newStringSlice(3),
			Marshaled: []byte{
				0xFF, 0xFF, 0xFF, 0xFF, // size=-1:""
				0x07, 0x00, 0x00, 0x00, // size=7:"A世界" ->
				65,
				228,
				184,
				150,
				231,
				149,
				140,
				0x06, 0x00, 0x00, 0x00, // size=6:"foobar" ->
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

func newStringSlice(size int) *[]string {
	l := make([]string, size)
	return &l
}
