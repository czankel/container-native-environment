package cli

import (
	"github.com/spf13/cobra"

	"github.com/czankel/cne/runtime"
)

var runCmd = &cobra.Command{
	Use:   "run [COMMAND]",
	Short: "Runs the container without the environment",
	Long: `
This command runs the image as the default user without access
to the users home directory.
The image can then be exported or pushed to a registry if the run
was successful.
`,
	Args: cobra.ExactArgs(1),
	RunE: runRunE,
}

func runRunE(cmd *cobra.Command, args []string) error {

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()
	//_, err = (run, conf.FullImageName(args[0]))

	return err
}

func init() {
	rootCmd.AddCommand(runCmd)
}
