package guma

import (
	"bytes"
	"encoding/binary"
)

type byteMarshaler struct {
	data       interface{}
	cutN, cutM uint
	buf        bytes.Buffer
}

// SetData sets the data source that will be used for encoding binary.
func (e *byteMarshaler) SetData(data interface{}) {
	e.data = data
	e.cutN, e.cutM = 0, 0
}

// SetSlice sets a slice to apply to e's marshaled binary output. If m is zero,
// a slice of [n:] will be applied.
func (e *byteMarshaler) SetSlice(n, m uint) {
	e.cutN, e.cutM = n, m
}

// MarshalBinary encodes the previously set data source as little endian binary
// content. If a slice has been set, only the n:m bytes will be returned.
func (e *byteMarshaler) MarshalBinary() ([]byte, error) {
	e.buf.Reset()
	err := binary.Write(&e.buf, binary.LittleEndian, e.data)
	if err != nil {
		return nil, err
	}
	bytes := e.buf.Bytes()
	if e.cutM > 0 {
		bytes = bytes[e.cutN:e.cutM]
	} else if e.cutN > 0 {
		bytes = bytes[e.cutN:]
	}
	return bytes, nil
}

type byteUnmarshaler struct {
	Target interface{}
}

// UnmarshalBinary decodes little endian binary date from data into d's target.
func (d byteUnmarshaler) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	return binary.Read(buf, binary.LittleEndian, d.Target)
}
