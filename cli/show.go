package cli

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/config"
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

var showConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show the environment configuration",
	Long: `Show the configuration for the environment in the current directory or globally
for all environments of the current user.
By default, this command returns the configuration derived from all
configuration files. The system option returns only the syste-wide
configuration and the user option the configuration for the user.`,
	RunE: showConfigRunE,
	Args: cobra.NoArgs,
}

var showConfigContextCmd = &cobra.Command{
	Use:   "context [name]",
	Short: "Show context configurartion",
	RunE:  showConfigRunE,
	Args:  cobra.RangeArgs(0, 1),
}

var showConfigRegistryCmd = &cobra.Command{
	Use:   "registry [name]",
	Short: "Show registry configurartion",
	RunE:  showConfigRunE,
	Args:  cobra.RangeArgs(0, 1),
}

var showConfigRuntimeCmd = &cobra.Command{
	Use:   "runtime [name]",
	Short: "Show runtime configurartion",
	RunE:  showConfigRunE,
	Args:  cobra.RangeArgs(0, 1),
}

var showSystemConfig bool
var showUserConfig bool
var showProjectConfig bool

func showConfigRunE(cmd *cobra.Command, args []string) error {

	var err error

	if showSystemConfig {
		conf, err = config.LoadSystemConfig()
	} else if showUserConfig {
		conf, err = config.LoadUserConfig()
	} else if showProjectConfig {
		prj, err := loadProject()
		if err != nil {
			return err
		}
		conf, err = config.LoadProjectConfig(filepath.Dir(prj.Path))
	} else {
		conf, err = config.Load()
		prj, err := loadProject()
		if err == nil {
			err = conf.UpdateProjectConfig(filepath.Dir(prj.Path))
		}
	}

	if err != nil {
		return err
	}

	entry := cmd.CalledAs()
	if entry == "config" {
		printValue("Configuration", "Value", "", conf)
	} else {
		if len(args) > 0 {
			entry = entry + "/" + args[0]
		}
		_, val, err := conf.GetAllByName(entry)
		if err == nil {
			printValue("Configuration2", "Value", "", val)
		}
	}
	return err
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

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.AddCommand(showConfigCmd)
	showConfigCmd.Flags().BoolVarP(
		&showSystemConfig, "system", "", false, "Show only system configurations")
	showConfigCmd.Flags().BoolVarP(
		&showProjectConfig, "project", "", false, "Show only project configurations")
	showConfigCmd.Flags().BoolVarP(
		&showUserConfig, "user", "", false, "Show only user configurations")
	showConfigCmd.AddCommand(showConfigContextCmd)
	showConfigCmd.AddCommand(showConfigRegistryCmd)
	showConfigCmd.AddCommand(showConfigRuntimeCmd)

	showCmd.AddCommand(showProjectCmd)
	showCmd.AddCommand(showWorkspaceCmd)
	showCmd.AddCommand(showImageCmd)
}
