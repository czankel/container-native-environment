// Package runtime provides an abstraction layer for managing containers, images, and snapshots.
// The included interfaces don't  beyond the the OCI runtime, which is limited to running containers and
// does not, for example, include volume management, which is part of a different specification.

package runtime

import (
	"fmt"
	"time"

	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/errdefs"
)

// Runtime is the main interface for managing containers and images, which provide additional
// interfaces to interact with them.
type Runtime interface {

	// Close closes the runtime and any open descriptors
	Close()

	// Images returns a list of images that are registered in the runtime
	Images() ([]Image, error)

	// PullImage returns a locally cached image or pulls the image from the registry
	PullImage(name string) (Image, error)

	// DeleteImage deletes the specified image from the registry.
	DeleteImage(name string) error
}

// Image describes an image
type Image interface {

	// Name returns the image name
	Name() string

	// Digest returns the digest of the image
	Digest() digest.Digest

	// CreatedAt returns the data the image was created
	CreatedAt() time.Time

	// Config returns the configuration of the image
	Config() (*v1.ImageConfig, error)

	// Size returns the size of the image
	Size() int64
}

//
// Registry
//

// RuntimeType is a construct that allows to self-register runtime implementations
type RuntimeType interface {
	Open(config.Runtime) (Runtime, error)
}

var runtimes map[string]RuntimeType

// Register registers a new Runtime Registrar
// ErrResourceExists: already registered
func Register(name string, runType RuntimeType) error {
	if runtimes == nil {
		runtimes = make(map[string]RuntimeType)
	}

	_, ok := runtimes[name]
	if ok {
		return errdefs.AlreadyExists("runtime", name)
	}
	runtimes[name] = runType
	return nil
}

// Runtimes returns a list of all registered runtimes.
func Runtimes() []string {
	names := make([]string, 0, len(runtimes))
	for n, _ := range runtimes {
		names = append(names, n)
	}
	return names
}

// Open opens a new runtime for the specified name
func Open(confRun config.Runtime) (Runtime, error) {
	reg, ok := runtimes[confRun.Name]
	if !ok {
		return nil, errdefs.NotFound("runtime", confRun.Name)
	}
	return reg.Open(confRun)
}

// Errorf returns a runtime error for unspecific errors that cannot be mapped to a error type.
//
// This function should be used mostly for internal errors. Others, for example, invalid arguments,
// already exists, not found, etc. should use the errors defined in errdefs directly.
func Errorf(format string, args ...interface{}) error {
	return errdefs.New(errdefs.ErrRuntimeError,
		fmt.Sprintf("runtime: "+format, args...))
}
