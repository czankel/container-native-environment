// Package containerd implements the runtime interface for the ContainerD Dameon containerd.io
package containerd

import (
	"encoding/hex"
	"errors"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	ctrderr "github.com/containerd/containerd/errdefs"

	runspecs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/google/uuid"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/runtime"
)

type container struct {
	domain        [16]byte
	id            [16]byte
	generation    [16]byte
	spec          runspecs.Spec
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

// composeCtrdID composes the containerd ID from the domain and container ID
func composeCtrdID(domain [16]byte, id [16]byte) string {
	return hex.EncodeToString(domain[:]) + "-" + hex.EncodeToString(id[:])
}

// getGeneration returns the generation from a containerD Container.
func getGeneration(ctrdRun *containerdRuntime, ctrdCtr containerd.Container) ([16]byte, error) {

	var gen [16]byte

	labels, err := ctrdCtr.Labels(ctrdRun.context)
	if err != nil {
		return [16]byte{}, runtime.Errorf("failed to get generation: %v", err)
	}

	val := labels[containerdGenerationLabel]
	str, err := hex.DecodeString(val)
	if err != nil {
		return [16]byte{}, runtime.Errorf("failed to decode generation '%s': $v", val, err)
	}
	copy(gen[:], str)

	return gen, nil
}

// getGenerationString returns the generation of a containerD Container as a string.
func getGenerationString(ctrdRun *containerdRuntime, ctrdCtr containerd.Container) string {

	ctrdCtx := ctrdRun.context
	labels, err := ctrdCtr.Labels(ctrdCtx)
	if err != nil {
		return "<error>"
	}
	return labels[containerdGenerationLabel]
}

// getContainers returns all containers in the specified domain
func getContainers(ctrdRun *containerdRuntime, domain [16]byte) ([]runtime.Container, error) {

	var runCtrs []runtime.Container

	ctrdCtrs, err := ctrdRun.client.Containers(ctrdRun.context)
	if err != nil {
		return nil, runtime.Errorf("failed to get containers: %v", err)
	}

	for _, c := range ctrdCtrs {

		dom, id, err := splitCtrdID(c.ID())
		if err != nil {
			return nil, err
		}
		if dom != domain {
			continue
		}

		gen, err := getGeneration(ctrdRun, c)
		if err != nil {
			return nil, err
		}

		img, err := c.Image(ctrdRun.context)
		if err != nil {
			return nil, runtime.Errorf("failed to get image: %v", err)
		}

		spec, err := c.Spec(ctrdRun.context)
		if err != nil {
			return nil, runtime.Errorf("failed to get image spec: %v", err)
		}

		ctr := newContainer(ctrdRun, c, domain, id, gen, &image{ctrdRun, img}, spec)
		if err != nil {
			return nil, err
		}

		runCtrs = append(runCtrs, ctr)
	}
	return runCtrs, nil
}

// newContainer defines a new container without creating it.
func newContainer(ctrdRun *containerdRuntime, ctrdCtr containerd.Container,
	domain, id, generation [16]byte, img *image, spec *runspecs.Spec) *container {

	return &container{
		domain:        domain,
		id:            id,
		generation:    generation,
		image:         img,
		spec:          *spec,
		ctrdRuntime:   ctrdRun,
		ctrdContainer: ctrdCtr,
	}
}

// getContainer looks up the container by domain, id, and generation. It returns not-found
// error if the container doesn't exist.
//
// Note that the container must be 'Exec-able', so a not-found error will also be returned if
// no valid active snapshot exists and the container will be deleted.
func getContainer(ctrdRun *containerdRuntime, domain, id, generation [16]byte) (*container, error) {

	ctrdID := composeCtrdID(domain, id)
	ctrdCtr, err := ctrdRun.client.LoadContainer(ctrdRun.context, ctrdID)
	if err != nil && ctrderr.IsNotFound(err) {
		return nil, errdefs.NotFound("container", ctrdID)
	}
	if err != nil {
		return nil, runtime.Errorf("failed to get container: %v", err)
	}

	ctrdGen, err := getGeneration(ctrdRun, ctrdCtr)
	if err != nil {
		return nil, err
	}

	if ctrdGen != generation {
		return nil, errdefs.NotFound("container", ctrdID)
	}

	_, err = getActiveSnapshot(ctrdRun, domain, id)
	if err != nil && errors.Is(err, errdefs.ErrNotFound) {
		deleteContainer(ctrdRun, ctrdCtr, ctrdID) // ignore error
		return nil, errdefs.NotFound("container", ctrdID)
	}
	if err != nil {
		return nil, err
	}

	img, err := ctrdCtr.Image(ctrdRun.context)
	if err != nil {
		return nil, runtime.Errorf("failed to get image: %v", err)
	}

	spec, err := ctrdCtr.Spec(ctrdRun.context)
	if err != nil {
		return nil, runtime.Errorf("failed to get image spec: %v", err)
	}

	ctr := newContainer(ctrdRun, ctrdCtr, domain, id, generation, &image{ctrdRun, img}, spec)

	return ctr, nil
}

// createTask creates a new task for the active snapshot
func createTask(ctr *container) (containerd.Task, error) {

	ctrdRun := ctr.ctrdRuntime
	ctrdCtx := ctrdRun.context

	mounts, err := getActiveSnapMounts(ctrdRun, ctr.domain, ctr.id)
	if err != nil {
		return nil, err
	}

	ctrdTask, err := ctr.ctrdContainer.NewTask(ctrdCtx, cio.NewCreator(),
		containerd.WithRootFS(mounts))
	if err != nil {
		ctrID := composeCtrdID(ctr.domain, ctr.id)
		deleteContainer(ctrdRun, ctr.ctrdContainer, ctrID)
		ctr.ctrdContainer = nil
		return nil, runtime.Errorf("failed to create container task: %v", err)
	}

	return ctrdTask, nil
}

func deleteTask(ctrdRun *containerdRuntime, ctrdCtr containerd.Container) error {

	ctrdTask, err := ctrdCtr.Task(ctrdRun.context, nil)
	if err != nil && ctrderr.IsNotFound(err) {
		return nil
	} else if err != nil {
		return runtime.Errorf("failed to get container task: %v", err)
	}

	stat, err := ctrdTask.Status(ctrdRun.context)
	if err != nil {
		return runtime.Errorf("failed to get status for task: %v", err)
	}
	if stat.Status != containerd.Stopped {

		c, err := ctrdTask.Wait(ctrdRun.context)
		if err != nil {
			return runtime.Errorf("failed to wait for task: %v", err)
		}
		err = ctrdTask.Kill(ctrdRun.context, syscall.SIGKILL)
		if err != nil {
			return runtime.Errorf("failed to kill task: %v", err)
		}
		<-c
	}
	_, err = ctrdTask.Delete(ctrdRun.context)
	if err != nil && !ctrderr.IsNotFound(err) {
		return runtime.Errorf("failed to delete task: %v", err)
	}
	return nil
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

func (ctr *container) SetRootFs(snap runtime.Snapshot) error {
	return createActiveSnapshot(ctr.ctrdRuntime, ctr.image, ctr.domain, ctr.id, snap)
}

func (ctr *container) Create() error {

	ctrdRun := ctr.ctrdRuntime
	ctrdCtx := ctrdRun.context
	ctrdID := composeCtrdID(ctr.domain, ctr.id)
	gen := hex.EncodeToString(ctr.generation[:])

	// if a container with a different generation exists, delete that container
	ctrdCtr, err := ctrdRun.client.LoadContainer(ctrdRun.context, ctrdID)
	if err != nil && !ctrderr.IsNotFound(err) {
		return err
	}
	if err == nil {
		ctr.ctrdContainer = ctrdCtr
		labels, err := ctrdCtr.Labels(ctrdCtx)
		if err != nil {
			return err
		}
		ctrdGen := labels[containerdGenerationLabel]
		if ctrdGen == gen {
			return errdefs.AlreadyExists("container", ctrdID)
		}
		err = deleteContainer(ctrdRun, ctr.ctrdContainer, ctrdID)
		if err != nil {
			return err
		}
	}

	// update any incomplete spec
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
	uuidName := composeCtrdID(ctr.domain, ctr.id)
	labels := map[string]string{}
	labels[containerdGenerationLabel] = gen

	ctrdCtr, err = ctrdRun.client.NewContainer(ctrdRun.context, uuidName,
		containerd.WithImage(ctr.image.ctrdImage),
		containerd.WithSpec(&spec),
		containerd.WithRuntime("io.containerd.runtime.v1.linux", nil),
		containerd.WithContainerLabels(labels))
	if err != nil {
		return runtime.Errorf("failed to create container: %v", err)
	}

	ctr.ctrdContainer = ctrdCtr
	return nil
}

func (ctr *container) Commit(gen [16]byte) error {

	ctx := ctr.ctrdRuntime.context

	labels, err := ctr.ctrdContainer.Labels(ctx)
	if err != nil {
		return err
	}

	labels[containerdGenerationLabel] = hex.EncodeToString(gen[:])
	_, err = ctr.ctrdContainer.SetLabels(ctx, labels)
	if err != nil {
		return err
	}

	return nil
}

// Exec executes the provided command.
func (ctr *container) Exec(stream runtime.Stream,
	procSpec *runspecs.Process) (runtime.Process, error) {

	ctrdRun := ctr.ctrdRuntime
	ctrdCtr := ctr.ctrdContainer
	ctrdCtx := ctrdRun.context

	ctrdTask, err := ctrdCtr.Task(ctrdCtx, nil)
	if err != nil && ctrderr.IsNotFound(err) {
		ctrdTask, err = createTask(ctr)
	}
	if err != nil {
		return nil, runtime.Errorf("failed to get task: %v", err)
	}

	cioOpts := []cio.Opt{cio.WithStreams(stream.Stdin, stream.Stdout, stream.Stderr)}
	if stream.Terminal {
		cioOpts = append(cioOpts, cio.WithTerminal)
	}

	ioCreator := cio.NewCreator(cioOpts...)
	execID := uuid.New()
	ctrdProc, err := ctrdTask.Exec(ctrdCtx, execID.String(), procSpec, ioCreator)
	if err != nil {
		return nil, runtime.Errorf("exec failed: %v", err)
	}

	err = ctrdProc.Start(ctrdCtx)
	if err != nil && !ctrderr.IsNotFound(err) {
		return nil, errdefs.NotFound("command", procSpec.Args[0])
	}
	if err != nil {
		return nil, runtime.Errorf("starting process failed: %v", err)
	}

	return &process{
		container: ctr,
		ctrdProc:  ctrdProc,
	}, nil
}

func (ctr *container) Processes() ([]runtime.Process, error) {
	return nil, errdefs.NotImplemented()
}

func deleteContainer(ctrdRun *containerdRuntime,
	ctrdCtr containerd.Container, ctrID string) error {

	err := deleteTask(ctrdRun, ctrdCtr)
	if err != nil {
		return runtime.Errorf("failed to delete task: %v", err)
	}

	err = ctrdCtr.Delete(ctrdRun.context)
	if err != nil {
		return runtime.Errorf("failed to delete container: %v", err)
	}

	err = deleteSnapshot(ctrdRun, ctrID)
	if err != nil {
		return runtime.Errorf("failed to delete snapshot: %v", err)
	}

	return nil
}

func (ctr *container) Delete() error {
	ctrID := composeCtrdID(ctr.domain, ctr.id)
	return deleteContainer(ctr.ctrdRuntime, ctr.ctrdContainer, ctrID)
}
