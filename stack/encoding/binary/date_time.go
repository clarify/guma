package binary

import "time"

import "encoding/binary"

// secondsToUnixEpoch counts the number of seconds since the Windows NT time
// epoch (January 1 1601) to the Unix/POSIX time epoch.
const (
	secondsToUnixEpoch            int64 = 11644473600
	hundredNanoSecondsToUnixEpoch int64 = secondsToUnixEpoch * 1e7
)

const minInt64 int64 = -1 << 63

// dateTime converts between OPC UA DateTime 64-bit binary representation
// (Windwows NT timestamp) and Go time.Time structs.
type dateTime time.Time

// MarshalBinary encodes a time.Time struct into a 64-bit OPC UA DateTime
// representation.
func (t dateTime) MarshalBinary() ([]byte, error) {
	b := make([]byte, 8)

	sec := time.Time(t).Unix()
	nsec := time.Time(t).Nanosecond()

	i := sec*1e7 + hundredNanoSecondsToUnixEpoch + int64(nsec/100)
	binary.LittleEndian.PutUint64(b, uint64(i))

	return b, nil
}

// UnmarshalBinary decodes a 64-bit Windows NT timestamp into a Go time.Time
// struct.
func (t *dateTime) UnmarshalBinary(data []byte) error {
	i := int64(binary.LittleEndian.Uint64(data[0:8]))
	i -= hundredNanoSecondsToUnixEpoch

	sec := i / 1e7
	nsec := (i - sec*1e7) * 100

	*t = dateTime(time.Unix(sec, nsec).UTC())
	return nil
}

func (t dateTime) BitLength() int {
	return 8
}
