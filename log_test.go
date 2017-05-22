package guma_test

import "testing"

type testLogger struct {
	testing.TB
}

func (l testLogger) Println(v ...interface{}) {
	l.TB.Log(v...)
}

func (l testLogger) Printf(format string, v ...interface{}) {
	l.TB.Logf(format, v...)
}
