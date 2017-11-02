package binary

import (
	"encoding/binary"
	"time"
)

type duration time.Duration

func (d duration) BitLength() int {
	return 4
}

func (d duration) MarshalBinary() ([]byte, error) {
	// d is stored as nanoseconds, we want it as milliseconds and we are fine
	// with always rounding down. We cast from int64 -> int32 before we cast to
	// uint32 to ensure the sign is kept in the scale-down operation.
	b := make([]byte, 4)
	msec := int32(d / 1e6)
	binary.LittleEndian.PutUint32(b, uint32(msec))
	return b, nil
}

func (d *duration) UnmarshalBinary(data []byte) error {
	i := int32(binary.LittleEndian.Uint32(data[0:4]))
	*d = duration(time.Millisecond) * duration(i)

	return nil
}
