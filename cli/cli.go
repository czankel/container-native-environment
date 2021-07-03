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

var basenamee string
var rootCneVersion bool

var projectPath string

// helper function to load the project
func loadProject() (*project.Project, error) {

	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return nil, errdefs.SystemError(err,
				"failed to load project file in '%s'",
				projectPath)
		}
	}

	return project.Load(projectPath)
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

var rootVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the version",
	Args:  cobra.NoArgs,
	Run:   rootVersionRun,
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

func rootVersionRun(cmd *cobra.Command, args []string) {
	fmt.Printf("%s version %s\n", basenamee, config.CneVersion)
	os.Exit(0)
}

func init() {
	rootCmd.Use = filepath.Base(os.Args[0])
	rootCmd.Flags().BoolVar(
		&rootCneVersion, "version", false, "Get version information")
	rootCmd.PersistentFlags().StringVarP(
		&projectPath, "project", "P", "", "Projet path")
	rootCmd.AddCommand(rootVersionCmd)
	cobra.OnInitialize(initConfig)
}

// Execute is the main entry point to the CLI. It executes the commands and arguments provided
// in os.Args[1:]
func Execute() error {

	err := rootCmd.Execute()
	if err != nil && errdefs.IsCneError(err) {
		err = fmt.Errorf("%s: %v", basenamee, err)
	}
	return err
}

func initConfig() {

	var err error
	basenamee = filepath.Base(os.Args[0])

	conf, err = config.Load()
	if err != nil {
		fmt.Printf("%s: %v\n", basenamee, err)
		os.Exit(1)
	}

	user, err = conf.User()
	if err != nil {
		fmt.Printf("%s: %v\n", basenamee, err)
		os.Exit(1)
	}
}
