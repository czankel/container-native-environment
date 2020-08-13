package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/container"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

var execCmd = &cobra.Command{
	Use:   "exec CMD",
	Short: "Execute a command in the container environment",
	Args:  cobra.MinimumNArgs(1),
	RunE:  execRunE,
}

var execShell bool

func execRunE(cmd *cobra.Command, args []string) error {

	conf := config.Load()
	prj, err := project.Load()
	if err != nil {
		return err
	}

	ws, err := prj.CurrentWorkspace()
	if err != nil {
		return err
	}

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	ctr, err := container.Find(run, ws)
	if err != nil {
		return err
	}

	if ctr == nil {
		ctr, err = buildContainer(conf, run, prj, ws)
		if err != nil {
			return err
		}
	}

	if execShell {
		args = append([]string{"/bin/sh", "-c"}, args...)
	}

	stream := runtime.Stream{
		Stdin:    os.Stdin,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Terminal: true,
	}

	code, err := ctr.Exec(stream, args)
	if err != nil {
		return err
	}

	os.Exit(int(code))

	return nil
}
func init() {
	execCmd.Flags().BoolVarP(&execShell, "", "c", false,
		"Start a shell for the provided commands")
	rootCmd.AddCommand(execCmd)
}
