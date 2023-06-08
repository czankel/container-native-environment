// Package runtime provides an abstraction layer for managing containers, images, and snapshots.
//
// The interfaces defined in this package provide a mix of functionality defined by the OCI runtime
// and for managing images and snapshots.
package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	digest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/errdefs"
)

// Runtime is the main interface for managing containers, images, and snapshots.
type Runtime interface {

	// WithNamespace sets the namespace
	WithNamespace(ctx context.Context, ns string) context.Context

	// Close closes the runtime and any open descriptors
	Close()

	// Images returns a list of images that are registered in the runtime
	Images(ctx context.Context) ([]Image, error)

	// GetImage returns an already pulled image or ErrNotFound if the image wasn't found.
	GetImage(ctx context.Context, name string) (Image, error)

	// PullImage pulls an image into a local registry and returns an image instance.
	//
	// PullImage is a blocking call and reports the progress through the optionally provided
	// channel. The channel can be nil to skip sending updates.
	//
	// Note that the status sent may exclude status information for entries that haven't
	// changed.
	PullImage(ctx context.Context, name string, progress chan<- []ProgressStatus) (Image, error)

	// DeleteImage deletes the specified image from the registry.
	DeleteImage(ctx context.Context, name string) error

	// Snapshots returns all snapshots.
	Snapshots(ctx context.Context) ([]Snapshot, error)

	// GetSnapshot returns the specific snapshot
	GetSnapshot(ctx context.Context, name string) (Snapshot, error)

	// DeleteSnapshot deletes the snapshot
	DeleteSnapshot(ctx context.Context, name string) error

	// Containers returns all containers in the specified domain.
	// FIXME: describe filters...
	Containers(ctx context.Context, filters ...interface{}) ([]Container, error)

	// GetContainer looks up and returns the specified container by domain, id, and generation.
	// It returns ErrNotFound if the container could not be found.
	//
	// The container can be used to execute commands with Exec.
	GetContainer(ctx context.Context, domain, id, generation [16]byte) (Container, error)

	// NewContainer defines a new Container without creating it.
	NewContainer(ctx context.Context,
		domain, id, generation [16]byte, uid uint32, image Image) (Container, error)

	// DeleteContainer deletes the specified container. It returns ErrNotFound if the container
	// doesn't exist.
	DeleteContainer(ctx context.Context, domain, id, generation [16]byte) error

	// PurgeContainer deletes the specified container and all associated resources. It returns
	// ErrNotFound if the container doesn't exist.
	PurgeContainer(ctx context.Context, domain, id, generation [16]byte) error
}

// Image describes an image that consists of a file system and configuration options.
type Image interface {

	// Name returns the image name.
	Name() string

	// Size returns the size of the image.
	Size() int64

	// Digest returns the digest of the image.
	Digest() digest.Digest

	// CreatedAt returns the data the image was created.
	CreatedAt() time.Time

	// Config returns the configuration of the image.
	Config(ctx context.Context) (*v1.ImageConfig, error)

	// RootFS returns the digests of the root fs the image consists of.
	RootFS(ctx context.Context) ([]digest.Digest, error)

	// Mount mounts the image to the provide path.
	Mount(ctx context.Context, path string) error

	// Unmount unmounts the image from the specified path,
	Unmount(ctx context.Context, path string) error
}

