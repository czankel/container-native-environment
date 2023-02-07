package remote

import (
	"fmt"
	"time"

	runspecs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/runtime"
)

// FIXME: is this 'common' across all implementations?
type container struct {
	domain     [16]byte
	id         [16]byte
	generation [16]byte
	uid        uint32
	spec       runspecs.Spec
	image      *image
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

func (ctr *container) UID() uint32 {
	return ctr.uid
}

func (ctr *container) CreatedAt() time.Time {
	fmt.Println("container.CreatedAt")
	// TODO: Container.CreatedAt not yet supported by containerd?
	return time.Now()
}

func (ctr *container) UpdatedAt() time.Time {
	fmt.Println("container.UpdatedAt")
	// TODO: Container.updatedAt not yet supported by containerd?
	return time.Now()
}

func (ctr *container) SetRootFs(snap runtime.Snapshot) error {
	fmt.Println("container.SetRootFs")
	return errdefs.NotImplemented()
}

func (ctr *container) Create() error {
	fmt.Println("container.Create")
	return errdefs.NotImplemented()
}

func (ctr *container) UpdateSpec(newSpec *runspecs.Spec) error {
	fmt.Println("container.UpdateSpec")
	return errdefs.NotImplemented()
}

func (ctr *container) Commit(gen [16]byte) error {
	fmt.Println("container.Commit")
	return errdefs.NotImplemented()
}

func (ctr *container) Snapshot() (runtime.Snapshot, error) {

	fmt.Println("container.Snapshot")
	return nil, errdefs.NotImplemented()
}

func (ctr *container) Amend() (runtime.Snapshot, error) {

	fmt.Println("container.Amend")
	return nil, errdefs.NotImplemented()
}

func (ctr *container) Exec(stream runtime.Stream,
	procSpec *runspecs.Process) (runtime.Process, error) {

	fmt.Println("container.Exec")

	return &process{
		container: nil,
	}, nil
}

func (ctr *container) Processes() ([]runtime.Process, error) {
	return nil, errdefs.NotImplemented()
}

func (ctr *container) Delete() error {
	fmt.Println("container.Delete")
	return errdefs.NotImplemented()
}

func (ctr *container) Purge() error {
	fmt.Println("container.Purge")
	return errdefs.NotImplemented()
}
