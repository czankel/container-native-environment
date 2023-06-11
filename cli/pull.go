package cli

import (
	"context"
	"sync"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/runtime"
)

func pullImage(ctx context.Context, run runtime.Runtime,
	imageName string) (runtime.Image, error) {

	progress := make(chan []runtime.ProgressStatus)
	var wg sync.WaitGroup
	defer wg.Wait()
	go func() {
		defer wg.Done()
		wg.Add(1)
		showProgress(progress)
	}()

	return run.PullImage(ctx, imageName, progress)
}

var pullCmd = &cobra.Command{
	Use:   "pull [registry]package[:tag|@digest]",
	Short: "Pull an image from a registry",
	Long: `
Pull an image from a registry to the local system.
REGISTRY can be one of the configured registries or directly
specify the domain and repository. If omitted, the default
registry is used.`,
	Args: cobra.ExactArgs(1),
	RunE: pullImageRunE,
}

func pullImageRunE(cmd *cobra.Command, args []string) error {

	imgName, err := conf.FullImageName(args[0])
	if err != nil {
		return err
	}

	runCfg, err := conf.GetRuntime()
	if err != nil {
		return err
	}

	ctx := context.Background()
	run, err := runtime.Open(ctx, runCfg)
	if err != nil {
		return err
	}
	defer run.Close()
	ctx = run.WithNamespace(ctx, runCfg.Namespace)

	_, err = pullImage(ctx, run, imgName)

	return err
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
