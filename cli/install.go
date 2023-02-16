package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/containerd/console"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
	"github.com/czankel/cne/support"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install software 'persistently'",
	Args:  cobra.MaximumNArgs(1),
}

var installAptUpdate bool

var installAptCmd = &cobra.Command{
	Use:   "apt",
	Short: "Install an APT package (for Ubuntu images)",
	Long: `
`,
	Args: cobra.MaximumNArgs(1),
	RunE: installAptRunE,
}

func installAptRunE(cmd *cobra.Command, args []string) error {

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

	aptLayerIdx, aptLayer := ws.FindLayer(project.LayerTypeApt)
	if aptLayer == nil {
		return errdefs.InvalidArgument("Workspace has no apt layer")
	}

	ctr, err := buildContainer(run, ws, aptLayerIdx+1)

	con := console.Current()
	defer con.Reset()

	con.SetRaw()
	winSz, _ := con.Size()
	con.Resize(winSz)

	stream := runtime.Stream{
		Stdin:    os.Stdin,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Terminal: true,
	}

	code, err := support.AptInstall(ws, aptLayerIdx, user, ctr, stream, installAptUpdate, args)
	if err != nil {
		ctr.Delete() // delete the container and active snapshot
		return err
	}
	if code != 0 {
		ctr.Delete()
		con.Reset()
		run.Close()
		os.Exit(code)
	}

	snap, err := ctr.Amend()
	if err != nil {
		ctr.Delete() // delete the container and active snapshot
		return err
	}
	layer := &ws.Environment.Layers[aptLayerIdx]
	layer.Digest = snap.Name()

	err = prj.Write()
	if err != nil {
		ctr.Delete() // delete the container and active snapshot
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.AddCommand(installAptCmd)
	installAptCmd.Flags().BoolVar(
		&installAptUpdate, "update", false, "Update the APT database")
}