// Container provides an abstraction for running processes in an isolated environment in user space.
//
// Containers are uniquely identified by these fields:
//   - domain:     identifies the project on a system
//   - id:         identification of the container in the domain
//   - generation: describing the underlying filesystem (snapshot)
//
// Domain and ID are immutable. Generation is mutable and updated for any modifications to the
// configuration and filesystem.
//
// Runtimes might only support a single container for a domain and ID and have other restriction.
// See additional information in the interface functions.
//
// Depending on the implementation, containers might also be destroyed and re-created internally.
//
// Note that the current implementation does not require to run any process, so the first
// process created will become the init task (PID 1).
type Container interface {

	// Name returns the container name consisting of the concatenaded string
	// of domain, ID, and generation
	Name() string

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

	// Return the User ID
	UID() uint32

	// Snapshots returns all container snapshots.
	Snapshots(ctx context.Context) ([]Snapshot, error)

	// SetRootFS sets the rootfs to the provide snapshot.
	//
	// The root filesystem can only be set when the container has not been created.
	SetRootFS(ctx context.Context, snapshot Snapshot) error

	// Create creates the container.
	Create(ctx context.Context) error

	// Delete deletes the container.
	Delete(ctx context.Context) error

	// Purge deletes the container and all snapshots that are not otherwise used.
	Purge(ctx context.Context) error

	// Snapshot creates a snapshot of the current file system.
	//
	// Snapshot support is optional, and runtimes that don't support it return an
	// ErrNotImplemented error and nil for the snapshot.
	Snapshot(ctx context.Context) (Snapshot, error)

	// Amend amends the committed snapshot with the current changes to the filesystem.
	Amend(ctx context.Context) (Snapshot, error)

	// Commit commits the container after it has been built with a new generation value.
	Commit(ctx context.Context, generation [16]byte) error

	// Mount adds a local mount point to the container.
	// This must be called before comitting the container, for example, to
	// mount the home directory after building the container.
	Mount(ctx context.Context, destination string, source string) error

	// Exec starts the provided command in the process spec and returns immediately.
	// The container must be started before calling Exec.
	Exec(ctx context.Context, stream Stream, procSpec *ProcessSpec) (Process, error)
}

// Stream describes the IO channels to a process that is running in a container.
type Stream struct {
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Terminal bool
}

// ProcessSpec defines the process to be executed inside the container
type ProcessSpec struct {
	Args []string // arguments
	Env  []string // environment variables
	Cwd  string   // current directory
	UID  uint32   // user ID
	GID  uint32   // group ID
}

// Snapshot describes a snapshot of the current container filesystem.
type Snapshot interface {

	// Name returns the snapshot name.
	Name() string

	// Parent returns the name of the parent snapshot.
	Parent() string

	// CreatedAt returns the time the snapshot was created.
	CreatedAt() time.Time

	// Size returns the size of the snapshot.
	Size() int64

	// Inodex returns the number of additional inodes in the snapshot.
	Inodes() int64
}

// Process describes a process running inside a container.
type Process interface {

	// Signal sends a signal to the process.
	Signal(ctx context.Context, sig os.Signal) error

	// Wait waits asynchronously for the process to exit and sends the exit code to the channel.
	Wait(ctx context.Context) (<-chan ExitStatus, error)
}

const (
	StatusUnknown   = "unknown"
	StatusPending   = "pending"
	StatusLoading   = "loading"
	StatusUnpacking = "unpacking"
	StatusCached    = "cached"
	StatusRunning   = "running"
	StatusComplete  = "complete"
	StatusError     = "error"
)

// ProgressStatus provides information about a running or completed image download or processes.
type ProgressStatus struct {
	Reference string    // Resource reference, such as image or process id.
	Status    string    // Progress status (StatusPending, ...)
	Offset    int64     // Nominator: Current offset in a file or progress
	Total     int64     // Denominator: Size or total time.
	Details   string    // Additional optional information
	StartedAt time.Time // Time the job was started.
	UpdatedAt time.Time // Time the job was last updated (or when it was completed).
}

// ExitStatus describes the exit status of a background operation.
type ExitStatus struct {
	ExitTime time.Time
	Error    error
	Code     uint32 // Exit value from the process
}

//
// Runtime Registry
//

// RuntimeType is a construct that allows to self-register runtime implementations.
type RuntimeType interface {
	Open(context.Context, *config.Runtime) (Runtime, error)
}

var runtimes map[string]RuntimeType

// Register registers a new Runtime Registrar.
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

// Open opens a new runtime for the specified name.
func Open(ctx context.Context, confRun *config.Runtime) (Runtime, error) {
	reg, ok := runtimes[confRun.Name]
	if !ok {
		return nil, errdefs.NotFound("runtime", confRun.Name)
	}
	return reg.Open(ctx, confRun)
}

// Errorf is an internal function to create an error specific to the runtime.
//
// This function should be used only for internal errors that cannot be mapped to one of the
// errors defined in errdefs.
func Errorf(format string, args ...interface{}) error {
	return errdefs.New(errdefs.ErrRuntimeError,
		"",
		fmt.Sprintf("runtime: "+format, args...))
}
