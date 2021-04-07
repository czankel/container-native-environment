package cli

import (
	"github.com/spf13/cobra"

	"github.com/czankel/cne/container"
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
	Aliases: []string{"workspace", "w"},
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

	prj, err := project.Load()
	if err != nil {
		return err
	}

	name := args[0]
	ws, err := prj.Workspace(name)
	if err != nil {
		return err
	}

	err = container.Delete(run, ws)
	if err != nil {
		return err
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

	prj, err := project.Load()
	if err != nil {
		return err
	}

	ws, err := prj.CurrentWorkspace()
	if err != nil {
		return err
	}

	oldCtr, err := container.Find(run, ws)
	if err != nil {
		return err
	}

	err = ws.DeleteLayer(args[0])
	if err != nil {
		return err
	}

	ctr, err := buildContainer(run, prj, ws)
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

	prj, err := project.Load()
	if err != nil {
		return err
	}

	ctrs, err := container.Containers(run, prj)
	if err != nil {
		return err
	}

	for _, c := range ctrs {
		if c.Name == args[0] {
			c.Delete()
			break
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(deleteImageCmd)
	deleteCmd.AddCommand(deleteWorkspaceCmd)
	deleteCmd.AddCommand(deleteLayerCmd)
	deleteCmd.AddCommand(deleteContainerCmd)
}
