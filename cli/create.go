package cli

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/opencontainers/image-spec/identity"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
	"github.com/czankel/cne/support"
)

// helper function to get commands from the args list or terminal; first indicates the index of the first arg
func getCommands(args []string, first int) ([]project.Command, error) {

	isTerminal := term.IsTerminal(int(os.Stdin.Fd()))
	if len(args) > first+1 && !isTerminal {
		return nil, errdefs.InvalidArgument("too many arguments")
	}
	var commands []project.Command
	var err error
	if len(args) > 1 {
		commands = scanLine(args[first])
	} else if !isTerminal {
		commands, err = readCommands(os.Stdin)
		if err != nil {
			return nil, err
		}
	}
	return commands, nil
}

// helper function to update the container starting from the provided layerIdx
func updateContainer(ws *project.Workspace, layerIdx int) error {

	cfgRun, err := conf.GetRuntime()
	if err != nil {
		return err
	}

	ctx := context.Background()
	run, err := runtime.Open(ctx, cfgRun)
	if err != nil {
		return err
	}
	defer run.Close()
	ctx = run.WithNamespace(ctx, cfgRun.Namespace)

	_, err = buildContainer(ctx, run, ws, layerIdx)
	return err
}

func initWorkspace(prj *project.Project, wsName, insert, imgName string) error {

	ws, err := prj.CreateWorkspace(wsName, "", insert)
	if err != nil {
		return err
	}

	if imgName != "" {
		cfgRun, err := conf.GetRuntime()
		if err != nil {
			return err
		}

		ctx := context.Background()
		run, err := runtime.Open(ctx, cfgRun)
		if err != nil {
			return err
		}
		defer run.Close()
		ctx = run.WithNamespace(ctx, cfgRun.Namespace)

		imgName, err := getImageName(ctx, run, imgName)
		if err != nil {
			return err
		}

		img, err := pullImage(ctx, run, imgName)
		if err != nil {
			return err
		}

		diffIDs, err := img.RootFS(ctx)
		if err != nil {
			return err
		}

		rootName := identity.ChainID(diffIDs).String()
		_, err = run.GetSnapshot(ctx, rootName)
		if err != nil && errors.Is(err, errdefs.ErrNotFound) {
			progress := make(chan []runtime.ProgressStatus)
			var wg sync.WaitGroup
			defer wg.Wait()
			defer close(progress)
			wg.Add(1)
			go func() {
				defer wg.Done()
				showProgress(progress)
			}()
			err = img.Unpack(ctx, progress)
		}
		if err != nil {
			return err
		}

		prj.UpdateWorkspace(ws, imgName)

		err = support.SetupWorkspace(ctx, ws, img)
		if err != nil {
			return err
		}
	}

	prj.CurrentWorkspaceName = wsName

	return prj.Write()
}

var createCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new resource",
	Aliases: []string{"c"},
	Args:    cobra.MinimumNArgs(1),
}

var createCommandCmd = &cobra.Command{
	Use:   "command layer name [line]",
	Short: "Create a new command line",
	Args:  cobra.MinimumNArgs(2),
	RunE:  createCommandRunE,
}

var createCommandAt string

func createCommandRunE(cmd *cobra.Command, args []string) error {

	commands, err := getCommands(args, 2)
	if err != nil {
		return err
	}

	prj, err := loadProject()
	if err != nil {
		return err
	}

	ws, err := prj.CurrentWorkspace()
	if err != nil {
		return err
	}

	layerIdx, layer, err := ws.FindLayer(args[0])
	if err != nil {
		return err
	}

	if err := ws.InsertCommands(layer, createCommandAt, commands); err != nil {
		return err
	}

	err = updateContainer(ws, layerIdx)
	if err != nil {
		return err
	}

	return prj.Write()
}

var createContextCmd = &cobra.Command{
	Use:   "context name",
	Short: "Create a new context",
	Args:  cobra.ExactArgs(1),
	RunE:  createContextRunE,
}

var createContextOptions string
var createContextRegistry string
var createContextRuntime string

func createContextRunE(cmd *cobra.Command, args []string) error {

	tempConf, err := loadConfig(false)
	if err != nil {
		return err
	}

	confCtx, err := tempConf.CreateContext(args[0])
	if err != nil {
		return err
	}

	// set current runtime and registry as default
	c, _, err := conf.GetContext()
	if err != nil {
		return err
	}
	confCtx.Runtime = c.Runtime
	confCtx.Registry = c.Registry

	if createContextOptions != "" {
		err := confCtx.UpdateContextOptions(createContextOptions)
		if err != nil {
			return err
		}
	}

	if createContextRegistry != "" {
		if _, ok := tempConf.Registry[createContextRegistry]; !ok {
			return errdefs.NotFound("registry", createContextRegistry)
		}
		confCtx.Registry = createContextRegistry
	}

	if createContextRuntime != "" {
		if _, ok := tempConf.Runtime[createContextRuntime]; !ok {
			return errdefs.NotFound("runtime", createContextRuntime)
		}
		confCtx.Runtime = createContextRuntime
	}

	err = writeConfig(tempConf)
	if err != nil {
		return err
	}

	return nil
}

