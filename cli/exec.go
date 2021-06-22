package cli

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/containerd/console"

	"github.com/czankel/cne/container"
	"github.com/czankel/cne/errdefs"
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
var execLayerName string
var execTestOnly bool

// execCommandsInShell executes the provided commands in a shell.
func execCommandsInShell(wsName, layerName string, args []string) (int, error) {
	args = append([]string{"/bin/sh", "-c"}, args...)
	return execCommands(wsName, layerName, args)
}

// execCommands executes the provided commands in the current or provided workspace.
// It returns a code != 0 if the executed command failed. The returned 'code' value
// is the value returned by the command. The caller should call exit(code) to have a
// similar return value as if the command was executed directly.
func execCommands(wsName, layerName string, args []string) (int, error) {

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return 0, err
	}
	defer run.Close()

	prj, err := project.Load()
	if err != nil {
		return 0, err
	}

	var ws *project.Workspace
	if wsName == "" {
		ws, err = prj.CurrentWorkspace()
		if err != nil {
			return 0, err
		}
	}

	stream := runtime.Stream{
		Stdin:    os.Stdin,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Terminal: true,
	}

	con := console.Current()
	defer con.Reset()

	// TODO: check return errors?
	con.SetRaw()
	winSz, _ := con.Size()
	con.Resize(winSz)

	if execLayerName == "" {

		ctr, err := container.Get(run, ws)
		if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
			return 0, err
		}
		if ctr == nil {
			ctr, err = buildContainer(run, ws)
			if err != nil {
				return 0, err
			}
			prj.Write()
		}

		code, err := ctr.Exec(&user, stream, args)
		if err != nil && errors.Is(err, errdefs.ErrNotFound) {
			return 0, errors.New(args[0] + ": no such command")
		}
		if err != nil {
			return int(code), nil
		}

	} else {

		layerIdx, layer := ws.FindLayer(execLayerName)
		if layer == nil {
			return 0, errdefs.InvalidArgument("No such layer: %s", execLayerName)
		}

		ctr, err := createContainer(run, ws)
		if err != nil {
			return 0, err
		}

		// build all layers including the destinationlayer (i.e. + 1)
		err = buildLayers(run, ctr, ws, layerIdx+1)
		if err != nil {
			return 0, err
		}
		code, err := ctr.BuildExec(&user, stream, args, []string{})
		if err != nil {
			return 0, err
		}
		if code != 0 {
			return int(code), nil
		}

		if !execTestOnly {

			layer.Commands = append(layer.Commands,
				project.Command{"", []string{}, args})

			err = ctr.Amend(ws, layerIdx)
			if err != nil && !errors.Is(err, errdefs.ErrAlreadyExists) {
				ctr.Delete() // delete the container and active snapshot
				return 0, err
			}

			err = prj.Write()
			if err != nil {
				ctr.Delete() // delete the container and active snapshot
				return 0, err
			}
		}
	}

	return 0, nil
}

func execRunE(cmd *cobra.Command, args []string) error {

	var code int
	var err error

	if execShell {
		code, err = execCommandsInShell("", "", args)
	} else {
		code, err = execCommands("", "", args)
	}
	if code != 0 {
		os.Exit(code)
	}
	return err
}

func init() {
	execCmd.Flags().BoolVarP(&execShell, "shell", "s", false,
		"Start a shell for the provided commands")
	execCmd.Flags().StringVarP(&execLayerName, "layer", "l", "",
		"Execute a command in this layer to rebuild the layer and amend the project")
	execCmd.Flags().BoolVar(&execTestOnly, "test-only", false,
		"Don't amend the layer")
	rootCmd.AddCommand(execCmd)
}
