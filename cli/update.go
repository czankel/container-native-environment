package cli

import (
	"github.com/spf13/cobra"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update an existing resource",
	Aliases: []string{"u"},
	Args:    cobra.MinimumNArgs(1),
}

var updateWorkspaceCmd = &cobra.Command{
	Use:     "workspace [NAME]",
	Short:   "Update a workspace resources",
	Aliases: []string{"ws"},
	Args:    cobra.ExactArgs(1),
	RunE:    updateWorkspaceRunE,
}

var updateWorkspaceName string
var updateWorkspaceImage string

func updateWorkspaceRunE(cmd *cobra.Command, args []string) error {

	prj, err := project.Load()
	if err != nil {
		return err
	}

	wsName := args[0]

	if updateWorkspaceName != "" {
		for _, ws := range prj.Workspaces {
			if ws.Name == updateWorkspaceName {
				return errdefs.ErrResourceExists
			} else if ws.Name == wsName {
				ws.Name = wsName
			}
		}
	}

	err = prj.Write()
	return err
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(updateWorkspaceCmd)
	updateWorkspaceCmd.Flags().StringVarP(
		&updateWorkspaceName, "name", "", "", "Rename the workspace")
}
