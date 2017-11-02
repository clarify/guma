package log

import (
	"fmt"
	"testing"

	"github.com/searis/guma"
)

// Logger is an internal wrapper for the guma.Logger interface that handles
// zero and nil values.
type Logger struct {
	l  guma.Logger
	tb testing.TB
}

// Println prints a single line of logger output.
func (l *Logger) Println(v ...interface{}) {
	if l == nil || l.l == nil {
		return
	}
	if l.tb != nil {
		l.tb.Helper()
		l.tb.Log(v...)
	} else {
		l.l.Output(2, fmt.Sprintln(v...))
	}
}

// Printf prints a formatted logger output.
func (l *Logger) Printf(format string, v ...interface{}) {
	if l == nil || l.l == nil {
		return
	}
	if l.tb != nil {
		l.tb.Helper()
		l.tb.Logf(format, v...)
	} else {
		l.l.Output(2, fmt.Sprintf(format, v...))
	}
}

// Spewln is a short-hand for l.Println(SpewFmt.Sdump(a...)) that only performs
// the SpewFmt operation if l is set.
func (l *Logger) Spewln(a ...interface{}) {
	if l == nil || l.l == nil {
		return
	}

	if l.tb != nil {
		l.tb.Helper()
		l.tb.Log(SpewFmt.Sdump(a...))
	} else {
		l.l.Output(2, SpewFmt.Sdump(a...))
	}
}

// LogIfError logs "<errPrefix>: <err>"  to l if l is set and err is not nil.
func (l *Logger) LogIfError(errPrefix string, err error) {
	if err == nil || l == nil || l.l == nil {
		return
	}
	if l.tb != nil {
		l.tb.Helper()
		l.tb.Log(errPrefix, ": ", err)
	} else {
		l.l.Output(2, fmt.Sprintln(errPrefix, ": ", err))
	}
}

// testLogger is an optional interface that, if implemented, will take
// presidense over the guma.Logger interface. This is needed because logging to
// a testing.T or testing.B don't support supplying a calldepth parameter.
type testLogger interface {
	TB() testing.TB
}

// WrapLogger wraps l and returns a usable internal logger even if l is nil. If
// l if nil, the returned Logger is also nil, and all operations called on it
// wil be NOOPs.
func WrapLogger(l guma.Logger) *Logger {
	if l == nil {
		return nil
	}
	if tl, ok := l.(testLogger); ok {
		return &Logger{l: l, tb: tl.TB()}
	}
	return &Logger{l: l}
}
