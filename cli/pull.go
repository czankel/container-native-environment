package cli

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/runtime"
)

var pullCmd = &cobra.Command{
	Use:   "pull [REGISTRY]PACKAGE[:TAG|@DIGEST]",
	Short: "Pull an image from a registry",
	Long: `
Pull an image from a registry to the local system.
REGISTRY can be one of the configured registries or directly
specify the domain and repository. If omitted, the default
registry is used.`,
	Args: cobra.ExactArgs(1),
	RunE: pullImageRunE,
}

func pullImageRunE(cmd *cobra.Command, args []string) error {

	name := args[0]

	conf := config.Load()
	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return errors.Wrap(err, "Failed to open runtime:")
	}

	reg, foundReg := conf.Registry[config.DefaultRegistryName]
	domEnd := strings.Index(name, "/") + 1
	if domEnd > 1 {
		reg, foundReg = conf.Registry[name[:domEnd-1]]
	}

	if foundReg {
		name = reg.Domain + "/" + reg.RepoName + "/" + name[domEnd:]
	}

	v := strings.LastIndex(name, ":")
	if v == -1 || v < domEnd {
		name = name + ":" + config.DefaultPackageVersion
	}

	_, err = run.PullImage(name)
	return err
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
