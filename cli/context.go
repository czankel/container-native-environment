package cli

import (
	"errors"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/errdefs"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Show or set the current context",
	Long: `... can be overwritten using --project or --user
configurations, mainly the runtime but also other settings.`,
	RunE: contextRunE,
	Args: cobra.RangeArgs(0, 1),
}

var configUser bool

func contextRunE(cmd *cobra.Command, args []string) error {

	var name string
	var err error
	var ctx *config.Context

	if len(args) == 0 {
		ctx, name, err = conf.GetContext()
	} else {

		var tempConf *config.Config
		var found bool

		name = args[0]

		if ctx, found = conf.Context[name]; !found {
			return errdefs.NotFound("context", name)
		}

		_, err := loadProject()
		if configProject && err != nil {
			return err
		}
		if !configUser && !configProject && err == nil {
			configProject = true
			tempConf, err = loadConfig(false)
			configUser = err != nil
			configProject = err == nil
		}
		if configUser || (configProject && err == nil) {
			tempConf, err = loadConfig(false)
		}
		if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
			return err
		} else if err != nil {
			tempConf = &config.Config{Settings: config.Settings{}}
		}

		tempConf.Settings.Context = name
		err = writeConfig(tempConf)
	}

	if err != nil {
		return err
	}

	printList(map[string]*config.Context{name: ctx}, false)
	return nil
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.Flags().BoolVarP(
		&configUser, "system", "", false, "Set context for user")
	contextCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Set context for project")
}
