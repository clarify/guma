package binary_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/searis/guma/stack/encoding/binary"
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
		t.Fatalf("TranscoderTest %s/%s/Encode: at least on SubTest must be set", t.Name(), tt.Name)
	}
	if (tt.SubTests & TestEncode) != 0 {
		tt.testEncode(t)
	}
	if (tt.SubTests & TestDecode) != 0 {
		tt.testDecode(t)
	}
}

func (tt *TranscoderTest) testEncode(t *testing.T) {
	binary.SetDebugLogger(testLogger{t})

	t.Run(tt.Name+"/Encode", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		enc := binary.NewEncoder(&buf)
		err := enc.Encode(tt.Unmarshaled)
		if tt.EncodeError != "" {
			assert.EqualError(t, err, tt.EncodeError, "enc.Encode(tt.Unmarshaled)")
			assert.Nil(t, buf.Bytes(), " buf.Bytes() != nil")
		} else {
			assert.NoError(t, err, "enc.Encode(tt.Unmarshaled)")
			assert.Equal(t, tt.Marshaled, buf.Bytes(), " buf.Bytes() != tt.Marshaled")
		}
	})
}

func (tt *TranscoderTest) testDecode(t *testing.T) {
	t.Run(tt.Name+"/Decode", func(t *testing.T) {
		t.Parallel()
		buf := bytes.NewBuffer(tt.Marshaled)
		dec := binary.NewDecoder(buf)
		err := dec.Decode(tt.DecodeTarget)
		if tt.DecodeError != "" {
			assert.EqualError(t, err, tt.DecodeError, "enc.Decode(NewBuffer(tt.Marshaled))")
		} else {
			assert.NoError(t, err, "enc.Decode(tt.Marshaled)")
			if !assert.NotNil(t, tt.DecodeTarget, "tt.DecodeTarget == nil") {
				return
			}
			r := reflect.ValueOf(tt.DecodeTarget).Elem().Interface()
			assert.Equal(t, tt.Unmarshaled, r, "*tt.DecodeTarget != tt.Unmarshaled")
		}
	})
}
