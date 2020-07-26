package cli

import (
	"github.com/spf13/cobra"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

var createCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new resource",
	Aliases: []string{"c"},
	Args:    cobra.MinimumNArgs(1),
}

var createWorkspaceCmd = &cobra.Command{
	Use:     "workspace [NAME]",
	Short:   "Create a new workspace",
	Aliases: []string{"ws"},
	Args:    cobra.MaximumNArgs(1),
	RunE:    createWorkspaceRunE,
}

var createWorkspaceFrom string
var createWorkspaceInsert string

func createWorkspaceRunE(cmd *cobra.Command, args []string) error {

	conf := config.Load()

	prj, err := project.Load()
	if err != nil {
		return err
	}

	wsName := ""
	if len(args) != 0 {
		wsName = args[0]
	}

	imgName := ""
	if createWorkspaceFrom != "" {

		run, err := runtime.Open(conf.Runtime)
		if err != nil {
			return err
		}
		defer run.Close()

		imgName = conf.FullImageName(createWorkspaceFrom)
		_, err = pullImage(run, imgName)
		if err != nil {
			return err
		}
	}

	_, err = prj.CreateWorkspace(wsName, imgName, createWorkspaceInsert)
	if err != nil {
		return err
	}

	return prj.Write()
}

func init() {

	rootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createWorkspaceCmd)
	createWorkspaceCmd.Flags().StringVar(
		&createWorkspaceFrom, "from", "", "Base image for the workspace")
	createWorkspaceCmd.MarkFlagRequired("from")
	createWorkspaceCmd.Flags().StringVar(
		&createWorkspaceInsert, "insert", "", "Insert before this workspace")
}
