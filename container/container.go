// Package container manages the containers of workspaces

package container

import (
	"errors"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/google/uuid"

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
func Containers(run runtime.Runtime, prj *project.Project, user *config.User) ([]runtime.Container, error) {

	var domain [16]byte
	var err error
	if prj != nil {
		domain, err = uuid.Parse(prj.UUID)
		if err != nil {
			return nil, errdefs.InvalidArgument("invalid project UUID: '%v'", prj.UUID)
		}
	}

	var ctrs []runtime.Container
	runCtrs, err := run.Containers()
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
func Get(run runtime.Runtime, ws *project.Workspace) (runtime.Container, error) {

	dom, err := uuid.Parse(ws.ProjectUUID)
	if err != nil {
		return nil, errdefs.InvalidArgument(
			"invalid project UUID in workspace: '%v'", ws.ProjectUUID)
	}

	cid := ws.ID()
	gen := ws.ConfigHash()
	runCtr, err := run.GetContainer(dom, cid, gen)
	if err != nil {
		return nil, err
	}

	return runCtr, nil
}

// NewContainer defines a new Container with a default generation value for the Workspace without
// the Layer configuration. The generation value will be updated through Commit().
func NewContainer(run runtime.Runtime, user *config.User,
	ws *project.Workspace, img runtime.Image) (runtime.Container, error) {

	dom, err := uuid.Parse(ws.ProjectUUID)
	if err != nil {
		return nil, errdefs.InvalidArgument("invalid project UUID: '%v'", ws.ProjectUUID)
	}

	cid := ws.ID()
	gen := ws.BaseHash()
	runCtr, err := run.NewContainer(dom, cid, gen, user.UID, img)
	if err != nil {
		return nil, err
	}

	return runCtr, nil
}

// find an existing top-most snapshot up to but excluding nextLayerIdx
// and return it with the layer index.
// Layer index 0 and snaphost nil means that there is no snapshot that matches
func findRootFs(runCtr runtime.Container,
	ws *project.Workspace, nextLayerIdx int) (int, runtime.Snapshot, error) {

	run := runCtr.Runtime()

	// identify the layer with the topmost existing snapshot
	bldLayerIdx := 0
	var snap runtime.Snapshot

	snaps, err := run.Snapshots()
	if err != nil {
		return -1, nil, err
	}

	for i := 0; i < nextLayerIdx; i++ {
		if l := ws.Environment.Layers[i]; l.Digest != "" {
			for _, s := range snaps {
				if l.Digest == s.Name() {
					bldLayerIdx = i + 1
					snap = s
					break
				}
			}
		}
	}

	return bldLayerIdx, snap, err
}

// Build builds the container.
//
// If a nextLayerIdx is provided, the build stops at the specified layer. Use 0 to exclude all
// layers and -1 or len(layers) to build all layers.
// A container may already be partially built. In that case, Build() will continue the build
// process.
// The progress argument is optional for outputting status updates during the build process.
func Build(runCtr runtime.Container, ws *project.Workspace, nextLayerIdx int,
	user *config.User, params *config.Parameters,
	progress chan []runtime.ProgressStatus, stream runtime.Stream) error {

	if nextLayerIdx == -1 {
		nextLayerIdx = len(ws.Environment.Layers)
	}

	bldLayerIdx, rootFsSnap, err := findRootFs(runCtr, ws, nextLayerIdx)
	if err != nil {
		return err
	}

	// prep the progress status updates
	defer func() {
		if progress != nil {
			close(progress)
		}
	}()
	layerStatus := make([]runtime.ProgressStatus, len(ws.Environment.Layers))
	if progress != nil {
		for i, l := range ws.Environment.Layers {
			layerStatus[i].Reference = l.Name
			layerStatus[i].StartedAt = time.Now()
			layerStatus[i].UpdatedAt = time.Now()

			if i < bldLayerIdx {
				layerStatus[i].Status = runtime.StatusExists
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

	err = runCtr.SetRootFs(rootFsSnap)
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
	for ; bldLayerIdx < nextLayerIdx; bldLayerIdx++ {

		layer := &ws.Environment.Layers[bldLayerIdx]
		for _, command := range layer.Commands {

			args, err := expandLine(command.Args, vars)
			if err != nil {
				runCtr.Delete() // ignore error
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
				layerStatus[bldLayerIdx].Status = runtime.StatusRunning
				layerStatus[bldLayerIdx].Details = lineOut
				stat := []runtime.ProgressStatus{layerStatus[bldLayerIdx]}
				progress <- stat
			}

			code, err := BuildExec(runCtr, user, stream, args, command.Envs)
			if code != 0 {
				err = errdefs.CommandFailed(args)
			}
			if err != nil {
				runCtr.Delete()
				return err
			}
		}

		// create a snapshot for the layer
		layer.Digest = ""
		snap, err := runCtr.Snapshot()
		if err != nil &&
			!errors.Is(err, errdefs.ErrNotImplemented) &&
			!errors.Is(err, errdefs.ErrAlreadyExists) {
			runCtr.Delete()
			return err
		}
		if snap != nil {
			layer.Digest = snap.Name()
		}
		if progress != nil {
			layerStatus[bldLayerIdx].Status = runtime.StatusComplete
			stat := []runtime.ProgressStatus{layerStatus[bldLayerIdx]}
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
func Exec(runCtr runtime.Container, user *config.User, stream runtime.Stream, args []string) (uint32, error) {

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

	return commonExec(runCtr, &procSpec, stream)
}

func BuildExec(runCtr runtime.Container, user *config.User, stream runtime.Stream,
	args []string, envs []string) (uint32, error) {

	procSpec := runtime.ProcessSpec{
		UID:  user.BuildUID,
		GID:  user.BuildGID,
		Args: args,
		Env:  append(baseEnv, envs...),
	}
	return commonExec(runCtr, &procSpec, stream)
}

func commonExec(runCtr runtime.Container, procSpec *runtime.ProcessSpec, stream runtime.Stream) (uint32, error) {

	proc, err := runCtr.Exec(stream, procSpec)
	if err != nil {
		return 0, err
	}

	ch, err := proc.Wait()
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
			proc.Signal(s)
		}
	}()

	exitStat := <-ch
	signal.Stop(sigc)
	close(sigc)
	return exitStat.Code, exitStat.Error
}
