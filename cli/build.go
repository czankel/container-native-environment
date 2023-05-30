package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/spf13/cobra"

	"github.com/containerd/console"

	"github.com/czankel/cne/container"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

const outputLineLength = 200
const outputLineCount = 100

// getContainer returns the existing active container for the workspace or
// creates the container and outputs progress status.
// Note that in an error case, it will keep any residual container and snapshots.
func getContainer(ctx context.Context,
	run runtime.Runtime, ws *project.Workspace) (runtime.Container, error) {

	// check if container already exists
	ctr, err := container.Get(ctx, run, ws)
	if err == nil {
		return ctr, nil
	}

	// check and pull the image, if required, for building the container
	if ws.Environment.Origin == "" {
		return nil, errdefs.InvalidArgument("Workspace has no image defined")
	}

	// check and pull the image, if required, for building the container
	img, err := run.GetImage(ctx, ws.Environment.Origin)
	if err != nil && errors.Is(err, errdefs.ErrNotFound) {
		img, err = pullImage(ctx, run, ws.Environment.Origin)
	}
	if err != nil {
		return nil, err
	}

	cfgCtx, err := conf.GetContext()
	if err != nil {
		return nil, err
	}

	return container.NewContainer(ctx, run, ws, &user, img, cfgCtx.Options)
}

// buildLayers builds the layers of a container and outputs progress status.
// The layerCount argument defines the number of layers that should have been built.
// with 0 meaning no layer should be built.  Use -1 or len(layers) to build all layers.
// This function is idempotent and can be called again to continue the build, for example,
// for a higher layer.
// Note that in an error case, it will keep any residual container and snapshots.
func buildLayers(ctx context.Context, run runtime.Runtime, ctr runtime.Container,
	ws *project.Workspace, layerCount int) error {

	// FIXME: why and does this interfere with e.g. cli/exec?
	con := console.Current()
	defer con.Reset()

	// build the container and provide progress output
	var wg sync.WaitGroup
	wg.Add(1)

	progress := make(chan []runtime.ProgressStatus)
	go func() {
		defer wg.Done()
		showProgress(progress)
	}()

	rb := NewRingBuffer(outputLineCount, outputLineLength)
	stream := rb.StreamWriter()

	err := container.Build(ctx, ctr, ws, layerCount, &user, &params, progress, stream)
	if err != nil && errors.Is(err, errdefs.ErrCommandFailed) {
		line := make([]byte, 100)
		fmt.Printf("Output:\n")
		for _, err := rb.Read(line); err != io.EOF; _, err = rb.Read(line) {
			fmt.Printf(" > %v\n", string(line))
		}
		wg.Wait()
		return err
	}
	if err != nil {
		wg.Wait() // FIXME: just clear??
		return err
	}
	wg.Wait()

	return nil
}

// buildContainer builds the full container for the provided workspace and
// commits it.
func buildContainer(ctx context.Context, run runtime.Runtime, ws *project.Workspace,
	layerCount int) (runtime.Container, error) {

	params.Upgrade = buildWorkspaceUpgrade
	ctr, err := getContainer(ctx, run, ws)
	if err != nil {
		return nil, err
	}

	err = buildLayers(ctx, run, ctr, ws, layerCount)
	if err != nil {
		return nil, err
	}

	// Mount $HOME
	err = ctr.Mount(ctx, user.HomeDir, user.HomeDir)
	if err != nil {
		return nil, err
	}

	err = ctr.Commit(ctx, ws.ConfigHash())
	return ctr, err
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
var buildWorkspaceUpgrade string

func buildWorkspaceRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
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

	ctx := context.Background()
	runCfg, err := conf.GetRuntime()
	if err != nil {
		return err
	}

	run, err := runtime.Open(ctx, runCfg)
	if err != nil {
		return err
	}

	defer run.Close()
	ctx = run.WithNamespace(ctx, runCfg.Namespace)

	// only allow a single build container at a time
	ctr, err := container.Get(ctx, run, ws)
	if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
		return err
	}
	if err == nil {
		if !buildWorkspaceForce && buildWorkspaceUpgrade == "" {
			return errdefs.AlreadyExists("container", ctr.Name())
		}
		err = ctr.Purge(ctx)
		if err != nil {
			return err
		}
	}

	_, err = buildContainer(ctx, run, ws, -1)
	if err != nil {
		return err
	}

	return prj.Write()
}

func init() {

	rootCmd.AddCommand(buildCmd)
	buildCmd.AddCommand(buildWorkspaceCmd)
	buildWorkspaceCmd.Flags().BoolVar(
		&buildWorkspaceForce, "force", false, "Force a rebuild of the container")
	buildWorkspaceCmd.Flags().StringVar(
		&buildWorkspaceUpgrade, "upgrade", "", "Upgrade image, apt, all")
}
