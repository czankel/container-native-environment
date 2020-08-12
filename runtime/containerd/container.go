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

func (ctr *container) Create() error {

	ctrdRun := ctr.ctrdRuntime

	ctrdCtrs, err := ctrdRun.client.Containers(ctrdRun.context)
	if err != nil {
		return runtime.Errorf("failed to create container: %v", err)
	}

	// if a container with a different generation exists, delete that container
	for _, c := range ctrdCtrs {

		dom, id, err := splitCtrdID(c.ID())
		if err != nil {
			return err
		}
		if dom == ctr.domain && id == ctr.id {
			deleteContainer(ctrdRun, c)
			break
		}
	}

	// update imcomplete spec
	spec := ctr.spec
	if spec.Process == nil {
		spec.Process = &runspecs.Process{}
	}

	config, err := ctr.image.Config()
	if err != nil {
		return runtime.Errorf("failed to get image OCI spec: %v", err)
	}
	if spec.Linux != nil {
		spec.Process.Args = append(config.Entrypoint, config.Cmd...)
		cwd := config.WorkingDir
		if cwd == "" {
			cwd = "/"
		}
		spec.Process.Cwd = cwd
	}

	// create container
	uuidName := composeID(ctr.domain, ctr.id)
	labels := map[string]string{}
	gen := hex.EncodeToString(ctr.generation[:])
	labels[containerdGenerationLabel] = gen

	ctrdCtr, err := ctrdRun.client.NewContainer(ctrdRun.context, uuidName,
		containerd.WithImage(ctr.image.ctrdImage),
		containerd.WithSpec(spec),
		containerd.WithRuntime("io.containerd.runtime.v1.linux", nil),
		containerd.WithContainerLabels(labels))
	if err != nil {
		return runtime.Errorf("failed to create container: %v", err)
	}

	ctr.ctrdContainer = ctrdCtr
	return nil
}

func deleteContainer(ctrdRun *containerdRuntime, ctrdCtr containerd.Container) error {

	err := ctrdCtr.Delete(ctrdRun.context)
	if err != nil {
		return runtime.Errorf("failed to delete container: %v", err)
	}
	return nil
}

func (ctr *container) Delete() error {
	return deleteContainer(ctr.ctrdRuntime, ctr.ctrdContainer)
}
