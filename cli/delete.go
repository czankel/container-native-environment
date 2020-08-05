package cli

import (
	"github.com/spf13/cobra"

	"github.com/czankel/cne/config"
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
	conf := config.Load()

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

	conf := config.Load()

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
	_, err = prj.Workspace(name)
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

	conf := config.Load()

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

	err = ws.DeleteLayer(args[0])
	if err != nil {
		return err
	}

	return prj.Write()
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(deleteImageCmd)
	deleteCmd.AddCommand(deleteWorkspaceCmd)
	deleteCmd.AddCommand(deleteLayerCmd)
}
