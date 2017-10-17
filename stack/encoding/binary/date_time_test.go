package binary_test

import (
	b "encoding/binary"
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	const hundredNanoSecondsToUnixEpoch = 11644473600 * 1e7
	unixEpochBytes := make([]byte, 8, 8)
	b.LittleEndian.PutUint64(unixEpochBytes, uint64(hundredNanoSecondsToUnixEpoch))

	cases := []TranscoderTest{
		{
			SubTests:     TestEncode | TestDecode,
			Name:         `1601-01-01T00:00:01.00`,
			Unmarshaled:  time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC),
			DecodeTarget: new(time.Time),
			Marshaled: []byte{
				0, 0, 0, 0, 0, 0, 0, 0,
			},
		},
		{
			SubTests:     TestEncode | TestDecode,
			Name:         `1970-01-01T00:00:00.00`,
			Unmarshaled:  time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			DecodeTarget: new(time.Time),
			Marshaled:    unixEpochBytes,
		},
	}

	for i := range cases {
		cases[i].Run(t)
	}
}
