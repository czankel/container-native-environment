// Package errdefs provides an extension to the error handling of go.
//
// All errors returned from a function in any of this project must return
// one of the error types defined in this file. This allows for a more
// homogenieous error handling and display.
package errdefs

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
	// error: invalid argument: <description>
	ErrAlreadyExists = errors.New("already exists")
	// error: <resource> '<name>' already exists
	ErrNotFound = errors.New("not found")
	// error: <resource> '<name>' not found
	ErrSystemError = errors.New("system")
	// error: <description>: <system error>
	ErrRuntimeError = errors.New("runtime")
	// error: runtime error: <runtime error>
	ErrNotImplemented = errors.New("not implemented")
	// error: function '<name>' is not implemented
	ErrInternalError = errors.New("internal error")
	// error: internal error: <description>
	// error: internal error: <description>
	ErrInUse = errors.New("in use")

	// pass-through errors
	ErrCommandFailed = errors.New("cmd failed")
)

type cneError struct {
	cause error
	msg   string
}

// Create a new error with one of the pre-defined Err* causes in this file and message
func New(cause error, msg string) error {
	return &cneError{
		cause: cause,
		msg:   msg,
	}
}

func (err *cneError) Error() string {
	return err.msg
}

// Is must be used with the Err* variables
func (cerr *cneError) Is(other error) bool {
	return cerr.cause == other
}

func IsCneError(err interface{}) bool {
	switch err.(type) {
	case cneError:
		return true
	default:
		return false
	}
}

func InvalidArgument(format string, args ...interface{}) error {
	return &cneError{
		cause: ErrInvalidArgument,
		msg:   fmt.Sprintf(format, args...),
	}
}

func AlreadyExists(resource, name string) error {
	return &cneError{
		cause: ErrAlreadyExists,
		msg:   fmt.Sprintf("%s '%s' already exist", resource, name),
	}
}

func NotFound(resource, name string) error {
	return &cneError{
		cause: ErrNotFound,
		msg:   fmt.Sprintf("%s '%s' not found", resource, name),
	}
}

func SystemError(err error, format string, args ...interface{}) error {
	return &cneError{
		cause: ErrSystemError,
		msg:   fmt.Sprintf(format+" : "+err.Error(), args...),
	}
}

func NotImplemented() error {

	fnName := "unknown"
	pc, _, _, ok := runtime.Caller(1)

	if ok {
		details := runtime.FuncForPC(pc)
		if details != nil {
			fnName = details.Name()
		}
	}

	return &cneError{
		cause: ErrNotImplemented,
		msg:   fmt.Sprintf("function '%s' has not been implemented", fnName),
	}
}

func InUse(resource, name string) error {
	return &cneError{
		cause: ErrInUse,
		msg:   fmt.Sprintf("%s '%s' is in use", resource, name),
	}
}

func InternalError(format string, args ...interface{}) error {
	return &cneError{
		cause: ErrInternalError,
		msg:   fmt.Sprintf(format, args...),
	}
}

func CommandFailed(cmd []string) error {
	return &cneError{
		cause: ErrCommandFailed,
		msg:   fmt.Sprintf("Command failed: %s", strings.Join(cmd, " ")),
	}
}
