package cli

import (
	"strconv"

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

	var wsName string
	if len(args) > 0 {
		wsName = args[0]
	} else {
		wsName = "main"
		idx := 0
		for i := 0; i < len(prj.Workspaces); i++ {
			if wsName == prj.Workspaces[idx].Name {
				wsName = "ws-" + strconv.Itoa(idx)
				idx++
				i = 0
			}
		}
	}

	ws := prj.NewWorkspace(wsName)
	err = prj.InsertWorkspace(ws, createWorkspaceInsert)
	if err != nil {
		return err
	}

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	imgName := conf.FullImageName(createWorkspaceFrom)
	_, err = run.PullImage(imgName)
	if err != nil {
		return err
	}
	ws.Origin = imgName

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
