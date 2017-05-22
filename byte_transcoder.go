package guma

import (
	"bytes"
	"encoding/binary"
)

type byteTranscoder struct {
	data       interface{}
	cutN, cutM uint
	buf        bytes.Buffer
}

// SetData sets the data source (marshal) or destination (unmarshal). For the
// latter, the value should always be a pointer type. The method clears any
// previously set slice.
func (b *byteTranscoder) SetData(data interface{}) {
	b.data = data
	b.cutN, b.cutM = 0, 0
}

// SetSlice sets a slice to apply to binary output (marshal) or input
// (unmarshal). If m is zero, a slice of [n:] will be applied.
func (b *byteTranscoder) SetSlice(n, m uint) {
	b.cutN, b.cutM = n, m
}

// MarshalBinary encodes the previously set data source as little endian binary
// content. If a slice has been set, only the n:m bytes will be returned.
func (b *byteTranscoder) MarshalBinary() ([]byte, error) {
	b.buf.Reset()
	err := binary.Write(&b.buf, binary.LittleEndian, b.data)
	if err != nil {
		return nil, err
	}
	bytes := b.buf.Bytes()
	if b.cutM > 0 {
		bytes = bytes[b.cutN:b.cutM]
	} else if b.cutN > 0 {
		bytes = bytes[b.cutN:]
	}
	return bytes, nil
}
