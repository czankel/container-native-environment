package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
	"github.com/czankel/cne/support"
)

var showCmd = &cobra.Command{
	Use:     "show",
	Short:   "Show resources",
	Aliases: []string{"g"},
	Args:    cobra.MinimumNArgs(1),
}

var showContextCmd = &cobra.Command{
	Use:   "context [name]",
	Short: "Show context configurartion",
	RunE:  showContextRunE,
	Args:  cobra.RangeArgs(0, 1),
}

func showContextRunE(cmd *cobra.Command, args []string) error {

	fmt.Printf("conf %v\n", conf.Settings)
	entry := conf.Settings.Context
	if len(args) > 0 {
		entry = args[0]
	} else if entry == "" {
		return errdefs.NotFound("context", "")
	}
	_, val, err := conf.GetAllByName("context/" + entry)
	if err == nil {
		printValue("Configuration", "Value", "", val)
	}
	return nil
}

var showImageCmd = &cobra.Command{
	Use:   "image [name]",
	Short: "Show image details",
	RunE:  showImageRunE,
	Args:  cobra.RangeArgs(0, 1),
}

type OS struct {
	Name    string
	Version string
	ID      string
	ID_LIKE string
}

// TODO hide some details and expose with option
func showImageRunE(cmd *cobra.Command, args []string) error {

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

	var imgName string
	if len(args) > 0 {
		imgName = args[0]
	} else {
		prj, err := loadProject()
		if err != nil {
			return err
		}
		var ws *project.Workspace
		ws, err = prj.CurrentWorkspace()
		if err != nil {
			return err
		}
		imgName = ws.Environment.Origin
	}

	imgName, err = conf.FullImageName(imgName)
	if err != nil {
		return err
	}

	img, err := run.GetImage(ctx, imgName)
	if err != nil && errors.Is(err, errdefs.ErrNotFound) {
		img, err = pullImage(ctx, run, imgName)
	}
	if err != nil {
		return err
	}

	fullName := "<unknown>"
	info, err := support.GetImageInfo(ctx, img)
	if info != nil {
		fullName = info.FullName
	}

	rootfs := []string{}
	imgRootFS, err := img.RootFS(ctx)
	if err != nil {
		return err
	}

	for _, r := range imgRootFS {
		rootfs = append(rootfs, r.String())
	}

	image := struct {
		Name   string
		Size   int64
		OS     string
		RootFS []string
	}{
		Name:   img.Name(),
		Size:   img.Size(),
		OS:     fullName,
		RootFS: rootfs,
	}

	printValue("Field", "Value", "", image)
	return nil
}

var showProjectCmd = &cobra.Command{
	Use:     "project",
	Short:   "Show the project configuration",
	Aliases: []string{"prj"},
	RunE:    showProjectRunE,
	Args:    cobra.NoArgs,
}

func showProjectRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}
	printValue("Field", "Value", "", prj)
	return nil
}

var showRegistryCmd = &cobra.Command{
	Use:   "registry [name]",
	Short: "Show registry configurartion",
	RunE:  showRegistryRunE,
	Args:  cobra.RangeArgs(0, 1),
}

func showRegistryRunE(cmd *cobra.Command, args []string) error {

	var entry string
	if len(args) > 0 {
		entry = args[0]
	} else {
		cfgCtx, _, err := conf.GetContext()
		if err != nil {
			return err
		}
		entry = cfgCtx.Registry
	}
	_, val, err := conf.GetAllByName("registry/" + entry)
	if err == nil {
		printValue("Configuration", "Value", "", val)
	}
	return nil
}

var showRuntimeCmd = &cobra.Command{
	Use:   "runtime [name]",
	Short: "Show runtime configurartion",
	RunE:  showRuntimeRunE,
	Args:  cobra.RangeArgs(0, 1),
}

var showSystemConfig bool
var showUserConfig bool
var showProjectConfig bool

func showRuntimeRunE(cmd *cobra.Command, args []string) error {

	var entry string
	if len(args) > 0 {
		entry = args[0]
	} else {
		cfgCtx, _, err := conf.GetContext()
		if err != nil {
			return err
		}
		entry = cfgCtx.Runtime
	}
	_, val, err := conf.GetAllByName("runtime/" + entry)
	if err == nil {
		printValue("Configuration", "Value", "", val)
	}
	return nil
}

var showWorkspaceCmd = &cobra.Command{
	Use:   "workspace [name]",
	Short: "Show workspace details",
	RunE:  showWorkspaceRunE,
	Args:  cobra.RangeArgs(0, 1),
}

func showWorkspaceRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}

	var ws *project.Workspace

	if len(args) > 0 {
		ws, err = prj.Workspace(args[0])
	} else {
		ws, err = prj.CurrentWorkspace()
	}
	if err != nil {
		return err
	}

	printValue("Field", "Value", "", ws)

	return nil
}

func init() {
	rootCmd.AddCommand(showCmd)

	showCmd.AddCommand(showContextCmd)
	showContextCmd.Flags().BoolVarP(
		&showSystemConfig, "system", "", false, "Show only system configurations")
	showContextCmd.Flags().BoolVarP(
		&showProjectConfig, "project", "", false, "Show only project configurations")
	showContextCmd.Flags().BoolVarP(
		&showUserConfig, "user", "", false, "Show only user configurations")

	showCmd.AddCommand(showImageCmd)
	showCmd.AddCommand(showProjectCmd)

	showCmd.AddCommand(showRegistryCmd)

	showCmd.AddCommand(showRuntimeCmd)

	showCmd.AddCommand(showWorkspaceCmd)
}
