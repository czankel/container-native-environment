package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/container"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/runtime"
)

// helper function to update container options
func updateContainerOptions(options map[string]string) error {

	// TODO: create custom boilerplate function
	prj, err := loadProject()
	if err != nil {
		return err
	}

	ws, err := prj.CurrentWorkspace()
	if err != nil {
		return err
	}

	ctx := context.Background()
	runCfg, err := conf.GetRuntime()
	if err != nil {
		return err
	}

	run, err := runtime.Open(ctx, runCfg)
	if err != nil {
		return err
	}

	defer run.Close()
	ctx = run.WithNamespace(ctx, runCfg.Namespace)

	ctr, err := container.GetContainer(ctx, run, ws)
	if err != nil && errors.Is(err, errdefs.ErrNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	return ctr.Update(ctx, options)
}

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

var updateRenameEntry string

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

	if updateRenameEntry != "" {
		entry := cmd.CalledAs()
		if len(args) == 0 {
			return errdefs.InvalidArgument("original name not provided")
		}
		err = tempConf.RenameEntry(entry, args[0], updateRenameEntry)
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

	if updateContextOptions != "" {
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

		if updateContextRuntime == "" {
			err = updateContainerOptions(confCtx.Options)
			if err != nil {
				return err
			}
		}
	}
	if updateContextRegistry != "" {
		if _, ok := tempConf.Registry[confCtx.Registry]; !ok {
			return errdefs.NotFound("registry", confCtx.Registry)
		}
		orig := confCtx.Registry
		confCtx.Registry = updateContextRegistry
		changes = append(changes, changeInfo{"Registry", orig, updateContextRegistry})
	}
	if updateContextRuntime != "" {
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

var updateRegistryCmd = &cobra.Command{
	Use:   "registry [name]",
	Short: "Update registry configurations",
	Args:  cobra.RangeArgs(0, 1),
	RunE:  updateContextRunE,
}

var updateRegistryDomain string
var updateRegistryRepoName string

func updateRegistryRunE(cmd *cobra.Command, args []string) error {

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

	if updateRenameEntry != "" {
		if len(args) == 0 {
			return errdefs.InvalidArgument("original name not provided")
		}
		err = tempConf.RenameEntry("registry", args[0], updateRenameEntry)
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

	if err == nil && updateRegistryDomain != "" {
		orig := confReg.Domain
		confReg.Domain = updateRegistryDomain
		changes = append(changes, changeInfo{"Domain", orig, updateRegistryDomain})
	}
	if err == nil && updateRegistryRepoName != "" {
		orig := confReg.RepoName
		confReg.RepoName = updateRegistryRepoName
		changes = append(changes, changeInfo{"RepoName", orig, updateRegistryRepoName})
	}

	err = writeConfig(tempConf)
	if err != nil {
		return err
	}

	printList(changes, false)
	return nil
}

var updateRuntimeCmd = &cobra.Command{
	Use:   "runtime [name]",
	Short: "Update runtime configurations",
	Args:  cobra.RangeArgs(0, 1),
	RunE:  updateRuntimeRunE,
}

var updateRuntimeRuntime string
var updateRuntimeSocketName string
var updateRuntimeNamespace string

func updateRuntimeRunE(cmd *cobra.Command, args []string) error {

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

	if updateRenameEntry != "" {
		if len(args) == 0 {
			return errdefs.InvalidArgument("original name not provided")
		}
		err = tempConf.RenameEntry("runtime", args[0], updateRenameEntry)
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

	if err == nil && updateRuntimeRuntime != "" {
		orig := confRun.Engine
		confRun.Engine = updateRuntimeRuntime
		changes = append(changes, changeInfo{"Runtime", orig, updateRuntimeRuntime})
	}
	if err == nil && updateRuntimeSocketName != "" {
		orig := confRun.SocketName
		confRun.SocketName = updateRuntimeSocketName
		changes = append(changes, changeInfo{"socketname", orig, updateRuntimeSocketName})
	}
	if err == nil && updateRuntimeNamespace != "" {
		orig := confRun.Namespace
		confRun.Namespace = updateRuntimeNamespace
		changes = append(changes, changeInfo{"namespace", orig, updateRuntimeNamespace})
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

	updateCmd.AddCommand(updateContextCmd)
	updateContextCmd.Flags().StringVar(
		&updateContextOptions, "options", "", "Container runtime options")
	updateContextCmd.Flags().StringVar(
		&updateContextRuntime, "runtime", "", "Change the runtime for the context")
	updateContextCmd.Flags().StringVar(
		&updateContextRegistry, "registry", "", "Change the registry for the context")
	updateContextCmd.Flags().StringVar(
		&updateRenameEntry, "rename", "", "Rename the entry")
	updateContextCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "Update system configuration")
	updateContextCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Update project configuration")

	updateCmd.AddCommand(updateProjectCmd)
	updateProjectCmd.Flags().StringVar(
		&updateProjectWorkspace, "workspace", "", "Change the current workspace for the project")
	updateProjectCmd.Flags().StringVar(
		&updateProjectWorkspace, "ws", "", "Change the current workspace for the project")

	updateCmd.AddCommand(updateRegistryCmd)
	updateRegistryCmd.Flags().StringVar(
		&updateRegistryDomain, "domain", "", "Change the registry domain address")
	updateRegistryCmd.Flags().StringVar(
		&updateRegistryRepoName, "reponame", "", "Change the registry repo-name")
	updateRegistryCmd.Flags().StringVar(
		&updateRenameEntry, "rename", "", "Rename the entry")
	updateRegistryCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "Update system configuration")
	updateRegistryCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Update project configuration")

	updateCmd.AddCommand(updateRuntimeCmd)
	updateRuntimeCmd.Flags().StringVar(
		&updateRuntimeRuntime, "runtime", "", "Change the container runtime")
	updateRuntimeCmd.Flags().StringVar(
		&updateRuntimeSocketName,
		"socketname", "", "Change the socket name to the container runtime")
	updateRuntimeCmd.Flags().StringVar(
		&updateRuntimeNamespace,
		"namespace", "", "Change the namespace for the container runtime")
	updateRuntimeCmd.Flags().StringVar(
		&updateRenameEntry, "rename", "", "Rename the entry")
	updateRuntimeCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "Update system configuration")
	updateRuntimeCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Update project configuration")

	updateCmd.AddCommand(updateWorkspaceCmd)
	updateWorkspaceCmd.Flags().StringVarP(
		&updateWorkspaceName, "rename", "", "", "Rename the workspace")
}
