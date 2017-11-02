package testutil

import "testing"

type testLogger struct {
	tb testing.TB
}

func (l testLogger) Output(calldepth int, s string) error {
	// NO-OP This method is ignred when TB is supplied.
	return nil
}

func (l testLogger) TB() testing.TB {
	return l.tb
}
