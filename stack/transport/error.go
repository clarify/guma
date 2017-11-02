package transport

import (
	"errors"
	"fmt"

	"github.com/searis/guma/stack/uatype"
)

// Error is returned by a SecureChannel operation if there is an error
// with the underlying protocol. It contains a status code, an origing string
// ("remote" or "local") and optionally a reason.
type Error struct {
	code   uatype.StatusCode
	origin string
	reason error
}

// LocalError returns a Error with origin set to "local" for the given
// code and reason.
func LocalError(code uatype.StatusCode, reason error) *Error {
	return &Error{
		code:   code,
		origin: "local",
		reason: reason,
	}
}

// RemoteError returns a Error with origin set to "remote" for the given
// code and error.
func RemoteError(code uatype.StatusCode, reason string) *Error {
	return &Error{
		code:   code,
		origin: "remote",
		reason: errors.New(reason),
	}
}

// Error returns the status code and text in a formatted string.
func (err Error) Error() string {
	msg := uatype.StatusText(err.code)
	if msg == "" {
		msg = fmt.Sprintf("%s 0x%.8X (unknown status code)", err.origin, err.code)
	} else {
		msg = fmt.Sprintf("%s 0x%.8X: %s", err.origin, err.code, msg)
	}
	if err.reason != nil {
		msg += ", reason: " + err.reason.Error()
	}
	return msg
}

// StatusCode returns the OPC UA status code for the error.
func (err Error) StatusCode() uatype.StatusCode {
	return err.code
}

// Origin returns either "local" or "remote", describing if err is generated
// locally, or retrieved from the remote connection.
func (err Error) Origin() string {
	return err.origin
}

// Reason returns either the go error that caused a local error, or a go error
// that wraps the reason given by the remote.
func (err Error) Reason() error {
	return err.reason
}
