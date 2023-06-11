package cli

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
)

var initProjectImage string

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Create or initialize a project",
	Long: `
The init command creates a new project in the current directory.
The project name is optional. If omitted, the name of the current directory
is used as the project name.`,
	Args: cobra.MaximumNArgs(1),
	RunE: initProjectRunE,
}

// initProject creates a new project with an optional name for the project.
func initProjectRunE(cmd *cobra.Command, args []string) error {

	path, err := os.Getwd()
	if err != nil {
		return errdefs.SystemError(err, "failed to get current working directory")
	}

	name := filepath.Base(path)
	if len(args) > 0 {
		name = args[0]
	}

	_, err = project.Load(path + "/" + project.ProjectFileName)
	if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
		return err
	}
	if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
		return errdefs.AlreadyExists("project", name)
	}

	prj, err := project.Create(name, path)
	if err != nil {
		return err
	}

	if initProjectImage != "" {
		err = initWorkspace(prj, project.WorkspaceDefaultName,
			"" /* Insert */, initProjectImage)
		if err != nil {
			prj.Delete()
			return err
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(
		&initProjectImage, "image", "", "Base image")
}
