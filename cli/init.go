package cli

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
)

var initCmd = &cobra.Command{
	Use:   "init [NAME]",
	Short: "Create or initialize a project",
	Long: `
The init command creates a new project or initializes an existing project.
The name argument is optional and can only be used for creating a new
project. If omitted the project will be created with the directory name
as the project name.`,
	Args: cobra.MaximumNArgs(1),
	RunE: initProjectRunE,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// initProject initializes and existing project or creates a new project.
// name: optional name for new projects.
func initProjectRunE(cmd *cobra.Command, args []string) error {

	_, err := project.Load()
	if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
		return err
	}

	if errors.Is(err, errdefs.ErrNotFound) {
		var path string
		path, err = os.Getwd()
		if err != nil {
			return errdefs.SystemError(err,
				"failed to setup project in working directory")
		}
		name := filepath.Base(path)
		if len(args) > 0 {
			name = args[0]
		}
		_, err = project.Create(name, path)
		if err != nil {
			return err
		}
	} else if len(args) > 0 {
		return errdefs.AlreadyExists("project", args[0])
	}

	return nil
}
