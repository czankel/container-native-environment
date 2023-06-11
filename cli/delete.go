package cli

import (
	"context"
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
	Use:     "image name",
	Aliases: []string{"image", "i"},
	Short:   "delete image",
	Args:    cobra.ExactArgs(1),
	RunE:    deleteImageRunE,
}

func deleteImageRunE(cmd *cobra.Command, args []string) error {

	imgName, err := conf.FullImageName(args[0])
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

	return run.DeleteImage(ctx, imgName)
}

var deleteWorkspaceCmd = &cobra.Command{
	Use:     "workspace name",
	Aliases: []string{"workspace", "ws"},
	Short:   "delete workspace",
	Args:    cobra.ExactArgs(1),
	RunE:    deleteWorkspaceRunE,
}

func deleteWorkspaceRunE(cmd *cobra.Command, args []string) error {

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
	ctr, err := container.GetContainer(ctx, run, ws)
	if err == nil {
		ctr.Purge(ctx)
	}

	err = prj.DeleteWorkspace(name)
	if err != nil {
		return err
	}

	return prj.Write()
}

var deleteLayerCmd = &cobra.Command{
	Use:     "layer name",
	Aliases: []string{"layer", "l"},
	Short:   "delete layer",
	Args:    cobra.ExactArgs(1),
	RunE:    deleteLayerRunE,
}

func deleteLayerRunE(cmd *cobra.Command, args []string) error {

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

	prj, err := loadProject()
	if err != nil {
		return err
	}

	ws, err := prj.CurrentWorkspace()
	if err != nil {
		return err
	}

	oldCtr, err := container.GetContainer(ctx, run, ws)
	if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
		return err
	}

	err = ws.DeleteLayer(args[0])
	if err != nil {
		return err
	}

	ctr, err := buildContainer(ctx, run, ws, -1)
	if err != nil {
		return err
	}

	err = prj.Write()
	if err != nil {
		ctr.Delete(ctx)
		return err
	}

	if oldCtr != nil {
		oldCtr.Delete(ctx)
	}

	return nil
}

var deleteContainerCmd = &cobra.Command{
	Use:   "container name",
	Short: "delete container",
	Args:  cobra.ExactArgs(1),
	RunE:  deleteContainerRunE,
}

func deleteContainerRunE(cmd *cobra.Command, args []string) error {

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

	prj, err := loadProject()
	if err != nil {
		return err
	}

	// delete all containers that match the domain+id
	ctrs, err := container.Containers(ctx, run, prj, &user)
	if err != nil {
		return err
	}
	for _, c := range ctrs {
		if c.Name() == args[0] {
			c.Purge(ctx)
			break
		}
	}

	return nil
}

var deleteCommandCmd = &cobra.Command{
	Use:     "command index|name",
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

var deleteConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Remove a configuraion entry",
	Long: `Remove a context, registry, or runtime. Use delete with the --system or
--project option to delete a system or project configuration.`,
	Args: cobra.NoArgs,
}

var deleteConfigContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Remove a configuration context entry",
	Args:  cobra.ExactArgs(1),
	RunE:  deleteConfigContextRunE,
}

func deleteConfigContextRunE(cmd *cobra.Command, args []string) error {

	tempConf, err := loadConfig()
	if err != nil {
		return err
	}

	err = tempConf.RemoveContext(args[0])
	if err != nil {
		return err
	}

	return writeConfig(tempConf)
}

var deleteConfigRegistryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Remove a configuration registry entry",
	Args:  cobra.ExactArgs(1),
	RunE:  deleteConfigRegistryRunE,
}

func deleteConfigRegistryRunE(cmd *cobra.Command, args []string) error {

	tempConf, err := loadConfig()
	if err != nil {
		return err
	}

	err = tempConf.RemoveRegistry(args[0])
	if err != nil {
		return err
	}

	return writeConfig(tempConf)
}

var deleteConfigRuntimeCmd = &cobra.Command{
	Use:   "runtime runtime",
	Short: "Remove a configuration runtime entry",
	Args:  cobra.ExactArgs(1),
	RunE:  deleteConfigRuntimeRunE,
}

func deleteConfigRuntimeRunE(cmd *cobra.Command, args []string) error {

	tempConf, err := loadConfig()
	if err != nil {
		return err
	}

	err = tempConf.RemoveRuntime(args[0])
	if err != nil {
		return err
	}
	return writeConfig(tempConf)
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(deleteCommandCmd)

	deleteCmd.AddCommand(deleteConfigCmd)
	deleteConfigCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "Update system configuration")
	deleteConfigCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Update project configuration")
	deleteConfigCmd.AddCommand(deleteConfigContextCmd)
	deleteConfigCmd.AddCommand(deleteConfigRegistryCmd)
	deleteConfigCmd.AddCommand(deleteConfigRuntimeCmd)

	deleteCmd.AddCommand(deleteContainerCmd)
	deleteCmd.AddCommand(deleteImageCmd)
	deleteCmd.AddCommand(deleteLayerCmd)

	deleteCmd.AddCommand(deleteWorkspaceCmd)
	deleteCommandCmd.Flags().StringVarP(
		&deleteCommandWorkspace, "workspace", "w", "", "Name of the workspace")
	deleteCommandCmd.Flags().StringVarP(
		&deleteCommandLayer, "layer", "l", "", "Name or index of the layer")
}
