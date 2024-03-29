package binary

import (
	"github.com/searis/guma"
	"github.com/searis/guma/internal/log"
)

var debugLogger *log.Logger

// SetDebugLogger sets a debug logger for this package.
func SetDebugLogger(l guma.Logger) {
	debugLogger = log.WrapLogger(l)
}
