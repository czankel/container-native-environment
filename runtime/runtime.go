// Package runtime provides an abstraction layer for managing containers, images, and snapshots.
//
// The interfaces defined in this package provide a mix of functionality defined by the OCI runtime
// and for managing images and snapshots.
package runtime

import (
	"fmt"
	"io"
	"os"
	"time"

	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go/v1"
	runspecs "github.com/opencontainers/runtime-spec/specs-go"

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

	// GetImage returns an already pulled image or ErrNotFound if the image wasn't found.
	GetImage(name string) (Image, error)

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

	// Snapshots returns all snapshots in the specified domain.
	Snapshots(domain [16]byte) ([]Snapshot, error)

	// Containers returns all containers in the specified domain.
	Containers(domain [16]byte) ([]Container, error)

	// NewContainer defines a new Container but doesn't start it.
	//
	// The container can be started using the <Container>.Start() method.
	// This allows for additional changes or mounting additional paths, etc.
	NewContainer(domain [16]byte, id [16]byte, generation [16]byte,
		image Image, spec *runspecs.Spec) (Container, error)
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

// Container provides an abstraction for running processes in an isolated environment in user space.
//
// Containers are uniquely identified by these fields:
//  - domain:     identifies the project on a system
//  - id:         identification of the container in the domain
//  - generation: describing the underlying filesystem (snapshot)
// Domain and ID are immutable. Generation is mutable and updated for each filesystem modificiation.
//
// Runtimes might only support a single container for a domain and ID and have other restriction.
// See additional information in the interface functions.
// Depending on the implementation, containers might also be destroyed and re-created internally
// for functions, such as Commit or Abort. In those cases, all processes must be deleted before
// calling these functions.
//
// Note that the current implementation does not require to run any process, so the first
// process created will become the init task (PID 1).
type Container interface {

	// CreatedAt returns the date the container was created.
	CreatedAt() time.Time

	// UpdatedAt returns the date the container was last updated.
	UpdatedAt() time.Time

	// Domain returns the immutable domain id of the container.
	// The domain allows for grouping containers.
	Domain() [16]byte

	// ID returns the immutable container id that has to be unique in a domain.
	ID() [16]byte

	// Generation returns a value representing the filesystem.
	Generation() [16]byte

	// Create creates the container.
	// Note that some runtimes delete a prior generation.
	Create() error

	// Start starts the container.
	// Snapshot is an optional argument for defining the base filesystem other than the image.
	// Mutable defines if a process running inside the container can write to the filesystem.
	Start(snapshot Snapshot, mutable bool) error

	// Stop stops the container gracefully and exists all processes asynchronously.
	// Force forces to kill all running processes.
	// This function is idempotent.
	Stop(force bool) error

	// Delete deletes the container.
	Delete() error

	// Commit commits the container with a new generation value. It is idempotent.
	Commit(generation [16]byte) error

	// Exec starts the provided command and returns immediately.
	Exec(stream Stream, cmd []string) (Process, error)
}

// Stream describes the IO channels to a process that is running in a container.
type Stream struct {
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Terminal bool
}

// Snapshot describes a snapshot of the current container filesystem.
// Supporting snapshots is optional, and not all runtimes support it.
type Snapshot interface {

	// Name returns the snapshot name
	Name() string

	// Parent returns the name of the parent snapshot
	Parent() string

	// CreatedAt returns the time the snapshot was created
	CreatedAt() time.Time
}

// Process describes a process running inside a container
type Process interface {

	// Signal sends a signal to the process
	Signal(sig os.Signal) error

	// Wait asynchronously waits for the process to exit, and sends the exit code to the
	// channel.
	Wait() (<-chan ExitStatus, error)
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

// ExitStatus describes the exit status of a background operation
type ExitStatus struct {
	ExitTime time.Time
	Error    error
	Code     uint32 // Exit value from the process
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
