// Package containerd implements the runtime interface for the ContainerD Dameon containerd.io
package containerd

import (
	"encoding/hex"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	ctrderr "github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/snapshots"

	"github.com/opencontainers/image-spec/identity"
	runspecs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/containerd/console"
	"github.com/google/uuid"

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

// createTask creates a new task
func createTask(ctr *container, mounts []mount.Mount) (containerd.Task, error) {

	ctrdRun := ctr.ctrdRuntime
	ctrdCtx := ctrdRun.context

	ctrdTask, err := ctr.ctrdContainer.NewTask(ctrdCtx,
		cio.NewCreator(cio.WithStdio),
		containerd.WithRootFS(mounts))
	if err != nil {
		ctr.ctrdContainer.Delete(ctrdCtx)
		ctr.ctrdContainer = nil
		return nil, runtime.Errorf("failed to create container task: %v", err)
	}

	return ctrdTask, nil
}

func deleteTask(ctrdRun *containerdRuntime, ctrdCtr containerd.Container) error {

	ctrdTask, err := ctrdCtr.Task(ctrdRun.context, nil)

	if err != nil && err != ctrderr.ErrNotFound {
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

func (ctr *container) Start(snapshot runtime.Snapshot, mutable bool) error {

	ctrdCtr := ctr.ctrdContainer
	ctrdRun := ctr.ctrdRuntime
	ctrdCtx := ctrdRun.context

	_, err := ctrdCtr.Task(ctrdRun.context, nil)
	if err != nil && !ctrderr.IsNotFound(err) {
		return runtime.Errorf("failed to check for existing task: %v", err)
	}
	if err == nil || !ctrderr.IsNotFound(err) {
		return nil
	}

	var mounts []mount.Mount

	snapName := composeID(ctr.domain, ctr.id)
	snapSVC := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)

	// check if the snapshot already exists
	info, err := snapSVC.Stat(ctrdCtx, snapName)
	if err == nil && (info.Kind == snapshots.KindActive && mutable ||
		info.Kind == snapshots.KindView && !mutable) {
		mounts, err = snapSVC.Mounts(ctrdCtx, snapName)
		if err != nil {
			return runtime.Errorf("failed to get snapshot mounts: %v", err)
		}
	}

	// otherwise, create snapshot
	if mounts == nil {

		var parentSnap string
		if snapshot != nil {
			parentSnap = snapshot.Name()
		} else {
			diffIDs, err := ctr.image.ctrdImage.RootFS(ctrdCtx)
			if err != nil {
				return runtime.Errorf("failed to get rootfs: %v", err)
			}
			parentSnap = identity.ChainID(diffIDs).String()
		}

		if mutable {
			mounts, err = snapSVC.Prepare(ctrdCtx, snapName, parentSnap)
		} else {
			mounts, err = snapSVC.View(ctrdCtx, snapName, parentSnap)
		}
		if err != nil {
			return runtime.Errorf("failed to create snapshot '%s': %v", snapName, err)
		}
	}

	_, err = createTask(ctr, mounts)
	return err
}

func (ctr *container) Stop(force bool) error {

	ctrdRun := ctr.ctrdRuntime
	ctrdCtr := ctr.ctrdContainer
	_, err := ctrdCtr.Task(ctrdRun.context, nil)
	if err != nil && !ctrderr.IsNotFound(err) {
		return runtime.Errorf("failed to check for existing task: %v", err)
	}
	if err != nil && ctrderr.IsNotFound(err) {
		return nil
	}

	return errdefs.NotImplemented()
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
func (ctr *container) Exec(stream runtime.Stream, cmd []string) (runtime.Process, error) {

	ctrdRun := ctr.ctrdRuntime
	ctrdCtr := ctr.ctrdContainer
	ctrdCtx := ctrdRun.context
	ctrdTask, err := ctrdCtr.Task(ctrdCtx, nil)
	if err != nil {
		return nil, runtime.Errorf("failed to get task: %v", err)
	}

	spec, err := ctrdCtr.Spec(ctrdCtx)
	if err != nil {
		return nil, runtime.Errorf("failed to get container spec: %v", err)
	}

	procSpec := spec.Process
	procSpec.Terminal = stream.Terminal
	procSpec.Args = cmd

	con := console.Current()
	defer con.Reset()

	if err := con.SetRaw(); err != nil {
		return nil, runtime.Errorf("failed to set terminal: %v", err)
	}
	ws, err := con.Size()
	con.Resize(ws)

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

func deleteContainer(ctrdRun *containerdRuntime, ctrdCtr containerd.Container) error {

	err := deleteTask(ctrdRun, ctrdCtr)
	if err != nil {
		return runtime.Errorf("failed to delete task: %v", err)
	}

	err = ctrdCtr.Delete(ctrdRun.context)
	if err != nil {
		return runtime.Errorf("failed to delete container: %v", err)
	}
	return nil
}

func (ctr *container) Delete() error {
	return deleteContainer(ctr.ctrdRuntime, ctr.ctrdContainer)
}
