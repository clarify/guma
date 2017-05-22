package guma

import "fmt"

// bitTranscoder allows encoding (compression) of bit sequences and small byte
// values into a single byte and decoding (extraction) of bit and byte values
// from a single byte.
type bitTranscoder struct {
	cache  byte
	cursor byte
}

// SetBits encodes the n least significant bits from data into bt's cache and
// increments the cursor. If data does not fit in n bits, an error is returned.
// If n is less than 1 or above 8, a panic is raised.
func (bt *bitTranscoder) SetBits(data, n byte) error {
	if n < 1 || n > 8 {
		panic("bit length out of range")
	}

	var max byte = 0xFF >> (8 - n)
	if data > max {
		return fmt.Errorf("can't encode 0x%.2X > 0x%.2X into %d bits", data, max, n)
	}

	bt.cache |= data << bt.cursor
	bt.cursor += n

	return nil
}

// MarshalBinary clears m's bit cache and return a singe byte if the bit cache
// is full. Otherwise, nil is returned.
func (bt *bitTranscoder) MarshalBinary() ([]byte, error) {
	if bt.cursor < 8 {
		return nil, nil
	}
	if bt.cursor > 8 {
		// FIXME: If OPC UA bug 3252 is fixed, we should be able to raise an
		// error here.
		// - https://opcfoundation-onlineapplications.org/mantis/view.php?id=3252
		debug.Printf("MarshalBinary: ignoring bitTranscoder cursor position > 8; cursor position is %d", bt.cursor)
	}
	ret := []byte{bt.cache}
	bt.cache = 0
	bt.cursor = 0
	return ret, nil
}

// BitLength returns 8 only when the cursor is 0.
func (bt *bitTranscoder) BitLength() int {
	if bt.cursor == 0 {
		return 8
	}
	return 0
}
