package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/runtime"
)

var fileserverCmd = &cobra.Command{
	Use:  "fileserver",
	Args: cobra.NoArgs,
	RunE: fileserverRunE,
}

func fileserverRunE(cmd *cobra.Command, args []string) error {

	if cneVersion {
		cliVersionRun(cmd, args)
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

	return errdefs.NotImplemented()
}

func init() {
	rootCmd.AddCommand(fileserverCmd)
}
