//go:generate opcua-xml2code -o uatype_auto.go -td ../schemas/1.03/Opc.Ua.Types.bsd.xml
//go:generate goimports -w uatype_auto.go

// Package uatype provides mostly auto-generated types used for marshalling and
// unmarshalling of objects for the OPC UA v1.03 binary protocol.
package uatype

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// Errors raised by BinaryUnmarshaler implementations.
var (
	ErrStringNotTerminated = errors.New("string not terminated")
	ErrNotEnoughData       = errors.New("not enough data")
)

// BitLengther is implemented by types that should be encoded into, or has been
// decoded from, a certain number of bits.
type BitLengther interface {
	BitLength() int
}

// Size returns the binary encoded size of b in whole bytes (rounded down).
func Size(b BitLengther) int {
	return b.BitLength() / 8
}

// A Bit can either be set (true) or not set (false).
type Bit bool

// BitLength returns the number of bits that should be used to encode and decode
// values.
func BitLength() int {
	return 1
}

// Guid represents a 16 byte unique ID.
type Guid [16]byte

// ByteString is encoded as a string of bytes prefixed by the length as int32.
// -1 is used to indicate a null string.
type ByteString []byte

// Marshal binary returns the binary representation of bs.
func (bs ByteString) MarshalBinary() ([]byte, error) {
	l := int32(len(bs))
	if l == 0 {
		l = -1
	}
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, l); err != nil {
		return nil, err
	}
	if l == -1 {
		return buf.Bytes(), nil
	}
	if err := binary.Write(&buf, binary.LittleEndian, bs); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BitLength returns the size in bits of bs when encoded to binary. The number
// is at least 32, and always a multiplum of 8.
func (bs ByteString) BitLength() int {
	return 32 + 8*len(bs)
}

// String is a null-terminated string of UTF-8 characters.
type String string

// MarshalBinary returns the binary representation of s.
func (s String) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, ([]rune)(string(s))); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, rune(0)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BitLength returns the size in bits of s when encoded to binary. The number
// is always a multiplum of 32.
func (s String) BitLength() int {
	return (len(s) + 1) * 32
}
