package cli

import (
	"context"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"github.com/containerd/console"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
	"github.com/czankel/cne/support"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove software 'peristently'",
	Args:  cobra.MaximumNArgs(1),
}

var removeAptCmd = &cobra.Command{
	Use:   "apt",
	Short: "Remove an APT package (for Ubuntu images)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  removeAptRunE,
}

func removeAptRunE(cmd *cobra.Command, args []string) error {

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

	aptLayerIdx, aptLayer := ws.FindLayer(project.LayerTypeApt)
	if aptLayer == nil {
		return errdefs.InvalidArgument("Workspace has no apt layer")
	}

	progress := make(chan []runtime.ProgressStatus)
	var wg sync.WaitGroup
	defer wg.Wait()
	go func() {
		defer wg.Done()
		wg.Add(1)
		showProgress(progress)
	}()

	ctr, img, err := getContainer(ctx, run, ws, progress)
	if err != nil {
		return err
	}

	err = buildLayers(ctx, run, ctr, img, ws, aptLayerIdx+1)
	if err != nil {
		return err
	}

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

	code, err := support.AptRemove(ctx, ws, aptLayerIdx, user, ctr, stream, args)
	if err != nil {
		ctr.Delete(ctx) // delete the container and active snapshot
		return err
	}
	if code != 0 {
		ctr.Delete(ctx)
		con.Reset()
		run.Close()
		os.Exit(code)
	}

	snap, err := ctr.Amend(ctx)
	if err != nil {
		ctr.Delete(ctx) // delete the container and active snapshot
		return err
	}
	layer := &ws.Environment.Layers[aptLayerIdx]
	layer.Digest = snap.Name()

	err = prj.Write()
	if err != nil {
		ctr.Delete(ctx) // delete the container and active snapshot
		return err
	}

	return nil
}

var removeConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Remove a configuraion entry",
	Long: `Remove a context, registry, or runtime. Use remove with the --system or
--project option to remove a system or project configuration.`,
	Args: cobra.NoArgs,
}

var removeConfigContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Remove a configuration context entry",
	Args:  cobra.ExactArgs(1),
	RunE:  removeConfigContextRunE,
}

func removeConfigContextRunE(cmd *cobra.Command, args []string) error {

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

var removeConfigRegistryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Remove a configuration registry entry",
	Args:  cobra.ExactArgs(1),
	RunE:  removeConfigRegistryRunE,
}

func removeConfigRegistryRunE(cmd *cobra.Command, args []string) error {

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

var removeConfigRuntimeCmd = &cobra.Command{
	Use:   "runtime runtime",
	Short: "Remove a configuration runtime entry",
	Args:  cobra.ExactArgs(1),
	RunE:  removeConfigRuntimeRunE,
}

func removeConfigRuntimeRunE(cmd *cobra.Command, args []string) error {

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
	rootCmd.AddCommand(removeCmd)
	removeCmd.AddCommand(removeAptCmd)

	removeCmd.AddCommand(removeConfigCmd)
	removeConfigCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "Update system configuration")
	removeConfigCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Update project configuration")
	removeConfigCmd.AddCommand(removeConfigContextCmd)
	removeConfigCmd.AddCommand(removeConfigRegistryCmd)
	removeConfigCmd.AddCommand(removeConfigRuntimeCmd)
}
