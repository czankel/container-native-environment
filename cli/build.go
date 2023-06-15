package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/opencontainers/image-spec/identity"
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
	run runtime.Runtime, ws *project.Workspace) (runtime.Container, runtime.Image, error) {

	// check and pull the image, if required, for building the container
	if ws.Environment.Origin == "" {
		return nil, nil, errdefs.InvalidArgument("Workspace has no image defined")
	}

	img, err := run.GetImage(ctx, ws.Environment.Origin)
	if err != nil && errors.Is(err, errdefs.ErrNotFound) {
		img, err = pullImage(ctx, run, ws.Environment.Origin)
	}
	if err != nil {
		return nil, nil, err
	}

	diffIDs, err := img.RootFS(ctx)
	if err != nil {
		return nil, nil, err
	}

	rootName := identity.ChainID(diffIDs).String()
	_, err = run.GetSnapshot(ctx, rootName)
	if err != nil && errors.Is(err, errdefs.ErrNotFound) {
		progress := make(chan []runtime.ProgressStatus)
		var wg sync.WaitGroup
		go func() {
			defer wg.Done()
			wg.Add(1)
			showProgress(progress)
		}()
		err = img.Unpack(ctx, progress)
		wg.Wait()
	}
	if err != nil {
		return nil, nil, err
	}

	// check and return the container if it already exists
	ctr, err := container.GetContainer(ctx, run, ws)
	if err == nil {
		return ctr, img, nil
	}

	confCtx, _, err := conf.GetContext()
	if err != nil {
		return nil, nil, err
	}

	ctr, err = container.CreateContainer(ctx, run, ws, &user, img, confCtx.Options)
	return ctr, img, err
}

// buildLayers builds the layers of a container and outputs progress status.
//
// The layerCount argument defines the number of layers that should have been built.
// with 0 meaning no layer should be built.  Use -1 or len(layers) to build all layers.
// This function is idempotent and can be called again to continue the build, for example,
// for a higher layer.
// Note that in an error case, it will keep any residual container and snapshots.
func buildLayers(ctx context.Context, run runtime.Runtime, ctr runtime.Container,
	img runtime.Image, ws *project.Workspace, layerCount int) error {

	con := console.Current()
	defer con.Reset()

	// build the container and provide progress output
	progress := make(chan []runtime.ProgressStatus)
	var wg sync.WaitGroup
	defer wg.Wait()
	go func() {
		defer wg.Done()
		wg.Add(1)
		showProgress(progress)
	}()

	rb := NewRingBuffer(outputLineCount, outputLineLength)
	stream := rb.StreamWriter()

	err := container.Build(ctx, run, ctr, img, ws, layerCount, &user, &params, progress, stream)
	if err != nil && errors.Is(err, errdefs.ErrCommandFailed) {
		line := make([]byte, 100)
		fmt.Printf("Output:\n")
		for _, err := rb.Read(line); err != io.EOF; _, err = rb.Read(line) {
			fmt.Printf(" > %v\n", string(line))
		}
		return err
	}
	return err
}

// buildContainer builds the full container for the provided workspace and
// commits it.
func buildContainer(ctx context.Context, run runtime.Runtime, ws *project.Workspace,
	layerCount int) (runtime.Container, error) {

	params.Upgrade = buildWorkspaceUpgrade
	ctr, img, err := getContainer(ctx, run, ws)
	if err != nil {
		return nil, err
	}

	err = buildLayers(ctx, run, ctr, img, ws, layerCount)
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
	Use:     "workspace [name]",
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
	ctr, err := container.GetContainer(ctx, run, ws)
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
