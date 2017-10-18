package binary_test

import (
	"testing"

	"github.com/searis/guma/internal/testutil"
	"github.com/searis/guma/stack/uatype"
)

func TestBitTranscoder(t *testing.T) {
	type eightBits struct{ Bit0, Bit1, Bit2, Bit3, Bit4, Bit5, Bit6, Bit7 uatype.Bit }
	type sixOneOne struct {
		Data       byte `opcua:"bits=6"`
		Bit6, Bit7 uatype.Bit
	}
	type oneSixOne struct {
		Bit0 uatype.Bit
		Data byte `opcua:"bits=6"`
		Bit7 uatype.Bit
	}
	type oneOneSix struct {
		Bit0, Bit1 uatype.Bit
		Data       byte `opcua:"bits=6"`
	}
	type sixTwo struct {
		Data0 byte `opcua:"bits=6"`
		Data1 byte `opcua:"bits=2"`
	}
	type sixTwoSixTwo struct {
		Data0 byte `opcua:"bits=6"`
		Data1 byte `opcua:"bits=2"`
		Data2 byte `opcua:"bits=6"`
		Data3 byte `opcua:"bits=2"`
	}
	type bitOverflow struct {
		Bit0, Bit1, Bit2 uatype.Bit
		Data1            byte `opcua:"bits=6"` // overflows by 1 here.
		Data2            byte `opcua:"bits=6"` // Should realign at cursor 0.
		Data3            byte `opcua:"bits=2"`
	}

	type invalidByte struct {
		Data byte `opcua:"bits=0"`
	}

	// Define aliases to faster set a sequence of bits.
	const I = uatype.Bit(true)
	const O = uatype.Bit(false)

	cases := []testutil.TranscoderTest{
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "eightBit{O,I,O,I,O,O,I,I}",
			Unmarshaled:  eightBits{O, I, O, I, O, O, I, I},
			DecodeTarget: new(eightBits),
			Marshaled:    []byte{0xCA},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "eightBit{O,O,I,I,O,I,O,I}",
			Unmarshaled:  eightBits{O, O, I, I, O, I, O, I},
			DecodeTarget: new(eightBits),
			Marshaled:    []byte{0xAC},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "eightBit{O,I,O,O,O,O,I,O}",
			Unmarshaled:  eightBits{O, I, O, O, O, O, I, O},
			DecodeTarget: new(eightBits),
			Marshaled:    []byte{0x42},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "eightBit{O,I,I,O,I,O,O,I}",
			Unmarshaled:  eightBits{O, I, I, O, I, O, O, I},
			DecodeTarget: new(eightBits),
			Marshaled:    []byte{0x96},
		},
		{
			SubTests:    testutil.TestEncode,
			Name:        "sixOneOne{0x40,false,false}",
			Unmarshaled: sixOneOne{0x40, O, O},
			EncodeError: "EncoderError sixOneOne.Data: can't encode 0x40 > 0x3F into 6 bits",
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "sixOneOne{0x3F,false,false}",
			Unmarshaled:  sixOneOne{0x3F, O, O},
			DecodeTarget: new(sixOneOne),
			Marshaled:    []byte{0x3F},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "oneSixOne{false,0x3F,false}",
			Unmarshaled:  oneSixOne{O, 0x3F, O},
			DecodeTarget: new(oneSixOne),
			Marshaled:    []byte{0x7E},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "oneSixOne{true,0x3F,false}",
			Unmarshaled:  oneSixOne{I, 0x3F, O},
			DecodeTarget: new(oneSixOne),
			Marshaled:    []byte{0x7F},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "oneSixOne{true,0x3F,true}",
			Unmarshaled:  oneSixOne{I, 0x3F, I},
			DecodeTarget: new(oneSixOne),
			Marshaled:    []byte{0xFF},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "oneOneSix{false,false,0x3F}",
			Unmarshaled:  oneOneSix{O, O, 0x3F},
			DecodeTarget: new(oneOneSix),
			Marshaled:    []byte{0xFC},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "sixTwo{0x3F,0x03}",
			Unmarshaled:  sixTwo{0x3F, 0x03},
			DecodeTarget: new(sixTwo),
			Marshaled:    []byte{0xFF},
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "sixTwo{0x01,0x01}",
			Unmarshaled:  sixTwo{0x01, 0x01},
			DecodeTarget: new(sixTwo),
			Marshaled:    []byte{0x41},
		},
		{

			SubTests:    testutil.TestEncode,
			Name:        "sixTwo{0x3F,0x03}",
			Unmarshaled: sixTwo{0x3F, 0x04},
			EncodeError: "EncoderError sixTwo.Data1: can't encode 0x04 > 0x03 into 2 bits",
		},
		{
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "sixTwoSixTwo{0x3F,0x01,0x00,0x3}",
			Unmarshaled:  sixTwoSixTwo{0x3F, 0x00, 0x00, 0x03},
			DecodeTarget: new(sixTwoSixTwo),
			Marshaled:    []byte{0x3F, 0xC0},
		},
		{
			SubTests:    testutil.TestEncode,
			Name:        "bitOverflow{0x3F,0x01,0x00,0x3}",
			Unmarshaled: bitOverflow{O, I, I, 0x3F, 0x00, 0x02},
			EncodeError: "EncoderError bitOverflow.Data1: bitCacheMarshaler: MarshalBinary: cursor position 9 > 8",
		},
		{
			// Documenting current behavior: we can't distinguish between bit
			// size 0, and a byte with no bit size tag.
			SubTests:     testutil.TestEncode | testutil.TestDecode,
			Name:         "invalidByte{0x01}",
			Unmarshaled:  invalidByte{0x01},
			DecodeTarget: new(invalidByte),
			Marshaled:    []byte{0x01},
		},
	}
	for i := range cases {
		cases[i].Run(t)
	}
}
