package guma

import (
	"bytes"
	"encoding/binary"
)

// charArray handles transcoding betwen an OPC UA CharArray and a go string.
type charArray string

// MarshalBinary encodes s into a OPC UA CharArray.
func (s charArray) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	l := int32(len(s))
	if l == 0 {
		l = -1
	}

	if err := binary.Write(&buf, binary.LittleEndian, l); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, ([]rune)(string(s))); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BitLength returns the size in bits of s when encoded to binary. The number
// is always a multiplum of 32.
func (s charArray) BitLength() int {
	return (len(s) + 1) * 32
}
