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
	ErrInUse = errors.New("in use")
	// error: not connected
	ErrNotConnected = errors.New("not connected")

	// pass-through errors
	ErrCommandFailed   = errors.New("cmd failed")
	ErrCommandNotFound = errors.New("cmd not found")
)

type cneError struct {
	cause    error
	resource string
	name     string
	msg      string
}

// Create a new error with one of the pre-defined Err* causes in this file and message
func New(cause error, resource, msg string) error {
	return &cneError{
		cause:    cause,
		resource: resource,
		msg:      msg,
	}
}

func (cerr *cneError) Error() string {
	return cerr.msg
}

func (cerr *cneError) Is(other error) bool {
	return cerr.cause == other
}

func IsError(err error, other error) bool {
	return err.(*cneError).Is(other)
}

func IsCneError(err interface{}) bool {
	switch err.(type) {
	case *cneError:
		return true
	default:
		return false
	}
}

func Cause(err interface{}) string {
	if !IsCneError(err) {
		return ""
	}
	return err.(*cneError).cause.Error()
}

func Resource(err interface{}) string {
	if !IsCneError(err) {
		return ""
	}
	return err.(*cneError).resource
}

func Name(err interface{}) string {
	if !IsCneError(err) {
		return ""
	}
	return err.(*cneError).name
}

func Message(err interface{}) string {
	if !IsCneError(err) {
		return ""
	}
	return err.(*cneError).msg
}

func InvalidArgument(format string, args ...interface{}) error {
	return &cneError{
		cause: ErrInvalidArgument,
		msg:   fmt.Sprintf(format, args...),
	}
}

func AlreadyExists(resource, name string) error {
	return &cneError{
		cause:    ErrAlreadyExists,
		resource: resource,
		name:     name,
		msg:      fmt.Sprintf("%s '%s' already exist", resource, name),
	}
}

func NotFound(resource, name string) error {
	return &cneError{
		cause:    ErrNotFound,
		resource: resource,
		name:     name,
		msg:      fmt.Sprintf("%s '%s' not found", resource, name),
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
		cause:    ErrNotImplemented,
		resource: "function",
		name:     fnName,
		msg:      fmt.Sprintf("function '%s' has not been implemented", fnName),
	}
}

func InternalError(format string, args ...interface{}) error {
	return &cneError{
		cause: ErrInternalError,
		msg:   fmt.Sprintf(format, args...),
	}
}

func InUse(resource, name string) error {
	return &cneError{
		cause:    ErrInUse,
		resource: resource,
		name:     name,
		msg:      fmt.Sprintf("%s '%s' is in use", resource, name),
	}
}

func NotConnected() error {
	return &cneError{
		cause: ErrNotConnected,
		msg:   "",
	}
}

// 'Pass-through' errors

// FIXME are these even used??

type execError struct {
	cause error
	msg   string
}

func (eerr *execError) Error() string {
	return eerr.msg
}

func (eerr *execError) Is(other error) bool {
	return eerr.cause == other
}

func CommandNotFound(cmd string) error {
	return &execError{
		cause: ErrCommandNotFound,
		msg:   fmt.Sprintf("%s: command not found", cmd),
	}
}
func CommandFailed(cmd []string) error {
	return &execError{
		cause: ErrCommandFailed,
		msg:   fmt.Sprintf("Command failed: %s", strings.Join(cmd, " ")),
	}
}