var createLayerCmd = &cobra.Command{
	Use:     "layer name [cmdline]",
	Short:   "Create a new layer",
	Aliases: []string{"l"},
	Args:    cobra.MinimumNArgs(1),
	RunE:    createLayerRunE,
}

var createLayerHandler string
var createLayerInsert string

func createLayerRunE(cmd *cobra.Command, args []string) error {

	cmds, err := getCommands(args, 1)
	if err != nil {
		return err
	}

	prj, err := loadProject()
	if err != nil {
		return err
	}

	ws, err := prj.CurrentWorkspace()
	if err != nil {
		return err
	}

	layerIdx, layer, err := ws.CreateLayer(args[0], createLayerInsert)
	if err != nil {
		return err
	}

	if createLayerHandler != "" {
		if err := support.InitHandler(layer, createLayerHandler); err != nil {
			return err
		}
	} else if err := ws.InsertCommands(layer, "", cmds); err != nil {
		return err
	}
	if err := updateContainer(ws, layerIdx); err != nil {
		return err
	}

	return prj.Write()
}

var createRegistryCmd = &cobra.Command{
	Use:   "registry name",
	Short: "Create a new registry",
	Args:  cobra.ExactArgs(1),
	RunE:  createRegistryRunE,
}

var createRegistryDomain string
var createRegistryRepoName string

func createRegistryRunE(cmd *cobra.Command, args []string) error {

	tempConf, err := loadConfig(false)
	if err != nil {
		return err
	}

	confReg, err := tempConf.CreateRegistry(args[0])
	if err != nil {
		return err
	}

	if createRegistryDomain != "" {
		confReg.Domain = createRegistryDomain
	}
	if createRegistryRepoName != "" {
		confReg.RepoName = createRegistryRepoName
	}

	err = writeConfig(tempConf)
	if err != nil {
		return err
	}

	return nil
}

var createRuntimeCmd = &cobra.Command{
	Use:   "runtime name runtime",
	Short: "Create a new runtime",
	Args:  cobra.ExactArgs(2),
	RunE:  createRuntimeRunE,
}

var createRuntimeEngine string
var createRuntimeSocketName string
var createRuntimeNamespace string

func createRuntimeRunE(cmd *cobra.Command, args []string) error {

	tempConf, err := loadConfig(false)
	if err != nil {
		return err
	}

	confRun, err := tempConf.CreateRuntime(args[0], args[1])
	if err != nil {
		return err
	}

	if createRuntimeEngine != "" {
		confRun.Engine = createRuntimeEngine
	}
	if createRuntimeSocketName != "" {
		confRun.SocketName = createRuntimeSocketName
	}
	if createRuntimeNamespace != "" {
		confRun.Namespace = createRuntimeNamespace
	}

	err = writeConfig(tempConf)
	if err != nil {
		return err
	}

	return nil
}

var createWorkspaceCmd = &cobra.Command{
	Use:     "workspace [name]",
	Short:   "Create a new workspace",
	Aliases: []string{"ws"},
	Args:    cobra.MaximumNArgs(1),
	RunE:    createWorkspaceRunE,
}

var createWorkspaceImage string
var createWorkspaceInsert string

func createWorkspaceRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}

	wsName := ""
	if len(args) != 0 {
		wsName = args[0]
	}

	return initWorkspace(prj, wsName, createWorkspaceImage, createWorkspaceInsert)
}

func init() {

	rootCmd.AddCommand(createCmd)

	createCmd.AddCommand(createContextCmd)
	createContextCmd.Flags().StringVar(
		&createContextOptions, "options", "", "Context options")
	createContextCmd.Flags().StringVar(
		&createContextRegistry, "registry", "", "Context registry")
	createContextCmd.Flags().StringVar(
		&createContextRuntime, "runtime", "", "Context runtime")
	createContextCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "System configuration")
	createContextCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Project configuration")

	createCmd.AddCommand(createLayerCmd)
	createLayerCmd.Flags().StringVar(
		&createLayerInsert, "insert", "", "Insert before this layer")
	createLayerCmd.Flags().StringVarP(
		&createLayerHandler, "handler", "h", "",
		"Handler for this layer. Use 'list handlers' for the list.")

	createCmd.AddCommand(createRegistryCmd)
	createRegistryCmd.Flags().StringVar(
		&createRegistryDomain, "domain", "", "Registry domain")
	createRegistryCmd.Flags().StringVar(
		&createRegistryRepoName, "reponame", "", "Registry repooname")
	createRegistryCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "System configuration")
	createRegistryCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Project configuration")

	createCmd.AddCommand(createRuntimeCmd)
	createRuntimeCmd.Flags().StringVar(
		&createRuntimeEngine, "engine", "", "Container engine")
	createRuntimeCmd.Flags().StringVar(
		&createRuntimeSocketName, "socketname", "", "Socket name")
	createRuntimeCmd.Flags().StringVar(
		&createRuntimeNamespace, "namespace", "", "Namespace")
	createRuntimeCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "System configuration")
	createRuntimeCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Project configuration")

	createCmd.AddCommand(createWorkspaceCmd)
	createWorkspaceCmd.Flags().StringVar(
		&createWorkspaceImage, "image", "", "Base image for the workspace")
	createWorkspaceCmd.Flags().StringVar(
		&createWorkspaceInsert, "insert", "", "Insert before this workspace")
}
