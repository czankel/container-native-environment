// Package runtime provides an abstraction layer for managing containers, images, and snapshots.
// The included interfaces don't  beyond the the OCI runtime, which is limited to running containers and
// does not, for example, include volume management, which is part of a different specification.

package runtime

import (
	"github.com/czankel/cne/config"
	"github.com/czankel/cne/errdefs"
)

// Runtime is the main interface for managing containers and images, which provide additional
// interfaces to interact with them.
type Runtime interface {

	// Close closes the runtime and any open descriptors
	Close()
}

type runtimeType interface {
	Open(config.Runtime) (Runtime, error)
}

var runtimes map[string]runtimeType

// Register registers a new Runtime Registrar
func Register(name string, runType runtimeType) error {
	if runtimes == nil {
		runtimes = make(map[string]runtimeType)
	}

	_, ok := runtimes[name]
	if ok {
		return errdefs.ErrResourceExists
	}
	runtimes[name] = runType
	return nil
}

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
		return nil, errdefs.ErrNoSuchResource
	}
	return reg.Open(confRun)
}
