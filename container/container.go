// Package container manages the containers of workspaces

package container

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/opencontainers/image-spec/identity"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

const MaxProgressOutputLength = 80

var baseEnv = []string{
	"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
}

// Containers returns all active containers in the project.
func Containers(ctx context.Context, run runtime.Runtime,
	prj *project.Project, user *config.User) ([]runtime.Container, error) {

	var domain [16]byte
	var err error
	if prj != nil {
		domain, err = uuid.Parse(prj.UUID)
		if err != nil {
			return nil, errdefs.InvalidArgument("invalid project UUID: '%v'", prj.UUID)
		}
	}

	var ctrs []runtime.Container
	runCtrs, err := run.Containers(ctx)
	if err != nil {
		return nil, err
	}

	for _, c := range runCtrs {
		dom := c.Domain()
		if prj != nil && dom != domain {
			continue
		}

		if !user.IsSudo && c.UID() != user.UID {
			continue
		}

		ctrs = append(ctrs, c)
	}

	return ctrs, nil
}

// Get looks up the current active Container for the specified Workspace.
func GetContainer(ctx context.Context,
	run runtime.Runtime, ws *project.Workspace) (runtime.Container, error) {

	dom, err := uuid.Parse(ws.ProjectUUID)
	if err != nil {
		return nil, errdefs.InvalidArgument(
			"invalid project UUID in workspace: '%v'", ws.ProjectUUID)
	}

	cid := ws.ID()
	gen := ws.ConfigHash()
	runCtr, err := run.GetContainer(ctx, dom, cid, gen)
	if err != nil {
		return nil, err
	}

	return runCtr, nil
}

// NewContainer defines a new Container with a default generation value for the Workspace without
// the Layer configuration. The generation value will be updated through Commit().
func NewContainer(ctx context.Context, run runtime.Runtime, ws *project.Workspace,
	user *config.User, img runtime.Image) (runtime.Container, error) {

	dom, err := uuid.Parse(ws.ProjectUUID)
	if err != nil {
		return nil, errdefs.InvalidArgument("invalid project UUID: '%v'", ws.ProjectUUID)
	}

	cid := ws.ID()
	gen := ws.BaseHash()
	runCtr, err := run.NewContainer(ctx, dom, cid, gen, user.UID)
	if err != nil {
		return nil, err
	}

	err = runCtr.Create(ctx, img)
	if err != nil && errors.Is(err, errdefs.ErrAlreadyExists) {
		runCtr.Delete(ctx)
		err = runCtr.Create(ctx, img)
	}
	if err != nil {
		return nil, err
	}

	return runCtr, err
}

// find RootFS looks up the top-most snapshot up to but excluding nextLayerIdx
// and returns the digest and layer index. ErrNotFound is returned if no snapshot was found.
func findRootFS(ctx context.Context, runCtr runtime.Container,
	ws *project.Workspace, nextLayerIdx int) (int, string, error) {

	// identify the layer with the topmost existing snapshot
	layerIdx := 0
	snaps, err := runCtr.Snapshots(ctx)
	if err != nil {
		return -1, "", err
	}

	var snapName string
	for i := 0; i < nextLayerIdx; i++ {
		if l := ws.Environment.Layers[i]; l.Digest != "" {
			for _, s := range snaps {
				if l.Digest == s.Name() {
					layerIdx = i + 1
					snapName = l.Digest
					break
				}
			}
		}
	}
	if layerIdx == 0 {
		return 0, "", errdefs.NotFound("snapshot", snapName)
	}

	return layerIdx, snapName, nil
}

