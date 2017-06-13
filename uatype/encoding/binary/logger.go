package binary

// Logger defines the interface that is required when providing a logger for
// the binary package.
type Logger interface {
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}

var debugLogger Logger = nopLogger{}

// nopLogger is a no-operation Logger implementation.
type nopLogger struct{}

func (d nopLogger) Println(v ...interface{}) {}

func (d nopLogger) Printf(format string, v ...interface{}) {}

// SetDebugLogger let's you specify which logger the binary package should use
// to log debug messages. To unset the logger, let l be nil. By default, no
// logger is set.
func SetDebugLogger(l Logger) {
	if l == nil {
		debugLogger = nopLogger{}
		return
	}
	debugLogger = l
}
