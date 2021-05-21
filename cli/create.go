package cli

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/czankel/cne/container"
	"github.com/czankel/cne/errdefs"
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

	prj.CurrentWorkspaceName = wsName

	return prj.Write()
}

var createLayerCmd = &cobra.Command{
	Use:     "layer NAME [CMDLINE]",
	Short:   "Create a new layer",
	Aliases: []string{"l"},
	Args:    cobra.MinimumNArgs(1),
	RunE:    createLayerRunE,
}

func createLayerRunE(cmd *cobra.Command, args []string) error {

	prj, err := project.Load()
	if err != nil {
		return err
	}

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	ws, err := prj.CurrentWorkspace()
	if err != nil {
		return err
	}

	oldCtr, err := container.Get(run, ws)
	if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
		return err
	}

	isTerminal := term.IsTerminal(int(os.Stdin.Fd()))
	if (len(args) > 1) != isTerminal {
		return errdefs.InvalidArgument("too many arguments")
	}

	var cmdLines []string
	if len(args) > 1 {
		cmdLines = scanLine(args[1])
	} else if !isTerminal {

		cmdLines, err = readCommands(os.Stdin)
		if err != nil {
			return err
		}
	}

	atIndex := -1
	if createWorkspaceInsert != "" {
		for i, l := range ws.Environment.Layers {
			if l.Name == createWorkspaceInsert {
				atIndex = i
				break
			}
		}
		if atIndex == -1 {
			return errdefs.InvalidArgument("invalid index")
		}
	}

	layer, err := ws.CreateLayer(args[0], atIndex)
	if err != nil {
		return err
	}
	layer.Commands = cmdLines

	if len(cmdLines) > 0 {
		_, err := buildContainer(run, prj, ws)
		if err != nil {
			return err
		}
	}

	err = prj.Write()
	if err != nil {
		return err
	}
	if oldCtr != nil {
		// Ignore any errors, TOOD: add warning
		oldCtr.Delete()
	}

	return nil
}

func init() {

	rootCmd.AddCommand(createCmd)

	createCmd.AddCommand(createWorkspaceCmd)
	createWorkspaceCmd.Flags().StringVar(
		&createWorkspaceFrom, "from", "", "Base image for the workspace")
	createWorkspaceCmd.Flags().StringVar(
		&createWorkspaceInsert, "insert", "", "Insert before this workspace")

	createCmd.AddCommand(createLayerCmd)
}
