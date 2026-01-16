package sqerr

import (
	"errors"
	"fmt"
	"io"
)

type Code int

type Error struct {
	// custom error type that holds an exit code
	Code Code   // exit code
	Msg  string // context message
	Err  error  // underlying error
}

const (
	Success     Code = 0 // exit codes
	Usage       Code = 1
	IO          Code = 2
	Corrupt     Code = 3
	Unsupported Code = 4
	Internal    Code = 5
)

func (e *Error) Error() string {
	if e.Msg != "" && e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown error"
}

func (e *Error) Unwrap() error {
	return e.Err // the underlying error
}

func New(code Code, msg string) error {
	return &Error{Code: code, Msg: msg}
}

func CodedError(err error, code Code, msg string) error {
	if err == nil {
		return nil // nil error deserves no code
	}
	var wrappedError *Error            // make a custom error
	if errors.As(err, &wrappedError) { // if there is an *Error in the err tree, return it
		return err
	}
	return &Error{Code: code, Msg: msg, Err: err} // otherwise return a new *Error
}

func ErrorCode(err error) Code {
	if err == nil {
		return Success // no error means 'Great Success!' - Borat
	}
	var wrappedError *Error            // make a custom error
	if errors.As(err, &wrappedError) { // if there is an *Error in the err tree, return it's code
		return wrappedError.Code
	}
	return Internal // otherwise the code was internal (catch all)
}

func ReadErrorCode(err error) Code {
	if err == nil {
		return Success // nil error means success
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.ErrShortBuffer) {
		// above errors present themselves when there is not enough bytes or the stream terminated early
		// typical of truncated or corrupted data
		return Corrupt
	}
	return IO
}
