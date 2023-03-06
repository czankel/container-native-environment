package cli

import (
	"github.com/spf13/cobra"

	"github.com/czankel/cne/runtime"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run cne as a server",
	Long: `
CNE runs in server mode accepting a connection to execute
from a remote CNE instance.`,
	Args: cobra.ExactArgs(1),
	RunE: serverRunE,
}

func serverRunE(cmd *cobra.Command, args []string) error {

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()
	//_, err = (run, conf.FullImageName(args[0]))

	return err
}

func init() {
	rootCmd.AddCommand(serverCmd)
	//	serverCmd.AddCommand(serverHelp)
}
