package cli

import (
	"github.com/spf13/cobra"

	"github.com/czankel/cne/image"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add resource",
	Args:  cobra.MaximumNArgs(1),
}

var addPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Add path(s) (file or directory) to the image",
	Args:  cobra.MinimumNArgs(1),
	RunE:  addPathRunE,
}

func addPathRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}

	return image.AddPath(prj, args)
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.AddCommand(addPathCmd)
}
