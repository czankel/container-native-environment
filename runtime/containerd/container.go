// Package containerd implements the runtime interface for the ContainerD Dameon containerd.io
package containerd

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/containerd/containerd"
	runspecs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/runtime"
)

type container struct {
	domain        [16]byte
	id            [16]byte
	generation    [16]byte
	spec          *runspecs.Spec
	image         *image
	ctrdRuntime   *containerdRuntime
	ctrdContainer containerd.Container
}

// splitCtrdID splits the containerd ID into domain and ID
func splitCtrdID(ctrdID string) ([16]byte, [16]byte, error) {

	idx := strings.Index(ctrdID, "-")
	s, err := hex.DecodeString(ctrdID[:idx])
	if err != nil {
		return [16]byte{}, [16]byte{},
			errdefs.InvalidArgument("container ID is invalid: '%s': %v", ctrdID, err)
	}
	var dom [16]byte
	copy(dom[:], s)

	s, err = hex.DecodeString(ctrdID[idx+1:])
	if err != nil {
		return [16]byte{}, [16]byte{},
			errdefs.InvalidArgument("container ID is invalid: '%s': %v", ctrdID, err)
	}
	var id [16]byte
	copy(id[:], s)

	return dom, id, nil
}

// composeID composes the containerd ID from the domain and container ID
func composeID(domain [16]byte, id [16]byte) string {
	return hex.EncodeToString(domain[:]) + "-" + hex.EncodeToString(id[:])
}

func (ctr *container) Domain() [16]byte {
	return ctr.domain
}

func (ctr *container) ID() [16]byte {
	return ctr.id
}

func (ctr *container) Generation() [16]byte {
	return ctr.generation
}

func (ctr *container) CreatedAt() time.Time {
	// TODO: Container.CreatedAt not yet supported by containerd?
	return time.Now()
}

func (ctr *container) UpdatedAt() time.Time {
	// TODO: Container.updatedAt not yet supported by containerd?
	return time.Now()
}

func (ctr *container) Delete() error {

	ctrdRun := ctr.ctrdRuntime
	ctrdCtr := ctr.ctrdContainer

	err := ctrdCtr.Delete(ctrdRun.context)
	if err != nil {
		return runtime.Errorf("failed to delete container: %v", err)
	}
	return nil
}
