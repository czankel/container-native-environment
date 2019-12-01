package cli

import (
	"github.com/spf13/cobra"

	"github.com/czankel/cne/config"
)

var setCmd = &cobra.Command{
	Use:     "set",
	Aliases: []string{"s"},
	Args:    cobra.MinimumNArgs(1),
}

var setSystemConfig bool

var setConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Set the environment configuration",
	Long: `Set the user or system configuration for the environment.
By default, the configuration is written to the user configuration file.
The system option modifies the system-wide configuration file stored in
/etc, and requires system permissions.`,
	RunE: setConfigRunE,
	Args: cobra.ExactArgs(2),
}

func setConfigRunE(cmd *cobra.Command, args []string) error {

	var conf *config.Config
	if setSystemConfig {
		conf = config.LoadSystemConfig()
	} else {
		conf = config.LoadUserConfig()
	}

	name := args[0]
	newVal := args[1]

	path, oldVal, err := conf.GetByName(name)
	if err != nil {
		return err
	}

	_, err = conf.SetByName(name, newVal)
	if err != nil {
		return err
	}

	if setSystemConfig {
		err = conf.WriteSystemConfig()
	} else {
		err = conf.WriteUserConfig()
	}

	if err != nil {
		return err
	}

	printList([]struct {
		Configuration string
		Value         string
		Old           string
	}{{path, newVal, oldVal}})
	return nil
}

func init() {
	rootCmd.AddCommand(setCmd)
	setCmd.AddCommand(setConfigCmd)
	setConfigCmd.Flags().BoolVarP(
		&setSystemConfig, "system", "", false, "Set system configuration")
}
