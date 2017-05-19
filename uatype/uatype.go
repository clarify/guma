//go:generate opcua-xml2code -o uatype_auto.go -td ../schemas/1.03/Opc.Ua.Types.bsd.xml
//go:generate goimports -w uatype_auto.go

// Package uatype provides mostly auto-generated types used for marshalling and
// unmarshalling of objects for the OPC UA v1.03 binary protocol.
package uatype

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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

func (id Guid) MarshalBinary() ([]byte, error) {
	return id[:], nil
}

func (id Guid) UnmarshalBinary(data []byte) error {
	if len(data) < 16 {
		return fmt.Errorf("expected 16 or more bytes, got %d", len(data))
	}
	copy(id[:], data[:])
	return nil
}

// FIXME: add methods if needed.
type ByteString []byte

// StringNotTerminated is returned by String.UnmarshalBinary if the passed in
// data does not contain a 32bit null character.
var ErrStringNotTerminated = errors.New("string not terminated")

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

// UnmarshalBinary extracts bytes from data until a UTF-8 null character is
// found. If no null-character is found, ErrStringNotTerminated is returned.
func (s *String) UnmarshalBinary(data []byte) error {
	length := bytes.IndexAny(data, string(0))
	if length < 0 {
		return ErrStringNotTerminated
	}
	read := make([]byte, length)
	copy(read, data[:length])
	*s = String(read)
	return nil
}

// BitLength returns the size in bits of s when encoded to binary. The number
// is always a multiplum of 32.
func (s String) BitLength() int {
	return (len(s) + 1) * 32
}
