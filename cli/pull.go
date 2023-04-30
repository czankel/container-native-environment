package cli

import (
	"context"
	"sync"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/runtime"
)

func pullImage(ctx context.Context, run runtime.Runtime,
	imageName string) (runtime.Image, error) {

	var wg sync.WaitGroup

	wg.Add(1)

	progress := make(chan []runtime.ProgressStatus)

	go func() {
		defer wg.Done()
		showImageProgress(progress)
	}()

	img, err := run.PullImage(ctx, imageName, progress)
	wg.Wait()

	return img, err
}

var pullCmd = &cobra.Command{
	Use:   "pull [REGISTRY]PACKAGE[:TAG|@DIGEST]",
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

	ctx := context.Background()
	run, err := runtime.Open(ctx, &conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()
	ctx = run.WithNamespace(ctx, conf.Runtime.Name)

	_, err = pullImage(ctx, run, conf.FullImageName(args[0]))

	return err
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
