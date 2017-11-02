package uacp

import (
	"github.com/searis/guma"
	"github.com/searis/guma/internal/log"
)

var debugLogger *log.Logger

// SetDebugLogger sets a debug logger for this package.
func SetDebugLogger(l guma.Logger) {
	debugLogger = log.WrapLogger(l)
}

var logger *log.Logger

// SetLogger sets a logger for this package. It will rarely be used.
func SetLogger(l guma.Logger) {
	logger = log.WrapLogger(l)
}
