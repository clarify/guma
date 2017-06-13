package binary

import "time"
import "bytes"
import "encoding/binary"

// secondsToUnixEpoch counts the number of seconds since Windows NT time epoch
// (January 1 1601) to the Unix/POSIX time epoch.
const secondsToUnixEpoch int64 = 11644473600

// dateTime converts between OPC UA DateTime 64-bit binary representation
// (Windwows NT timestamp) and Go time.Time structs.
type dateTime time.Time

// MarshalBinary encodes a time.Time struct into a 64-bit OPC UA DateTime
// representation.
func (t dateTime) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	// nanoseconds (e-9) -> milliseconds (e-7).
	i := time.Time(t).UnixNano() / 100
	i += secondsToUnixEpoch * int64(1e7)

	binary.Write(&buf, binary.LittleEndian, i)
	return buf.Bytes(), nil
}

// UnmarshalBinary decodes a 64-bit Windows NT timestamp into a Go time.Time
// struct.
func (t *dateTime) UnmarshalBinary(data []byte) error {
	var i int64

	buf := bytes.NewBuffer(data[0:8])
	if err := binary.Read(buf, binary.LittleEndian, &i); err != nil {
		return err
	}
	// NB! secondsToUnixEpoch * int64(1e9) would overflow.
	i -= secondsToUnixEpoch * int64(1e7)
	// milliseconds (e-7) -> nanoseconds (e-9).
	nsec := i * 100

	*t = dateTime(time.Unix(0, nsec).UTC())
	return nil
}

func (t dateTime) BitLength() int {
	return 8
}
