package cli

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/czankel/cne/container"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
	"github.com/czankel/cne/support"
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

var createWorkspaceImage string
var createWorkspaceInsert string

func createWorkspaceRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}

	wsName := ""
	if len(args) != 0 {
		wsName = args[0]
	}

	return initWorkspace(prj, wsName, createWorkspaceImage, createWorkspaceInsert)
}

func initWorkspace(prj *project.Project, wsName, insert, imgName string) error {

	imgName, err := conf.FullImageName(imgName)
	if err != nil {
		return err
	}

	ws, err := prj.CreateWorkspace(wsName, imgName, insert)
	if err != nil {
		return err
	}

	if imgName != "" {
		runCfg, err := conf.GetRuntime()
		if err != nil {
			return err
		}

		ctx := context.Background()
		run, err := runtime.Open(ctx, runCfg)
		if err != nil {
			return err
		}
		defer run.Close()
		ctx = run.WithNamespace(ctx, runCfg.Namespace)

		img, err := pullImage(ctx, run, imgName)
		if err != nil {
			return err
		}

		err = support.SetupWorkspace(ctx, ws, img)
		if err != nil {
			return err
		}
	}

	prj.CurrentWorkspaceName = wsName

	return prj.Write()
}

var createLayerSystem bool
var createLayerInsert string

var createLayerCmd = &cobra.Command{
	Use:     "layer [FLAGS] NAME [CMDLINE]",
	Short:   "Create a new layer",
	Aliases: []string{"l"},
	Args:    cobra.MinimumNArgs(1),
	RunE:    createLayerRunE,
}

func createLayerRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}

	runCfg, err := conf.GetRuntime()
	if err != nil {
		return err
	}

	ctx := context.Background()

	run, err := runtime.Open(ctx, runCfg)
	if err != nil {
		return err
	}
	defer run.Close()
	ctx = run.WithNamespace(ctx, runCfg.Namespace)

	ws, err := prj.CurrentWorkspace()
	if err != nil {
		return err
	}

	oldCtr, err := container.Get(ctx, run, ws)
	if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
		return err
	}

	isTerminal := term.IsTerminal(int(os.Stdin.Fd()))
	if len(args) > 1 && !isTerminal {
		return errdefs.InvalidArgument("too many arguments")
	}

	var commands []project.Command
	if len(args) > 1 {
		commands = scanLine(args[1])
	} else if !isTerminal {

		commands, err = readCommands(os.Stdin)
		if err != nil {
			return err
		}
	}

	atIndex := -1
	if createLayerInsert != "" {
		for i, l := range ws.Environment.Layers {
			if l.Name == createLayerInsert {
				atIndex = i
				break
			}
		}
		if atIndex == -1 {
			return errdefs.InvalidArgument("invalid index")
		}
	}

	rebuildContainer := createLayerSystem
	if createLayerSystem {
		err = support.CreateSystemLayer(ws, args[0], atIndex)
		if err != nil {
			return err
		}
	} else {
		layerName := args[0]
		for _, n := range project.SystemLayerTypes {
			if layerName == n {
				return errdefs.InvalidArgument("%s is a reserved layer name, use --system",
					layerName)
			}
		}

		layer, err := ws.CreateLayer(createLayerSystem, layerName, atIndex)
		layer.Commands = commands
		if err != nil {
			return err
		}
		rebuildContainer = len(commands) > 0
	}

	if rebuildContainer {
		_, err := buildContainer(ctx, run, ws, -1)
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
		oldCtr.Delete(ctx)
	}

	return nil
}

func init() {

	rootCmd.AddCommand(createCmd)

	createCmd.AddCommand(createWorkspaceCmd)
	createWorkspaceCmd.Flags().StringVar(
		&createWorkspaceImage, "image", "", "Base image for the workspace")
	createWorkspaceCmd.Flags().StringVar(
		&createWorkspaceInsert, "insert", "", "Insert before this workspace")

	createCmd.AddCommand(createLayerCmd)
	createLayerCmd.Flags().StringVar(
		&createLayerInsert, "insert", "", "Insert before this layer")
	createLayerCmd.Flags().BoolVarP(
		&createLayerSystem, "system", "s", false,
		"User the system handler of the same name")
}
