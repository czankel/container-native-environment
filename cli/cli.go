// The package cli implements the command line interface.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
)

var conf *config.Config
var user config.User
var params config.Parameters

var basename string
var rootCneVersion bool

var projectPath string

var configSystem bool  // use sytem configuration file
var configProject bool // use project configuration file

// helper function to load a specific configuration file:
//
// system configuration, if configSystem is set
// project configuration, if configProject is set
// user configuration
func loadConfig() (*config.Config, error) {

	if configSystem {
		return config.LoadSystemConfig()
	} else if configProject {
		prj, err := loadProject()
		if err != nil {
			return nil, err
		}
		return config.LoadProjectConfig(filepath.Dir(prj.Path))
	}
	return config.LoadUserConfig()
}

// helper function to write the configuration to a specific file:
//
// system configuration if configSystem is set,
// project configuration if configProject is set
// user configuration
func writeConfig(conf *config.Config) error {
	if configSystem {
		return conf.WriteSystemConfig()
	} else if configProject {
		return conf.WriteProjectConfig(filepath.Dir(projectPath))
	}
	return conf.WriteUserConfig()
}

// helper function to load the project
func loadProject() (*project.Project, error) {

	prj, err := project.Load(projectPath)
	if err != nil {
		return nil, err
	}
	projectPath = prj.Path

	return prj, conf.UpdateProjectConfig(filepath.Dir(prj.Path))
}

var rootCmd = &cobra.Command{
	SilenceErrors: true,
	SilenceUsage:  true,
	Short:         "Container based environment and deployment tool.",
	Long: `
Container Native Environment (CNE) is a tool for building and managing
virtual environment based on containers to provide a reliable and
reproducible environment for development and other use cases, such as
machine learning or analytics.
`,
	Run: rootRun,
}

func rootRun(cmd *cobra.Command, args []string) {
	if rootCneVersion {
		rootVersionRun(cmd, args)
	}

	if len(args) == 0 {
		cmd.Help()
		os.Exit(0)
	}
}

var rootVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the version",
	Args:  cobra.NoArgs,
	Run:   rootVersionRun,
}

func rootVersionRun(cmd *cobra.Command, args []string) {
	fmt.Printf("%s version %s\n", basename, config.CneVersion)
	os.Exit(0)
}

// Execute is the main entry point to the CLI. It executes the commands and arguments provided
// in os.Args[1:]
func Execute() error {

	err := rootCmd.Execute()
	if err != nil && errdefs.IsCneError(err) {
		err = fmt.Errorf("%s: %v", basename, err)
	}
	return err
}

func initConfig() {

	var err error
	basename = filepath.Base(os.Args[0])

	conf, err = config.Load()
	if err != nil {
		fmt.Printf("%s: %v\n", basename, err)
		os.Exit(1)
	}

	if projectPath == "" {
		projectPath, err = os.Getwd()
	}
	if err == nil {
		projectPath, err = project.GetProjectPath(projectPath)
	}
	if err == nil {
		user, err = conf.User()
	}

	if err != nil {
		fmt.Printf("%s: %v\n", basename, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Use = filepath.Base(os.Args[0])
	rootCmd.Flags().BoolVar(
		&rootCneVersion, "version", false, "Get version information")
	rootCmd.PersistentFlags().StringVarP(
		&projectPath, "path", "P", "", "Projet path")
	// Remove the -h help shorthand
	rootCmd.PersistentFlags().BoolP("help", "", false, "help for cne")
	rootCmd.AddCommand(rootVersionCmd)
	cobra.OnInitialize(initConfig)
}
