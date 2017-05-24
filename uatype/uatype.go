//go:generate opcua-xml2code -o uatype_auto.go -td ../schemas/1.03/Opc.Ua.Types.bsd.xml
//go:generate goimports -w uatype_auto.go

// Package uatype provides mostly auto-generated types used for marshalling and
// unmarshalling of objects for the OPC UA v1.03 binary protocol.
package uatype

import (
	"encoding/binary"
	"errors"
)

// Errors raised by BinaryUnmarshaler implementations.
var (
	ErrNotEnoughData = errors.New("not enough data")
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
	size := int32(len(bs))
	target := make([]byte, size+4)

	// At least initally, we don't distinguish between null values and arrays
	// with length null. We could change this if it's required.
	if size == 0 {
		size = -1
		binary.LittleEndian.PutUint32(target, uint32(size))
		return target, nil
	}

	// Encode string.
	binary.LittleEndian.PutUint32(target, uint32(size))
	copy(target[4:], bs)

	return target, nil
}

// UnmarshalBinary reads from the head of data and sets bs. If there is not
// enough data available, the ErrNotEnoughData error is returned.
func (bs *ByteString) UnmarshalBinary(data []byte) error {
	l := len(data)
	if l < 4 {
		return ErrNotEnoughData
	}
	size := int32(binary.LittleEndian.Uint32(data[0:4]))

	if size == -1 || size == 0 {
		// At least initially, we don't distinguish between null values and
		// arrays with length 0. We could change this if it's required.
		bs = nil
		return nil
	}

	stop := int(size) + 4
	if stop < l {
		return ErrNotEnoughData
	}

	*bs = make([]byte, size)
	copy(*bs, data[4:stop])
	return nil
}

// BitLength returns the size in bits of bs when encoded to binary. The number
// is at least 32, and always a multiplum of 8.
func (bs ByteString) BitLength() int {
	return 32 + 8*len(bs)
}
