package cli

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/containerd/console"

	"github.com/czankel/cne/container"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

var execCmd = &cobra.Command{
	Use:   "exec cmd",
	Short: "Execute a command in the container environment",
	Args:  cobra.MinimumNArgs(1),
	RunE:  execRunE,
}

var execShell bool
var execLayerName string
var execTestOnly bool

// execCommandsInShell executes the provided commands in a shell.
func execCommandsInShell(wsName, layerName string, args []string) (int, error) {
	args = append([]string{"/bin/sh", "-c", strings.Join(args, " ")})
	return execCommands(wsName, layerName, args)
}

// execCommands executes the provided commands in the current or provided workspace.
// It returns a code != 0 if the executed command failed. The returned 'code' value
// is the value returned by the command. The caller should call exit(code) to have a
// similar return value as if the command was executed directly.
func execCommands(wsName, layerName string, args []string) (int, error) {

	runCfg, err := conf.GetRuntime()
	if err != nil {
		return 0, err
	}

	ctx := context.Background()

	run, err := runtime.Open(ctx, runCfg)
	if err != nil {
		return 0, err
	}
	defer run.Close()
	ctx = run.WithNamespace(ctx, runCfg.Namespace)

	prj, err := loadProject()
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
		ctr, err := container.GetContainer(ctx, run, ws)
		if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
			return 0, err
		}
		if errors.Is(err, errdefs.ErrNotFound) {
			ctr, err = buildContainer(ctx, run, ws, -1)
			if err != nil {
				return 0, err
			}
			prj.Write()
		}

		code, err := container.Exec(ctx, ctr, &user, stream, args)
		if err != nil && errors.Is(err, errdefs.ErrNotFound) && errdefs.Resource(err) == "command" {
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

		ctr, img, err := getContainer(ctx, run, ws)
		if err != nil {
			return 0, err
		}

		// build all layers including the destinationlayer (i.e. + 1)
		err = buildLayers(ctx, run, ctr, img, ws, layerIdx+1)
		if err != nil {
			return 0, err
		}

		code, err := container.BuildExec(ctx, ctr, &user, stream, args, []string{})
		if err != nil {
			return 0, err
		}
		if code != 0 {
			return int(code), nil
		}

		if !execTestOnly {

			layer.Commands = append(layer.Commands,
				project.Command{"", []string{}, args})

			snap, err := ctr.Amend(ctx)
			if err != nil && !errors.Is(err, errdefs.ErrAlreadyExists) {
				ctr.Delete(ctx) // delete the container and active snapshot
				return 0, err
			}
			layer := &ws.Environment.Layers[layerIdx]
			layer.Digest = snap.Name()

			err = prj.Write()
			if err != nil {
				ctr.Delete(ctx) // delete the container and active snapshot
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
