// The package cli implements the command line interface.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
)

const cneServerCmdName = "cned"

var conf *config.Config
var user config.User
var params config.Parameters

var basename string
var cneVersion bool

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
	Run: cliRun,
}

var cliVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the version",
	Args:  cobra.NoArgs,
	Run:   cliVersionRun,
}

func cliRun(cmd *cobra.Command, args []string) {

	if cneVersion {
		cliVersionRun(cmd, args)
	}

	if len(args) == 0 {
		cmd.Help()
		os.Exit(0)
	}
}

func cliVersionRun(cmd *cobra.Command, args []string) {
	fmt.Printf("%s version %s\n", basename, config.CneVersion)
	os.Exit(0)
}

func init() {
	basename = filepath.Base(os.Args[0])
	exec, _ := filepath.EvalSymlinks(os.Args[0])

	if basename == cneServerCmdName {

		args := []string{exec, "server"}
		args = append(args, os.Args[1:]...)
		childPID, _ := syscall.ForkExec(exec, args,
			&syscall.ProcAttr{
				Env: os.Environ(),
				Sys: &syscall.SysProcAttr{
					Setsid: true,
				},
				Files: []uintptr{0, 1, 2},
			})
		fmt.Printf("process %d started as daemon.\n", childPID)

	} else {
		rootCmd.Use = basename
		rootCmd.Flags().BoolVar(
			&cneVersion, "version", false, "Get version information")
		rootCmd.PersistentFlags().StringVarP(
			&projectPath, "project", "P", "", "Project path")
		rootCmd.PersistentFlags().StringVarP(
			&config.ContextName, "context", "", "", "Context")
		rootCmd.AddCommand(cliVersionCmd)
	}

	cobra.OnInitialize(initConfig)
}

// Execute is the main entry point to the CLI. It executes the commands and arguments provided
// in os.Args[1:]
func Execute() error {

	var err error
	if basename == cneServerCmdName {
		err = serverCmd.Execute()
	} else {
		err = rootCmd.Execute()
	}

	if err != nil && errdefs.IsCneError(err) {
		err = fmt.Errorf("%s: %v", basename, err)
	}
	return err
}

func initConfig() {

	var err error

	conf, err = config.Load()
	if err != nil {
		fmt.Printf("%s: %v\n", basename, err)
		os.Exit(1)
	}

	user, err = conf.User()
	if err != nil {
		fmt.Printf("%s: %v\n", basename, err)
		os.Exit(1)
	}
}
