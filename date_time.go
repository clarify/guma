package guma

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
