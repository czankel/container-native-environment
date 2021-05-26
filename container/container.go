// Package container manages the containers of workspaces

package container

import (
	"encoding/hex"
	"errors"
	"os"
	"os/signal"
	"strings"
	"time"

	runspecs "github.com/opencontainers/runtime-spec/specs-go"
	specs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/google/uuid"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

const MaxProgressOutputLength = 80

type Container struct {
	runRuntime   runtime.Runtime   `output:"-"`
	runContainer runtime.Container `output:"-"`
	Namespace    string
	Name         string
	Domain       [16]byte
	ID           [16]byte
	Generation   [16]byte
	CreatedAt    time.Time
}

// containerName is a helper function returning the unique name of a container consisting
// of the domain, container id, and generation.
func containerName(dom, cid, gen [16]byte) string {

	return hex.EncodeToString(dom[:]) + "-" +
		hex.EncodeToString(cid[:]) + "-" +
		hex.EncodeToString(gen[:])
}

// containerNameRunCtr is a helper function to extract the container name from a runtime Container.
func containerNameRunCtr(runCtr runtime.Container) string {

	dom := runCtr.Domain()
	cid := runCtr.ID()
	gen := runCtr.Generation()

	return containerName(dom, cid, gen)
}

// Containers returns all active containers in the project.
func Containers(run runtime.Runtime, prj *project.Project) ([]Container, error) {

	var domains [][16]byte

	if prj != nil {
		dom, err := uuid.Parse(prj.UUID)
		if err != nil {
			return nil, errdefs.InvalidArgument("invalid project UUID: '%v'", prj.UUID)
		}
		domains = [][16]byte{dom}
	}

	var ctrs []Container
	for _, dom := range domains {
		runCtrs, err := run.Containers(dom)
		if err != nil {
			return nil, err
		}
		for _, c := range runCtrs {
			cid := c.ID()
			ctrs = append(ctrs, Container{
				runContainer: c,
				Name:         containerNameRunCtr(c),
				Domain:       dom,
				ID:           cid,
				Generation:   c.Generation(),
				CreatedAt:    c.CreatedAt(),
			})
		}
	}

	return ctrs, nil
}

// Get looks up the current active Container for the specified Workspace.
func Get(run runtime.Runtime, ws *project.Workspace) (*Container, error) {

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

	name := containerNameRunCtr(runCtr)
	return &Container{
		runRuntime:   run,
		runContainer: runCtr,
		Namespace:    run.Namespace(),
		Name:         name,
		Domain:       runCtr.Domain(),
		ID:           cid,
		Generation:   gen,
		CreatedAt:    runCtr.CreatedAt(),
	}, nil
}

// NewContainer defines a new Container with a default generation value for the Workspace without
// the Layer configuration. The generation value will be updated through Commit().
func NewContainer(run runtime.Runtime,
	ws *project.Workspace, img runtime.Image) (*Container, error) {

	dom, err := uuid.Parse(ws.ProjectUUID)
	if err != nil {
		return nil, errdefs.InvalidArgument("invalid project UUID: '%v'", ws.ProjectUUID)
	}

	cid := ws.ID()
	gen := ws.BaseHash()
	ctrName := containerName(dom, cid, gen)

	// start with a base container
	spec, err := DefaultSpec(run.Namespace(), ctrName)
	if err != nil {
		return nil, err
	}

	runCtr, err := run.NewContainer(dom, cid, gen, img, &spec)
	if err != nil {
		return nil, err
	}

	return &Container{
		runRuntime:   run,
		runContainer: runCtr,
		Namespace:    run.Namespace(),
		Name:         ctrName,
		Domain:       dom,
		ID:           cid,
		Generation:   gen,
	}, nil
}

// Create creates the container after it has been defined and before it can be built.
func (ctr *Container) Create() error {

	runCtr := ctr.runContainer
	return runCtr.Create()
}

// find an existing top-most snapshot up to but excluding nextLayerIdx
// and return it with the layer index.
// Layer index 0 and snaphost nil means that there is no snapshot that matches
func findRootFs(ctr *Container,
	ws *project.Workspace, nextLayerIdx int) (int, runtime.Snapshot, error) {

	run := ctr.runRuntime

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
func (ctr *Container) Build(ws *project.Workspace, nextLayerIdx int,
	user *config.User, params *config.Parameters,
	progress chan []runtime.ProgressStatus, stream runtime.Stream) error {

	runCtr := ctr.runContainer

	if nextLayerIdx == -1 {
		nextLayerIdx = len(ws.Environment.Layers)
	}

	bldLayerIdx, rootFsSnap, err := findRootFs(ctr, ws, nextLayerIdx)
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
	procSpec := DefaultProcessSpec()
	for ; bldLayerIdx < nextLayerIdx; bldLayerIdx++ {

		layer := &ws.Environment.Layers[bldLayerIdx]
		for _, cmdgrp := range layer.Commands {

			for _, cmdline := range cmdgrp.Cmdlines {

				args, err := expandLine(cmdline, vars)
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

				procSpec.Args = args
				procSpec.User.UID = user.BuildUID
				procSpec.User.GID = user.BuildGID
				process, err := runCtr.Exec(stream, &procSpec)
				if err != nil {
					runCtr.Delete()
					return err
				}

				c, err := process.Wait()
				if err == nil {
					exitStatus := <-c
					err = exitStatus.Error
					code := exitStatus.Code
					if code != 0 {
						err = errdefs.CommandFailed(args)
					}
				}
				if err != nil {
					runCtr.Purge() // ignore error
					return err
				}
			}
		}

		if err != nil {
			runCtr.Delete()
			return err
		}

		// create a snapshot for the bldent layer, ignore any errors
		snap, err := runCtr.Snapshot()
		if err == nil {
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

// Commit commits a container that has been built and updates its configuration
func (ctr *Container) Commit(ws *project.Workspace, user config.User, rootPath string) error {

	spec, err := DefaultSpec(ctr.Namespace, ctr.Name)
	if err != nil {
		return err
	}

	// Mount $HOME
	spec.Mounts = append(spec.Mounts, runspecs.Mount{
		Destination: user.HomeDir,
		Source:      user.HomeDir,
		Options:     []string{"rbind"},
	})

	err = ctr.runContainer.UpdateSpec(&spec)
	if err != nil {
		return err
	}

	runCtr := ctr.runContainer
	confHash := ws.ConfigHash()
	err = runCtr.Commit(confHash)
	if err != nil {
		return err
	}

	ctr.Generation = confHash
	return nil
}

// Exec excutes the provided command, using the default proces runtime spec.
// The user defines the current working directory and UID and GID.
// It uses the default environment from the calling process.
// I/O is defined by the provided stream.
// The container must be started before calling this function
func (ctr *Container) Exec(user *config.User, stream runtime.Stream, cmd []string) (uint32, error) {

	procSpec := DefaultProcessSpec()
	procSpec.Cwd = user.Pwd
	procSpec.User.UID = user.UID
	procSpec.User.GID = user.GID
	procSpec.Args = cmd

	// TODO: have a mechanism to permit or disallow sudo, i.e. 'sudo cne'
	allowSudo := true
	if user.IsSudo {
		if !allowSudo {
			return 0, errdefs.InvalidArgument("sudo not allowed")
		}
		procSpec.User.UID = 0
	}

	return commonExec(ctr, &procSpec, stream)
}

func (ctr *Container) BuildExec(user *config.User, stream runtime.Stream,
	cmd []string) (uint32, error) {

	procSpec := DefaultProcessSpec()
	procSpec.User.UID = user.BuildUID
	procSpec.User.GID = user.BuildGID
	procSpec.Args = cmd
	return commonExec(ctr, &procSpec, stream)
}

func commonExec(ctr *Container, procSpec *specs.Process, stream runtime.Stream) (uint32, error) {

	runCtr := ctr.runContainer

	procSpec.Env = os.Environ()
	procSpec.Terminal = stream.Terminal

	proc, err := runCtr.Exec(stream, procSpec)
	if err != nil && errors.Is(err, errdefs.ErrNotFound) && errdefs.Resource(err) == "command" {
		return 0, err
	}
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

func (ctr *Container) Delete() error {
	return ctr.runContainer.Delete()
}

// Purge deletes the container if not already deleted and also all associated Snapshots.
func (ctr *Container) Purge() error {
	return ctr.runContainer.Purge()
}
