package cli

import (
	"errors"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/container"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete resources",
	Aliases: []string{"d"},
	Args:    cobra.MinimumNArgs(1),
}

var deleteImageCmd = &cobra.Command{
	Use:     "image NAME",
	Aliases: []string{"image", "i"},
	Short:   "delete image",
	Args:    cobra.ExactArgs(1),
	RunE:    deleteImageRunE,
}

func deleteImageRunE(cmd *cobra.Command, args []string) error {

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	return run.DeleteImage(conf.FullImageName(args[0]))
}

var deleteWorkspaceCmd = &cobra.Command{
	Use:     "workspace NAME",
	Aliases: []string{"workspace", "ws"},
	Short:   "delete workspace",
	Args:    cobra.ExactArgs(1),
	RunE:    deleteWorkspaceRunE,
}

func deleteWorkspaceRunE(cmd *cobra.Command, args []string) error {

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	prj, err := loadProject()
	if err != nil {
		return err
	}

	name := args[0]
	ws, err := prj.Workspace(name)
	if err != nil {
		return err
	}

	// ignore error TODO: print warning for error other than not-found
	ctr, err := container.Get(run, ws)
	if err == nil {
		ctr.Purge()
	}

	err = prj.DeleteWorkspace(name)
	if err != nil {
		return err
	}

	return prj.Write()
}

var deleteLayerCmd = &cobra.Command{
	Use:     "layer NAME",
	Aliases: []string{"layer", "l"},
	Short:   "delete layer",
	Args:    cobra.ExactArgs(1),
	RunE:    deleteLayerRunE,
}

func deleteLayerRunE(cmd *cobra.Command, args []string) error {

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	prj, err := loadProject()
	if err != nil {
		return err
	}

	ws, err := prj.CurrentWorkspace()
	if err != nil {
		return err
	}

	oldCtr, err := container.Get(run, ws)
	if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
		return err
	}

	err = ws.DeleteLayer(args[0])
	if err != nil {
		return err
	}

	ctr, err := buildContainer(run, ws)
	if err != nil {
		return err
	}

	err = prj.Write()
	if err != nil {
		ctr.Delete()
		return err
	}

	if oldCtr != nil {
		oldCtr.Delete()
	}

	return nil
}

var deleteContainerCmd = &cobra.Command{
	Use:   "container NAME",
	Short: "delete container",
	Args:  cobra.ExactArgs(1),
	RunE:  deleteContainerRunE,
}

func deleteContainerRunE(cmd *cobra.Command, args []string) error {

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	prj, err := loadProject()
	if err != nil {
		return err
	}

	// delete all containers that match the domain+id
	ctrs, err := container.Containers(run, prj, &user)
	if err != nil {
		return err
	}
	for _, c := range ctrs {
		if c.Name() == args[0] {
			c.Purge()
			break
		}
	}

	return nil
}

var deleteCommandCmd = &cobra.Command{
	Use:     "command INDEX|NAME",
	Short:   "delete commands",
	Aliases: []string{"cmd"},
	Args:    cobra.ExactArgs(1),
	RunE:    deleteCommandRunE,
}

var deleteCommandWorkspace string
var deleteCommandLayer string

func deleteCommandRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}

	var ws *project.Workspace
	if deleteCommandWorkspace != "" {
		ws, err = prj.Workspace(deleteCommandWorkspace)
	} else {
		ws, err = prj.CurrentWorkspace()
	}
	if err != nil {
		return err
	}

	if len(ws.Environment.Layers) == 0 {
		return errdefs.InvalidArgument("No layers in workspace")
	}

	layer := &ws.Environment.Layers[len(ws.Environment.Layers)-1]
	if deleteCommandLayer != "" {
		_, layer = ws.FindLayer(deleteCommandLayer)
		if layer == nil {
			return errdefs.InvalidArgument("No such layer: %s", deleteCommandLayer)
		}
	}

	index := -1
	if index, err = strconv.Atoi(args[0]); err != nil {
		for i, c := range layer.Commands {
			if args[0] == c.Name {
				index = i
				break
			}
		}
		if index == -1 {
			return errdefs.InvalidArgument("No commands for name: %s", args[0])
		}
	}
	if index >= len(layer.Commands) {
		return errdefs.InvalidArgument("Index out of range: %d", index)
	}

	layer.Commands = append(layer.Commands[:index], layer.Commands[index+1:]...)

	ws.UpdateLayer(layer)
	return prj.Write()
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(deleteImageCmd)
	deleteCmd.AddCommand(deleteWorkspaceCmd)
	deleteCmd.AddCommand(deleteLayerCmd)
	deleteCmd.AddCommand(deleteContainerCmd)
	deleteCmd.AddCommand(deleteCommandCmd)
	deleteCommandCmd.Flags().StringVarP(
		&deleteCommandWorkspace, "workspace", "w", "", "Name of the workspace")
	deleteCommandCmd.Flags().StringVarP(
		&deleteCommandLayer, "layer", "l", "", "Name or index of the layer")
}
