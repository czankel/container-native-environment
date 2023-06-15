package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/errdefs"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update an existing resource",
	Aliases: []string{"u"},
	Args:    cobra.MinimumNArgs(1),
}

var updateWorkspaceCmd = &cobra.Command{
	Use:     "workspace [name]",
	Short:   "Update a workspace resources",
	Aliases: []string{"ws"},
	Args:    cobra.ExactArgs(1),
	RunE:    updateWorkspaceRunE,
}

var updateWorkspaceName string
var updateWorkspaceImage string

func updateWorkspaceRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}

	wsName := args[0]

	if updateWorkspaceName != "" {
		for _, ws := range prj.Workspaces {
			if ws.Name == updateWorkspaceName {
				return errdefs.AlreadyExists("workspace", updateWorkspaceName)
			} else if ws.Name == wsName {
				ws.Name = wsName
			}
		}
	}

	err = prj.Write()
	return err
}

var updateConfigCmd = &cobra.Command{
	Use:   "config config",
	Short: "Update the configuration",
	Long: `Update the user or system configuration for the environment.
By default, the configuration is written to the user configuration file.
The system option modifies the system-wide configuration file stored in
/etc, and requires system permissions.`,
}

var updateConfigRenameEntry string

var updateContextCmd = &cobra.Command{
	Use:   "context [context]",
	Short: "Update context configurations",
	Args:  cobra.RangeArgs(0, 1),
	RunE:  updateContextRunE,
}

var updateContextOptions string
var updateContextRuntime string
var updateContextRegistry string

func updateContextRunE(cmd *cobra.Command, args []string) error {

	tempConf, err := loadConfig()
	if err != nil {
		return err
	}

	name := conf.Settings.Context
	if len(args) > 0 {
		name = args[0]
	}

	confCtx, found := tempConf.Context[name]
	if !found {
		return errdefs.NotFound("context", name)
	}

	if updateConfigRenameEntry != "" {
		entry := cmd.CalledAs()
		if len(args) == 0 {
			return errdefs.InvalidArgument("original name not provided")
		}
		err = tempConf.RenameEntry(entry, args[0], updateConfigRenameEntry)
		if err != nil {
			return err
		}
	}

	type changeInfo struct {
		Configuration string
		OldValue      string
		NewValue      string
	}
	var changes []changeInfo

	if err == nil && updateContextOptions != "" {
		opts := make([]string, 0, len(confCtx.Options))
		for k, v := range confCtx.Options {
			opts = append(opts, k+"="+v)
		}
		orig := strings.Join(opts, ",")
		err := confCtx.UpdateContextOptions(updateContextOptions)
		if err != nil {
			return err
		}
		opts = make([]string, 0, len(confCtx.Options))
		for k, v := range confCtx.Options {
			opts = append(opts, k+"="+v)
		}
		changes = append(changes, changeInfo{"Options", orig, strings.Join(opts, ",")})
	}
	if err == nil && updateContextRegistry != "" {
		if _, ok := tempConf.Registry[confCtx.Registry]; !ok {
			return errdefs.NotFound("registry", confCtx.Registry)
		}
		orig := confCtx.Registry
		confCtx.Registry = updateContextRegistry
		changes = append(changes, changeInfo{"Registry", orig, updateContextRegistry})
	}
	if err == nil && updateContextRuntime != "" {
		if _, ok := tempConf.Runtime[confCtx.Runtime]; !ok {
			return errdefs.NotFound("runtime", confCtx.Runtime)
		}
		orig := confCtx.Runtime
		confCtx.Runtime = updateContextRuntime
		changes = append(changes, changeInfo{"Runtime", orig, updateContextRuntime})
	}
	err = writeConfig(tempConf)
	if err != nil {
		return err
	}

	printList(changes, false)
	return nil
}

var updateConfigRegistryCmd = &cobra.Command{
	Use:   "registry [name]",
	Short: "Update registry configurations",
	Args:  cobra.RangeArgs(0, 1),
	RunE:  updateContextRunE,
}

var updateConfigRegistryDomain string
var updateConfigRegistryRepoName string

