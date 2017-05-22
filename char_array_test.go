package guma_test

import "testing"

func TestCharArray(t *testing.T) {
	type oneStringStruct struct {
		Data0 string
	}
	cases := []TranscoderTest{
		{
			SubTests:    TestEncode | TestDecode,
			Name:        `""`,
			Unmarshaled: "",
			Marshaled: []byte{
				0xFF, 0xFF, 0xFF, 0xFF, // -1
			},
		},
		{
			SubTests:    TestEncode | TestDecode,
			Name:        `"foobar"`,
			Unmarshaled: "foobar",
			Marshaled: []byte{
				6, 0, 0, 0,
				'f', 0, 0, 0,
				'o', 0, 0, 0,
				'o', 0, 0, 0,
				'b', 0, 0, 0,
				'a', 0, 0, 0,
				'r', 0, 0, 0,
			},
		},
		{
			SubTests:    TestEncode | TestDecode,
			Name:        `oneStringStruct{"foobar"}`,
			Unmarshaled: oneStringStruct{"foobar"},
			Marshaled: []byte{
				6, 0, 0, 0,
				'f', 0, 0, 0,
				'o', 0, 0, 0,
				'o', 0, 0, 0,
				'b', 0, 0, 0,
				'a', 0, 0, 0,
				'r', 0, 0, 0,
			},
		},
	}

	for i := range cases {
		cases[i].Run(t)
	}
}
