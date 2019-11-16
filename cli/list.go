package cli

import (
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "list",
	Args:    cobra.MinimumNArgs(1),
}

var listRuntimeCmd = &cobra.Command{
	Use:     "runtimes",
	Aliases: []string{"runtime", "r"},
	Short:   "list runtimes",
	Args:    cobra.NoArgs,
	RunE:    listRuntimeRunE,
}

func listRuntimeRunE(cmd *cobra.Command, args []string) error {
	return nil
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listRuntimeCmd)
}