func updateConfigRegistryRunE(cmd *cobra.Command, args []string) error {

	tempConf, err := loadConfig()
	if err != nil {
		return err
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	// find registry in merged configuration
	confReg, err := conf.GetRegistry(name)
	if err != nil {
		return err
	}

	if updateConfigRenameEntry != "" {
		if len(args) == 0 {
			return errdefs.InvalidArgument("original name not provided")
		}
		err = tempConf.RenameEntry("registry", args[0], updateConfigRenameEntry)
		if err != nil {
			return err
		}
	}

	type changeInfo struct {
		Configuration string
		OldValue      string
		NewValue      string
	}
	var changes []changeInfo

	if err == nil && updateConfigRegistryDomain != "" {
		orig := confReg.Domain
		confReg.Domain = updateConfigRegistryDomain
		changes = append(changes, changeInfo{"Domain", orig, updateConfigRegistryDomain})
	}
	if err == nil && updateConfigRegistryRepoName != "" {
		orig := confReg.RepoName
		confReg.RepoName = updateConfigRegistryRepoName
		changes = append(changes, changeInfo{"RepoName", orig, updateConfigRegistryRepoName})
	}

	err = writeConfig(tempConf)
	if err != nil {
		return err
	}

	printList(changes, false)
	return nil
}

var updateConfigRuntimeCmd = &cobra.Command{
	Use:   "runtime [name]",
	Short: "Update runtime configurations",
	Args:  cobra.RangeArgs(0, 1),
	RunE:  updateConfigRuntimeRunE,
}

var updateConfigRuntimeRuntime string
var updateConfigRuntimeSocketName string
var updateConfigRuntimeNamespace string

func updateConfigRuntimeRunE(cmd *cobra.Command, args []string) error {

	tempConf, err := loadConfig()
	if err != nil {
		return err
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	// find runtime in merged configuration
	confRun, err := conf.GetRuntime(name)
	if err != nil {
		return err
	}

	if updateConfigRenameEntry != "" {
		if len(args) == 0 {
			return errdefs.InvalidArgument("original name not provided")
		}
		err = tempConf.RenameEntry("runtime", args[0], updateConfigRenameEntry)
		if err != nil {
			return err
		}
	}

	type changeInfo struct {
		Configuration string
		OldValue      string
		NewValue      string
	}
	var changes []changeInfo

	if err == nil && updateConfigRuntimeRuntime != "" {
		orig := confRun.Runtime
		confRun.Runtime = updateConfigRuntimeRuntime
		changes = append(changes, changeInfo{"Runtime", orig, updateConfigRuntimeRuntime})
	}
	if err == nil && updateConfigRuntimeSocketName != "" {
		orig := confRun.SocketName
		confRun.SocketName = updateConfigRuntimeSocketName
		changes = append(changes, changeInfo{"socketname", orig, updateConfigRuntimeSocketName})
	}
	if err == nil && updateConfigRuntimeNamespace != "" {
		orig := confRun.Namespace
		confRun.Namespace = updateConfigRuntimeNamespace
		changes = append(changes, changeInfo{"namespace", orig, updateConfigRuntimeNamespace})
	}

	err = writeConfig(tempConf)
	if err != nil {
		return err
	}

	printList(changes, false)
	return nil
}

var updateProjectCmd = &cobra.Command{
	Use:     "project",
	Short:   "Update the project",
	Aliases: []string{"prj"},
	RunE:    updateProjectRunE,
	Args:    cobra.NoArgs,
}

var updateProjectWorkspace string

func updateProjectRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}

	err = prj.SetCurrentWorkspace(updateProjectWorkspace)
	if err != nil {
		return err
	}

	return prj.Write()
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.AddCommand(updateConfigCmd)
	updateConfigCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "Update system configuration")
	updateConfigCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Update project configuration")

	updateCmd.AddCommand(updateContextCmd)
	updateContextCmd.Flags().StringVar(
		&updateContextOptions, "options", "", "Container runtime options")
	updateContextCmd.Flags().StringVar(
		&updateContextRuntime, "runtime", "", "Change the runtime for the context")
	updateContextCmd.Flags().StringVar(
		&updateContextRegistry, "registry", "", "Change the registry for the context")
	updateContextCmd.Flags().StringVar(
		&updateConfigRenameEntry, "rename", "", "Rename the entry")

	updateConfigCmd.AddCommand(updateConfigRegistryCmd)
	updateConfigRegistryCmd.Flags().StringVar(
		&updateConfigRegistryDomain, "domain", "", "Change the registry domain address")
	updateConfigRegistryCmd.Flags().StringVar(
		&updateConfigRegistryRepoName, "reponame", "", "Change the registry repo-name")
	updateConfigRegistryCmd.Flags().StringVar(
		&updateConfigRenameEntry, "rename", "", "Rename the entry")

	updateConfigCmd.AddCommand(updateConfigRuntimeCmd)
	updateConfigRuntimeCmd.Flags().StringVar(
		&updateConfigRuntimeRuntime, "runtime", "", "Change the container runtime")
	updateConfigRuntimeCmd.Flags().StringVar(
		&updateConfigRuntimeSocketName,
		"socketname", "", "Change the socket name to the container runtime")
	updateConfigRuntimeCmd.Flags().StringVar(
		&updateConfigRuntimeNamespace,
		"namespace", "", "Change the namespace for the container runtime")
	updateConfigRuntimeCmd.Flags().StringVar(
		&updateConfigRenameEntry, "rename", "", "Rename the entry")

	updateCmd.AddCommand(updateProjectCmd)
	updateProjectCmd.Flags().StringVar(
		&updateProjectWorkspace, "workspace", "", "Change the current workspace for the project")
	updateProjectCmd.Flags().StringVar(
		&updateProjectWorkspace, "ws", "", "Change the current workspace for the project")

	updateCmd.AddCommand(updateWorkspaceCmd)
	updateWorkspaceCmd.Flags().StringVarP(
		&updateWorkspaceName, "rename", "", "", "Rename the workspace")

}
