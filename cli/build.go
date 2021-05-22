package cli

import (
	"errors"
	"sync"

	"github.com/spf13/cobra"

	"github.com/containerd/console"

	"github.com/czankel/cne/container"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

// createContainer defines and creates a new container
func createContainer(run runtime.Runtime, ws *project.Workspace) (*container.Container, error) {

	if ws.Environment.Origin == "" {
		return nil, errdefs.InvalidArgument("Workspace has no image defined")
	}

	// check and pull the image, if required, for building the container
	img, err := run.GetImage(ws.Environment.Origin)
	if err != nil && errors.Is(err, errdefs.ErrNotFound) {
		img, err = pullImage(run, ws.Environment.Origin)
	}
	if err != nil {
		return nil, err
	}

	ctr, err := container.NewContainer(run, ws, img)
	if err != nil {
		return nil, err
	}

	err = ctr.Create()
	if err != nil && errors.Is(err, errdefs.ErrAlreadyExists) {
		run.DeleteContainer(ctr.Domain, ctr.ID, ctr.Generation)
		err = ctr.Create()
	}
	if err != nil {
		return nil, err
	}

	return ctr, err
}

// buildLayers builds the layers of a container and outputs progress status.
// The nextLayerIdx argument defines the layer index following the one that should be built,
// with 0 meaning no layer should be built.  Use -1 or len(layers) to build all layers.
// This function is idempotent and can be called again to continue the build, for example,
// for a higher layer.
// Note that in an error case, it will keep any residual container and snapshots.
func buildLayers(run runtime.Runtime, ctr *container.Container,
	ws *project.Workspace, nextLayerIdx int) error {

	con := console.Current()
	defer con.Reset()

	// build the container and provide progress output
	var wg sync.WaitGroup

	wg.Add(1)

	progress := make(chan []runtime.ProgressStatus)
	go func() {
		defer wg.Done()
		showBuildProgress(progress)
	}()

	// TODO: --------------------------------------------------
	// TODO: running as root inside the container during build!
	// TODO: --------------------------------------------------
	user.BuildUID = 0
	user.BuildGID = 0

	stream := runtime.Stream{}

	err := ctr.Build(ws, nextLayerIdx, &user, progress, stream)
	if err != nil && errors.Is(err, errdefs.ErrCommandFailed) {
		wg.Wait()
		return err
	}
	if err != nil {
		wg.Wait()
		return err
	}
	wg.Wait()

	return nil
}

// commitContainer commits the container
func commitContainer(ctr *container.Container, ws *project.Workspace) error {
	return ctr.Commit(ws, user, ws.Path)
}

// buildContainer builds the container for the provided workspace and outputs progress status.
// Note that in an error case, it will keep any residual container and snapshots.
func buildContainer(run runtime.Runtime, ws *project.Workspace) (*container.Container, error) {

	ctr, err := createContainer(run, ws)
	if err != nil {
		return nil, err
	}

	err = buildLayers(run, ctr, ws, -1)
	if err != nil {
		return nil, err
	}

	err = commitContainer(ctr, ws)
	if err != nil {
		return nil, err
	}

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

	ctr, err := container.Get(run, ws)
	if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
		return err
	}
	if ctr != nil && buildWorkspaceForce {
		err = ctr.Purge()
		if err != nil {
			return err
		}
	}

	_, err = buildContainer(run, ws)
	return err
}

func init() {

	rootCmd.AddCommand(buildCmd)
	buildCmd.AddCommand(buildWorkspaceCmd)
	buildWorkspaceCmd.Flags().BoolVar(
		&buildWorkspaceForce, "force", false, "Force a rebuild of the container")
}
