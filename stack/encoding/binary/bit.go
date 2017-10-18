package binary

import (
	"errors"
	"fmt"
)

// bitCacheMarshaler allows encoding (compression) of bit sequences and small
// byte values into a single byte. To achieve this, the set bits are cached
// until full byte is ready for writing/releasing.
type bitCacheMarshaler struct {
	cache  byte
	cursor byte
}

// SetBits encodes the n least significant bits from data into b's cache and
// increments the cursor. An error is returned if n is not in range 1-8, or if
// data can not be encoded into n bits.
func (m *bitCacheMarshaler) SetBits(data, n byte) error {
	if n < 1 || n > 8 {
		return ErrInvalidBitLength
	}

	var max byte = 0xFF >> (8 - n)
	if data > max {
		return fmt.Errorf("can't encode 0x%.2X > 0x%.2X into %d bits", data, max, n)
	}

	m.cache |= data << m.cursor
	m.cursor += n

	return nil
}

// MarshalBinary clears m's bit cache and return a singe byte if the bit cache
// is full. Otherwise, nil is returned.
func (m *bitCacheMarshaler) MarshalBinary() ([]byte, error) {
	if m.cursor < 8 {
		return nil, nil
	}
	if m.cursor > 8 {
		return nil, fmt.Errorf("bitCacheMarshaler: MarshalBinary: cursor position %d > 8", m.cursor)
	}
	ret := []byte{m.cache}
	m.cache = 0
	m.cursor = 0
	return ret, nil
}

// bitCacheUnmarshaler allows unmarshaling (decompressing) bit sequences and
// small byte values from a single byte. To achieve this, a cache is used to
// extract values from. Only when the cache is empty will a byte be consumed
// by UnmarshalBinary (and reported by BitLength).
type bitCacheUnmarshaler struct {
	cache      byte
	cursor     byte
	byteTarget *byte
	boolTarget *bool
	nBits      byte
	byteRead   bool
}

// SetTarget lets you set where the next UnmarshalBinary call will write it's
// results. An error is returned if n is not in range 1-8 or if v is nil.
func (u *bitCacheUnmarshaler) SetTarget(v *byte, n byte) error {
	if v == nil {
		return errors.New("target may not be nil")
	}
	if n < 1 || n > 8 {
		return ErrInvalidBitLength
	}

	u.boolTarget = nil
	u.byteTarget = v
	u.nBits = n
	return nil
}

// SetBoolTarget lets you set a boolean pointer where the next UnmarshalBinary
// call will write it's result.
func (u *bitCacheUnmarshaler) SetBoolTarget(v *bool) {
	u.boolTarget = v
	u.byteTarget = nil
	u.nBits = 1
}

// UnmarshalBinary sets the target value from the bit cache, and then forgets
// the previously set target. If the bit cache is empty, a byte is first read
// from data. If no target is set, a panic is raised.
func (u *bitCacheUnmarshaler) UnmarshalBinary(data []byte) error {
	// read one byte to cache if needed.
	if u.cursor == 0 {
		u.cache = data[0]
		u.byteRead = true
	} else {
		u.byteRead = false
	}

	// read bits from cache.
	val := u.readBits()
	if u.boolTarget != nil {
		*u.boolTarget = (val == 1)
	} else if u.byteTarget != nil {
		*u.byteTarget = val
	} else {
		// May only be caused by a programming error within the guma package.
		panic("decode target not set")
	}

	// Clear target.
	u.boolTarget = nil
	u.byteTarget = nil
	u.nBits = 0

	return nil

}

func (u *bitCacheUnmarshaler) readBits() byte {
	var mask byte = 0xFF >> (8 - u.nBits)

	ret := (u.cache >> u.cursor) & mask
	u.cursor += u.nBits
	if u.cursor > 8 {
		// FIXME: If OPC UA bug 3252 is fixed, we should be able to raise an
		// error here.
		// - https://opcfoundation-onlineapplications.org/mantis/view.php?id=3252
		debugLogger.Printf("bitCacheUnmarshaler: UnmarshalBinary: ignoring cursor position > 8; cursor position is %d", u.cursor)
	}
	if u.cursor >= 8 {
		u.cache = 0
		u.cursor = 0
	}
	return ret
}

// BitLength returns 8 if the last call to UnmarshalBinary resulted in a byte
// being read. 0 is returned otherwise.
func (u *bitCacheUnmarshaler) BitLength() int {
	if u.byteRead {
		return 8
	}
	return 0
}
