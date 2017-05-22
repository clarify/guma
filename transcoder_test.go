package guma_test

import (
	"bytes"
	"testing"

	"github.com/searis/guma"
	"github.com/stretchr/testify/assert"
)

// Sub-test selection for TranscoderTest. Multiple tests can be joined by
// logical OR (|).
const (
	TestEncode = 1 << iota
	TestDecode
)

type TranscoderTest struct {
	SubTests     uint8
	Name         string
	Marshaled    []byte
	Unmarshaled  interface{}
	DecodeTarget interface{}
	EncodeError  string
	DecodeError  string
}

func (tt *TranscoderTest) Run(t *testing.T) {
	if (tt.SubTests & (TestEncode | TestDecode)) == 0 {
		panic("TranscoderTest must have at least one SubTest set")
	}
	if (tt.SubTests & TestEncode) != 0 {
		tt.testEncode(t)
	}
	// TODO: Run decoder test when TestDecode is set.
}

func (tt *TranscoderTest) testEncode(t *testing.T) {
	guma.SetDebugLogger(testLogger{t})

	t.Run(tt.Name+"/Encode", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		enc := guma.NewBinaryEncoder(&buf)
		err := enc.Encode(tt.Unmarshaled)
		if tt.EncodeError != "" {
			assert.EqualError(t, err, tt.EncodeError, "enc.Encode(tt.Unmarshaled)")
		} else {
			assert.NoError(t, err, "enc.Encode(tt.Unmarshaled)")
			assert.Equal(t, tt.Marshaled, buf.Bytes(), " buf.Bytes() != tt.Marshaled")
		}
	})
}
