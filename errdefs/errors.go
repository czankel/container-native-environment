// Package errdefs defines fixed error test to unify error responses across
// modules and functions.
package errdefs

import "errors"

var (
	InvalidArgument = "invalid argument"
	ResourceExists  = "resource exists"
	NoSuchResource  = "no such resource"
	Internal        = "internal error"
	Busy            = "busy"
	Uninitialized   = "uninitialized"
	NotImplemented  = "not implemented"
	Overflow        = "too many resources"
	ReadOnly        = "read only"
)

var (
	ErrInvalidArgument = errors.New(InvalidArgument)
	ErrResourceExists  = errors.New(ResourceExists)
	ErrNoSuchResource  = errors.New(NoSuchResource)
	ErrInternal        = errors.New(Internal)
	ErrBusy            = errors.New(Busy)
	ErrUninitialized   = errors.New(Uninitialized)
	ErrNotImplemented  = errors.New(NotImplemented)
	ErrOverflow        = errors.New(Overflow)
	ErrReadOnly        = errors.New(ReadOnly)
)
