package guma

// Logger defines the interface that is required when providing a debug logger
// for a guma a package. Some guma package provice a `SetDebugLogger` function
// which can be called to enable logging from that package.
type Logger interface {
	Output(calldepth int, s string) error
}
