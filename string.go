package guma

import (
	"encoding/binary"

	"github.com/searis/guma/uatype"
)

// uaString handles encoding and decoding of an OPC UA String.
type uaString string

// MarshalBinary prefixes s with it's size in bytes as int32. A length of -1 is
// encoded for empty strings.
func (s uaString) MarshalBinary() ([]byte, error) {
	data := []byte(s)
	size := int32(len(data))
	target := make([]byte, size+4)

	// Encode empty string as null-string.
	if size == 0 {
		size = -1
		binary.LittleEndian.PutUint32(target, uint32(size))
		return target, nil
	}

	// Encode string.
	binary.LittleEndian.PutUint32(target, uint32(size))
	copy(target[4:], data)

	return target, nil
}

// UnmarshalBinary first reads the data size in bytes as int32, and then reads
// the string value according to the size into s.
func (s *uaString) UnmarshalBinary(data []byte) error {
	l := len(data)
	if l < 4 {
		// FIXME: sort out import dependency on reorg.
		return uatype.ErrNotEnoughData
	}
	size := int32(binary.LittleEndian.Uint32(data[0:4]))
	if size == -1 || size == 0 {
		*s = ""
		return nil
	}
	stop := int(size) + 4
	if stop < l {
		return uatype.ErrNotEnoughData
	}

	*s = uaString(data[4:stop])
	return nil
}

// BitLength returns the size in bits of s when encoded to binary. The number
// is at least 32, and always a multiplum of 8.
func (s uaString) BitLength() int {
	return 32 + len([]byte(s))*8
}
