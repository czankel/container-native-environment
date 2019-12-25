package cli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/runtime"
)

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

	conf := config.Load()
	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return errors.Wrap(err, "Failed to open runtime:")
	}

	_, err = run.PullImage(conf.FullImageName(args[0]))
	return err
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
