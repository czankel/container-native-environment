// Package container manages the containers of workspaces

package container

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

type Container struct {
	runContainer runtime.Container `output:"-"`
	Name         string
	CreatedAt    time.Time
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

func containerName(runCtr runtime.Container) string {

	dom := runCtr.Domain()
	cid := runCtr.ID()
	gen := runCtr.Generation()

	return hex.EncodeToString(dom[:]) + "-" +
		hex.EncodeToString(cid[:]) + "-" +
		hex.EncodeToString(gen[:])
}

// Containers returns all active containers in the project.
func Containers(run runtime.Runtime, prj *project.Project) ([]Container, error) {

	dom, err := uuid.Parse(prj.UUID)
	if err != nil {
		return nil, errdefs.InvalidArgument("invalid project UUID: '%v'", prj.UUID)
	}

	runCtrs, err := run.Containers(dom)
	if err != nil {
		return nil, err
	}
	ctrs := make([]Container, len(runCtrs))
	for i, c := range runCtrs {
		ctrs[i] = Container{
			runContainer: c,
			Name:         containerName(c),
			CreatedAt:    c.CreatedAt(),
		}
	}

	return ctrs, nil
}

// Find looks up the container and returns it or nil if it doesn't exist
func Find(run runtime.Runtime, ws *project.Workspace) (*Container, error) {

	dom, err := uuid.Parse(ws.ProjectUUID)
	if err != nil {
		return nil, errdefs.InvalidArgument(
			"invalid project UUID in workspace: '%v'", ws.ProjectUUID)
	}

	runCtr, err := findContainer(run, dom, ws.ID(), ws.ConfigHash())
	if err != nil || runCtr == nil {
		return nil, err
	}
	return &Container{
		runContainer: runCtr,
		Name:         containerName(runCtr),
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
	spec, err := DefaultSpec(run.Namespace(), ctrID, ws.Environment.Capabilities)
	if err != nil {
		return nil, err
	}

	runCtr, err := run.NewContainer(dom, cid, gen, img, &spec)
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

	// start the actual container (snap may be nil)
	err = runCtr.Start(nil, true)
	if err != nil {
		return nil, err
	}

	// build new image: execute in the current layer
	for layIdx := 0; layIdx < len(ws.Environment.Layers); layIdx++ {

		layer := &ws.Environment.Layers[layIdx]
		for _, line := range layer.Commands {

			if progress != nil {
				layerStatus[layIdx].Status = runtime.StatusRunning
				layerStatus[layIdx].Details = line
				stat := []runtime.ProgressStatus{layerStatus[layIdx]}
				progress <- stat
			}

			stream := runtime.Stream{}
			cmdArgs := strings.Split(line, " ")
			process, err := runCtr.Exec(stream, cmdArgs)
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

	return &Container{runContainer: runCtr, Name: containerName(runCtr)}, nil
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

	proc, err := ctr.runContainer.Exec(stream, cmd)
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
