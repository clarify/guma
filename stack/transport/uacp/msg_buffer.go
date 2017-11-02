package uacp

import (
	"errors"
	"io"
)

// errNotEnoughSpace is returned by fixedSizeBuffer Write when there is not
// enough space to write all the passed in data.
var errNotEnoughSpace = errors.New("not enough space")

// fixedSizeBuffer is similar to bytes.Buffer, but without automatic allocation
// and growning of the buffer, it returns errNotEnoughSpace when the buffer
// is full.
type fixedSizeBuffer struct {
	bytes      []byte
	wpos, rpos int
}

// newFixedSizeBuffer expects b to be preallocated with the desired fixed size.
// the slice capasity is ignored for now.
func newFixedSizeBuffer(b []byte) *fixedSizeBuffer {
	return &fixedSizeBuffer{bytes: b, wpos: 0}
}

// Len returns the number of bytes written to buf.
func (buf fixedSizeBuffer) Len() int {
	return buf.wpos
}

// Cap returns the maximum number of bytes that may be written to buf.
func (buf fixedSizeBuffer) Cap() int {
	return len(buf.bytes)
}

// Write will write b to buf. If there is not enough space to write all of b, we
// write as much of b as we can fit before returning errNotEnoughSpace.
func (buf *fixedSizeBuffer) Write(b []byte) (n int, err error) {
	n = copy(buf.bytes[buf.wpos:], b)
	buf.wpos += n
	if len(b) > n {
		err = errNotEnoughSpace
	}
	return
}

// Bytes returns all bytes written to buf.
func (buf *fixedSizeBuffer) Bytes() []byte {
	return buf.bytes[:buf.wpos]
}

// Read reads data from buf into p or returns io.EOF.
func (buf *fixedSizeBuffer) Read(p []byte) (int, error) {
	if len(p) == buf.rpos-buf.wpos {
		return 0, io.EOF
	}
	n := copy(p, buf.bytes[buf.rpos:buf.wpos])
	buf.rpos += n
	return n, nil
}

// Truncate reduces the size resets the writer position to maxSize, if maxSize
// is larger than the current position. It returns weather or not the cursor
// was moved.
func (buf *fixedSizeBuffer) Truncate(maxSize int) bool {
	if maxSize < buf.wpos {
		buf.wpos = maxSize
		return true
	}
	return false
}
