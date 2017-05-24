package guma

var debugLogger Logger = noLogger{}

// Logger defines the interface that must be implemented when assigning a guma
// debug logger.
type Logger interface {
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}

// noLogger is a nop Logger implementation,
type noLogger struct{}

func (d noLogger) Println(v ...interface{}) {}

func (d noLogger) Printf(format string, v ...interface{}) {}

// SetDebugLogger lets you set or clear the debug logger. Turn off the debug log
// by letting l be nil. By default, debug logging is turned off.
func SetDebugLogger(l Logger) {
	if l == nil {
		debugLogger = &noLogger{}
	} else {
		debugLogger = l
	}
}
