package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/runtime"
	"github.com/czankel/cne/service"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run cne as a server",
	Long: `
CNE runs in server mode accepting a connection to execute
from a remote CNE instance.`,
	Args: cobra.NoArgs,
	RunE: serverRunE,
}

// FIXME: should remote target be prohibited? does 'chaining' make any sense? Otherwise, it will call itself... or check that somehow?
func serverRunE(cmd *cobra.Command, args []string) error {

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

	return service.Listen(run)
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
