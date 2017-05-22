package guma

import (
	"bytes"
	"errors"
	"fmt"
)

// Common errors that may be returned as the cause for EncoderError and
// DecoderError.
var (
	ErrUnknownType   = errors.New("can not handle type")
	ErrInvalidTag    = errors.New("invalid struct tag")
	ErrInvalidLength = errors.New("length don't match length field value")
)

// EncoderError provides a way of getting the logical path to where in a nested
// data structure an encoder error occurred.
type EncoderError struct {
	transcoderError
	typeName string
}

// Error returns a human readable description of the error.
func (err EncoderError) Error() string {
	return fmt.Sprintf("EncoderError %s%s", err.typeName, err.transcoderError)
}

type transcoderError struct {
	path  []interface{}
	cause error
}

// wrap a trancoderError or other error with field prepended to it's path.
func wrapError(cause error, field interface{}) error {
	err := transcoderError{
		path: []interface{}{field},
	}
	if trErr, ok := cause.(transcoderError); ok {
		err.path = append(err.path, trErr.path...)
		err.cause = trErr.cause
	} else {
		err.cause = cause
	}
	return err
}

// Path returns the error path.
func (err transcoderError) Path() []interface{} {
	return err.path
}

// Cause returns the original error.
func (err transcoderError) Cause() error {
	return err.cause
}

// Error returns a nicely formatted error string.
func (err transcoderError) Error() string {
	var buf bytes.Buffer

	for _, field := range err.path {
		switch ft := field.(type) {
		case int:
			fmt.Fprintf(&buf, "[%d]", ft)
		default:
			fmt.Fprint(&buf, ".")
			fmt.Fprint(&buf, ft)
		}
	}
	fmt.Fprint(&buf, ": ", err.cause)
	return buf.String()
}
