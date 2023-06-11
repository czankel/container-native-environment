package cli

import (
	"github.com/spf13/cobra"

	"github.com/czankel/cne/errdefs"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update an existing resource",
	Aliases: []string{"u"},
	Args:    cobra.MinimumNArgs(1),
}

var updateWorkspaceCmd = &cobra.Command{
	Use:     "workspace [name]",
	Short:   "Update a workspace resources",
	Aliases: []string{"ws"},
	Args:    cobra.ExactArgs(1),
	RunE:    updateWorkspaceRunE,
}

var updateWorkspaceName string
var updateWorkspaceImage string

func updateWorkspaceRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}

	wsName := args[0]

	if updateWorkspaceName != "" {
		for _, ws := range prj.Workspaces {
			if ws.Name == updateWorkspaceName {
				return errdefs.AlreadyExists("workspace", updateWorkspaceName)
			} else if ws.Name == wsName {
				ws.Name = wsName
			}
		}
	}

	err = prj.Write()
	return err
}

var updateConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Update the environment configuration",
	Long: `Update the user or system configuration for the environment.
By default, the configuration is written to the user configuration file.
The system option modifies the system-wide configuration file stored in
/etc, and requires system permissions.`,
	RunE: updateConfigRunE,
	Args: cobra.ExactArgs(2),
}

func updateConfigRunE(cmd *cobra.Command, args []string) error {

	var err error

	conf, err = loadConfig()
	if err != nil {
		return err
	}

	name := args[0]
	newVal := args[1]
	oldVal, path, err := conf.SetByName(name, newVal)
	if err != nil {
		return err
	}

	err = writeConfig()
	if err != nil {
		return err
	}

	printList([]struct {
		Configuration string
		Value         string
		Old           string
	}{{path, newVal, oldVal}}, false)
	return nil
}

var updateProjectCmd = &cobra.Command{
	Use:     "project",
	Short:   "Update the project",
	Aliases: []string{"prj"},
	RunE:    updateProjectRunE,
	Args:    cobra.NoArgs,
}

var updateProjectWorkspace string

func updateProjectRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}

	err = prj.SetCurrentWorkspace(updateProjectWorkspace)
	if err != nil {
		return err
	}

	return prj.Write()
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(updateWorkspaceCmd)
	updateWorkspaceCmd.Flags().StringVarP(
		&updateWorkspaceName, "name", "", "", "Rename the workspace")
	updateCmd.AddCommand(updateConfigCmd)
	updateConfigCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "Update system configuration")
	updateConfigCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Update project configuration")
	updateProjectCmd.Flags().StringVar(
		&updateProjectWorkspace, "workspace", "", "Update current workspace")
	updateProjectCmd.Flags().StringVar(
		&updateProjectWorkspace, "ws", "", "Update current workspace")
	updateCmd.AddCommand(updateProjectCmd)
}
