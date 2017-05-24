package guma_test

import "testing"

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
	}

	for i := range cases {
		cases[i].Run(t)
	}
}
