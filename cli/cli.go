// The package cli implements the command line interface.
package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/config"
)

var conf *config.Config

var rootCmd = &cobra.Command{
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Container based environment and deployment tool.",
	Long: `
Container Native Environment (CNE) is a tool for building and managing
virtual environment based on containers to provide a reliable and
reproducible environment for development and other use cases, such as
machine learning or analytics.
`,
}

func init() {
	rootCmd.Use = os.Args[0]
	cobra.OnInitialize(initConfig)
}

// Execute is the main entry point to the CLI. It executes the commands and arguments provided
// in os.Args[1:]
func Execute() error {
	return rootCmd.Execute()
}

func initConfig() {

	conf = config.Load()
}