// Build builds the container.
//
// A container may already be partially built. In that case, Build() will continue the build
// process.
//
// layerCount determines the number of layers built. Use 0 to only create the image and
// -1 or len(layers) to build all layers.
// The progress argument is optional for outputting status updates during the build process.
func Build(ctx context.Context, run runtime.Runtime, runCtr runtime.Container,
	img runtime.Image, ws *project.Workspace, layerCount int,
	user *config.User, params *config.Parameters,
	progress chan []runtime.ProgressStatus, stream runtime.Stream) error {

	defer close(progress)

	if layerCount == -1 {
		layerCount = len(ws.Environment.Layers)
	}

	layerIdx, name, err := findRootFS(ctx, runCtr, ws, layerCount)
	if err != nil && errors.Is(err, errdefs.ErrNotFound) {
		diffIDs, err := img.RootFS(ctx)
		if err != nil {
			return runtime.Errorf("failed to get rootfs: %v", err)
		}
		name = identity.ChainID(diffIDs).String()
		_, err = run.GetSnapshot(ctx, name)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	layerStatus := make([]runtime.ProgressStatus, len(ws.Environment.Layers))
	if progress != nil {
		for i, l := range ws.Environment.Layers {
			layerStatus[i].Reference = l.Name
			layerStatus[i].StartedAt = time.Now()
			layerStatus[i].UpdatedAt = time.Now()

			if i < layerIdx {
				layerStatus[i].Status = runtime.StatusCached
				layerStatus[i].Offset = layerStatus[i].Total
			} else {
				layerStatus[i].Status = runtime.StatusPending
				layerStatus[i].Total = int64(len(l.Commands))
			}
		}
		var stat []runtime.ProgressStatus
		copy(stat, layerStatus)
		progress <- stat
	}

	err = runCtr.SetRootFS(ctx, name)
	if err != nil {
		return err
	}

	vars := struct {
		Environment *project.Environment
		User        *config.User
		Parameters  *config.Parameters
	}{
		Environment: &ws.Environment,
		User:        user,
		Parameters:  params,
	}

	// build all remaining layers
	for ; layerIdx < layerCount; layerIdx++ {

		layer := &ws.Environment.Layers[layerIdx]
		for _, command := range layer.Commands {

			args, err := expandLine(command.Args, vars)
			if err != nil {
				runCtr.Delete(ctx) // ignore error
				return err
			}

			if len(args) == 0 {
				continue
			}

			if progress != nil {
				lineOut := "Executing: " + strings.Join(args, " ")
				if len(lineOut) > MaxProgressOutputLength {
					lineOut = lineOut[:MaxProgressOutputLength-4] + " ..."
				}
				layerStatus[layerIdx].Status = runtime.StatusRunning
				layerStatus[layerIdx].Details = lineOut
				stat := []runtime.ProgressStatus{layerStatus[layerIdx]}
				progress <- stat
			}
			code, err := BuildExec(ctx, runCtr, user, stream, args, command.Envs)
			if code != 0 {
				err = errdefs.CommandFailed(args)
			}
			if err != nil {
				runCtr.Delete(ctx)
				return err
			}
		}

		// create a snapshot for the layer
		layer.Digest = ""
		snap, err := runCtr.Snapshot(ctx)
		if err != nil &&
			!errors.Is(err, errdefs.ErrNotImplemented) &&
			!errors.Is(err, errdefs.ErrAlreadyExists) {
			runCtr.Delete(ctx)
			return err
		}
		if snap != nil {
			layer.Digest = snap.Name()
		}
		if progress != nil {
			layerStatus[layerIdx].Status = runtime.StatusComplete
			stat := []runtime.ProgressStatus{layerStatus[layerIdx]}
			progress <- stat
		}
	}

	return nil
}

// Exec excutes the provided command, using the default proces runtime spec.
// The user defines the current working directory and UID and GID.
// It uses the default environment from the calling process.
// I/O is defined by the provided stream.
// The container must be started before calling this function
func Exec(ctx context.Context, runCtr runtime.Container,
	user *config.User, stream runtime.Stream, args []string) (uint32, error) {

	procSpec := runtime.ProcessSpec{
		Cwd:  user.Pwd,
		UID:  user.UID,
		GID:  user.GID,
		Args: args,
		Env:  os.Environ(),
	}

	// TODO: have a mechanism to permit or disallow sudo, i.e. 'sudo cne'
	allowSudo := true
	if user.IsSudo {
		if !allowSudo {
			return 0, errdefs.InvalidArgument("sudo not allowed")
		}
		procSpec.UID = 0
	}

	return commonExec(ctx, runCtr, &procSpec, stream)
}

func BuildExec(ctx context.Context, runCtr runtime.Container,
	user *config.User, stream runtime.Stream,
	args []string, envs []string) (uint32, error) {

	procSpec := runtime.ProcessSpec{
		UID:  user.BuildUID,
		GID:  user.BuildGID,
		Args: args,
		Env:  append(baseEnv, envs...),
	}
	return commonExec(ctx, runCtr, &procSpec, stream)
}

func commonExec(ctx context.Context, runCtr runtime.Container,
	procSpec *runtime.ProcessSpec, stream runtime.Stream) (uint32, error) {

	proc, err := runCtr.Exec(ctx, stream, procSpec)
	if err != nil {
		return 0, err
	}

	ch, err := proc.Wait(ctx)
	if err != nil {
		return 0, err
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc)
	go func() {
		for {
			s, more := <-sigc
			if !more {
				return
			}
			proc.Signal(ctx, s)
		}
	}()

	exitStat := <-ch
	signal.Stop(sigc)
	close(sigc)
	return exitStat.Code, exitStat.Error
}
