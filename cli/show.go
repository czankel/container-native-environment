package cli

import (
	"errors"

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
	Args: cobra.RangeArgs(0, 1),
}

var showSystemConfig bool
var showUserConfig bool

func showConfigRunE(cmd *cobra.Command, args []string) error {

	var err error

	if showUserConfig == showSystemConfig {
		conf, err = config.Load()
	} else if showSystemConfig {
		conf, err = config.LoadSystemConfig()
	} else {
		conf, err = config.LoadUserConfig()
	}
	if err != nil {
		return err
	}

	if len(args) == 0 {
		printValue("Configuration", "Value", "", conf)
	} else {
		name := args[0]
		prefix, val, err := conf.GetAllByName(name)
		if err != nil {
			return err
		}
		printValue("Configuration", "Value", prefix, val)
	}

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

	prj, err := project.Load()
	if err != nil {
		return err
	}

	printValue("Field", "Value", "", prj)

	return nil
}

var showWorkspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Show the current workspace",
	RunE:  showWorkspaceRunE,
	Args:  cobra.RangeArgs(0, 1),
}

func showWorkspaceRunE(cmd *cobra.Command, args []string) error {

	prj, err := project.Load()
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
	Use:   "image [NAME]",
	Short: "Show image details",
	RunE:  showImageRunE,
	Args:  cobra.ExactArgs(1),
}

type OS struct {
	Name    string
	Version string
	ID      string
	ID_LIKE string
}

func showImageRunE(cmd *cobra.Command, args []string) error {

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	imgName := conf.FullImageName(args[0])
	img, err := run.GetImage(imgName)
	if err != nil && errors.Is(err, errdefs.ErrNotFound) {
		img, err = pullImage(run, imgName)
	}
	if err != nil {
		return err
	}

	fullName := "<unknown>"
	info, err := support.GetImageInfo(img)
	if info != nil {
		fullName = info.FullName
	}

	rootfs := []string{}
	imgRootFS, err := img.RootFS()
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
		&showUserConfig, "user", "", false, "Show only user configurations")
	showCmd.AddCommand(showProjectCmd)
	showCmd.AddCommand(showWorkspaceCmd)
	showCmd.AddCommand(showImageCmd)
}
