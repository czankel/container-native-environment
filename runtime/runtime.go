// Package runtime provides an abstraction layer for managing containers, images, and snapshots.
//
// The interfaces defined in this package provide a mix of functionality defined by the OCI runtime
// and for managing images and snapshots.
package runtime

import (
	"fmt"
	"time"

	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/errdefs"
)

// Runtime is the main interface for managing containers, images, and snapshots.
type Runtime interface {

	// Namespace returns the namespace that was used for opening the runtime
	Namespace() string

	// Close closes the runtime and any open descriptors
	Close()

	// Images returns a list of images that are registered in the runtime
	Images() ([]Image, error)

	// PullImage pulls an image into a local registry and returns an image instance.
	//
	// PullImage is a blocking call and reports the progress through the optionally provided
	// channel. The channel can be nil to skip sending updates.
	//
	// Note that the status sent may exclude status information for entries that haven't
	// changed.
	PullImage(name string, progress chan<- []ProgressStatus) (Image, error)

	// DeleteImage deletes the specified image from the registry.
	DeleteImage(name string) error
}

// Image describes an image that consists of a file system and configuration options.
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

// Current status of the progress
const (
	StatusUnknown  = "unknown"
	StatusExists   = "exists"
	StatusPending  = "pending"
	StatusRunning  = "running"
	StatusComplete = "complete"
	StatusAborted  = "aborted"
	StatusError    = "error"
)

// ProgressStatus provides information about a running or completed image download or processes.
type ProgressStatus struct {
	Reference string    // Resource reference, such as image or process id.
	Status    string    // Progress status (StatusPending, ...)
	Offset    int64     // Nominator: Current offset in a file or progress
	Total     int64     // Denominator: Size or total time.
	Details   string    // Additional optional information
	StartedAt time.Time // Time the job was started.
	UpdatedAt time.Time // Time the job was last updated (or when it was completed)
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
