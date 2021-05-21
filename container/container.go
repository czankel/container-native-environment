// Package container manages the containers of workspaces

package container

import (
	"encoding/hex"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

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

// findContainer is a helper function to find and return the existing container for the provided
// domain, identifier, and generation. It returns nil if the container was not found.
func findContainer(run runtime.Runtime,
	domain [16]byte, id [16]byte, generation [16]byte) (runtime.Container, error) {

	runCtrs, err := run.Containers(domain)
	if err != nil {
		return nil, err
	}

	for _, c := range runCtrs {
		if c.ID() == id && c.Generation() == generation {
			return c, nil
		}
	}

	return nil, nil
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

// Create creates and builds a new container.
// The progress is optional for outputting status updates
func Create(run runtime.Runtime, ws *project.Workspace, img runtime.Image,
	progress chan []runtime.ProgressStatus) (*Container, error) {

	defer func() {
		if progress != nil {
			close(progress)
		}
	}()

	dom, err := uuid.Parse(ws.ProjectUUID)
	if err != nil {
		return nil, errdefs.InvalidArgument("invalid project UUID: '%v'", ws.ProjectUUID)
	}

	cid := ws.ID()
	gen := ws.ConfigHash()
	ctrID := hex.EncodeToString(dom[:]) + "-" +
		hex.EncodeToString(cid[:]) + "-" +
		hex.EncodeToString(gen[:])

	// start with the base container
	spec, err := DefaultSpec(run.Namespace(), ctrID)
	if err != nil {
		return nil, err
	}

	runCtr, err := run.NewContainer(dom, cid, gen, img, &spec)
	if err != nil {
		return nil, err
	}

	err = runCtr.SetRootFs(nil)
	if err != nil {
		return nil, err
	}

	// create the container
	err = runCtr.Create()
	if err != nil {
		return nil, err
	}

	// Prep the progress status updates
	layerStatus := make([]runtime.ProgressStatus, len(ws.Environment.Layers))
	if progress != nil {
		for i, l := range ws.Environment.Layers {
			layerStatus[i].Reference = l.Name
			layerStatus[i].Status = runtime.StatusPending
			layerStatus[i].Total = int64(len(l.Commands))
			layerStatus[i].StartedAt = time.Now()
			layerStatus[i].UpdatedAt = time.Now()
		}
		var stat []runtime.ProgressStatus
		copy(stat, layerStatus)
		progress <- stat
	}

	procSpec := DefaultProcessSpec()

	// build new image: execute in the current layer
	for layIdx := 0; layIdx < len(ws.Environment.Layers); layIdx++ {

		layer := &ws.Environment.Layers[layIdx]
		for _, cmdgrp := range layer.Commands {

			for _, cmdline := range cmdgrp.Cmdlines {

				args := cmdline

				if progress != nil {
					lineOut := "Executing: " + strings.Join(args, " ")
					if len(lineOut) > MaxProgressOutputLength {
						lineOut = lineOut[:MaxProgressOutputLength-4] + " ..."
					}
					layerStatus[layIdx].Status = runtime.StatusRunning
					layerStatus[layIdx].Details = lineOut
					stat := []runtime.ProgressStatus{layerStatus[layIdx]}
					progress <- stat
				}

				stream := runtime.Stream{}

				procSpec.Args = args
				process, err := runCtr.Exec(stream, &procSpec)
				if err != nil {
					runCtr.Delete()
					return nil, err
				}

				c, err := process.Wait()
				if err == nil {
					exitStatus := <-c
					err = exitStatus.Error
				}
				if err != nil {
					runCtr.Delete()
					return nil, err
				}
			}
		}

		if progress != nil {
			layerStatus[layIdx].Status = runtime.StatusComplete
			stat := []runtime.ProgressStatus{layerStatus[layIdx]}
			progress <- stat
		}
	}

	// commit the container
	err = runCtr.Commit(ws.ConfigHash())
	if err != nil {
		runCtr.Delete()
		return nil, err
	}

	return &Container{runContainer: runCtr, Name: containerName(dom, cid, gen)}, nil
}

// Delete deletes the container if it exists
// Note that this function does not return an error if the container doesn't exist
func Delete(run runtime.Runtime, ws *project.Workspace) error {

	dom, err := uuid.Parse(ws.ProjectUUID)
	if err != nil {
		return errdefs.InvalidArgument(
			"invalid project UUID in workspace: '%v'", ws.ProjectUUID)
	}

	ctr, err := findContainer(run, dom, ws.ID(), ws.ConfigHash())
	if err != nil || ctr == nil {
		return err
	}

	return ctr.Delete()
}

func (ctr *Container) Exec(stream runtime.Stream, cmd []string) (uint32, error) {

	procSpec := DefaultProcessSpec()
	procSpec.Env = os.Environ()
	procSpec.Terminal = stream.Terminal
	procSpec.Args = cmd
	proc, err := ctr.runContainer.Exec(stream, &procSpec)
	if err != nil {
		return 0, err
	}

	ch, err := proc.Wait()
	if err != nil {
		return 0, err
	}

	exitStat := <-ch
	return exitStat.Code, exitStat.Error
}

func (ctr *Container) Delete() error {
	return ctr.runContainer.Delete()
}
