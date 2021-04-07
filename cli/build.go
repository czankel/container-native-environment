package cli

import (
	"errors"
	"sync"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/container"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

// buildContainer builds the container for the provided workspace and outputs progress status
func buildContainer(run runtime.Runtime,
	prj *project.Project, ws *project.Workspace) (*container.Container, error) {

	if ws.Environment.Origin == "" {
		return nil, errdefs.InvalidArgument("Workspace has not image defined")
	}

	// check and pull the image, if required, for building the container
	img, err := run.GetImage(ws.Environment.Origin)
	if err != nil && errors.Is(err, errdefs.ErrNotFound) {
		img, err = pullImage(run, ws.Environment.Origin)
	}
	if err != nil {
		return nil, err
	}

	// build the container and provide progress output
	var wg sync.WaitGroup

	wg.Add(1)

	progress := make(chan []runtime.ProgressStatus)
	go func() {
		defer wg.Done()
		showBuildProgress(progress)
	}()
	ctr, err := container.Create(run, ws, img, progress)
	if err != nil {
		return nil, err
	}
	wg.Wait()

	return ctr, nil
}

var buildCmd = &cobra.Command{
	Use:     "build",
	Short:   "Build or rebuild an object",
	Aliases: []string{"b"},
	Args:    cobra.MinimumNArgs(1),
}

var buildWorkspaceCmd = &cobra.Command{
	Use:     "workspace [NAME]",
	Short:   "Manually build or rebuild the current or specified workspace",
	Aliases: []string{"ws"},
	Args:    cobra.MaximumNArgs(1),
	RunE:    buildWorkspaceRunE,
}

var buildWorkspaceForce bool

func buildWorkspaceRunE(cmd *cobra.Command, args []string) error {

	prj, err := project.Load()
	if err != nil {
		return err
	}

	ws, err := prj.CurrentWorkspace()
	if len(args) != 0 {
		ws, err = prj.Workspace(args[0])
	}
	if err != nil {
		return err
	}

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	ctr, err := container.Find(run, ws)
	if err != nil {
		return err
	}
	if ctr != nil && buildWorkspaceForce {
		err = ctr.Delete()
		if err != nil {
			return err
		}
	}

	_, err = buildContainer(run, prj, ws)
	return err
}

func init() {

	rootCmd.AddCommand(buildCmd)
	buildCmd.AddCommand(buildWorkspaceCmd)
	buildWorkspaceCmd.Flags().BoolVar(
		&buildWorkspaceForce, "force", false, "Force a rebuild of the container")
}
