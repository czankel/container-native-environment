package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Show or get the current context",
	Long: `The context allows users to swich quickly between different
configurations, mainly the runtime but also other settings.`,
	RunE: contextRunE,
	Args: cobra.RangeArgs(0, 1),
}

func contextRunE(cmd *cobra.Command, args []string) error {

	if len(args) == 0 {
		fmt.Printf("Context: %s\n", conf.Settings.Context)
	} else {
		// FIXME: check if context exists or error; set context an save config

	}
	return nil
}

func init() {
	rootCmd.AddCommand(contextCmd)
}
