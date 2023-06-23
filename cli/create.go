package cli

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/opencontainers/image-spec/identity"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/czankel/cne/container"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
	"github.com/czankel/cne/support"
)

func initWorkspace(prj *project.Project, wsName, insert, imgName string) error {

	ws, err := prj.CreateWorkspace(wsName, "", insert)
	if err != nil {
		return err
	}

	if imgName != "" {
		runCfg, err := conf.GetRuntime()
		if err != nil {
			return err
		}

		ctx := context.Background()
		run, err := runtime.Open(ctx, runCfg)
		if err != nil {
			return err
		}
		defer run.Close()
		ctx = run.WithNamespace(ctx, runCfg.Namespace)

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

	tempConf, err := loadConfig()
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

	type changeInfo struct {
		Configuration string
		Value         string
	}
	var changes []changeInfo

	if createContextOptions != "" {
		err := confCtx.UpdateContextOptions(createContextOptions)
		if err != nil {
			return err
		}
		changes = append(changes, changeInfo{"Options", createContextOptions})
	}

	if createContextRegistry != "" {
		if _, ok := tempConf.Registry[createContextRegistry]; !ok {
			return errdefs.NotFound("registry", createContextRegistry)
		}
		confCtx.Registry = createContextRegistry
		changes = append(changes, changeInfo{"Registry", createContextRegistry})
	}

	if createContextRuntime != "" {
		if _, ok := tempConf.Runtime[createContextRuntime]; !ok {
			return errdefs.NotFound("runtime", createContextRuntime)
		}
		confCtx.Runtime = createContextRuntime
		changes = append(changes, changeInfo{"Runtime", createContextRuntime})
	}

	err = writeConfig(tempConf)
	if err != nil {
		return err
	}

	printList(changes, false)
	return nil
}

var createLayerCmd = &cobra.Command{
	Use:     "layer [name] [cmdline]",
	Short:   "Create a new layer",
	Aliases: []string{"l"},
	Args:    cobra.MinimumNArgs(1),
	RunE:    createLayerRunE,
}

var createLayerHandler string
var createLayerInsert string

func createLayerRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}

	runCfg, err := conf.GetRuntime()
	if err != nil {
		return err
	}

	ctx := context.Background()

	run, err := runtime.Open(ctx, runCfg)
	if err != nil {
		return err
	}
	defer run.Close()
	ctx = run.WithNamespace(ctx, runCfg.Namespace)

	ws, err := prj.CurrentWorkspace()
	if err != nil {
		return err
	}

	oldCtr, err := container.GetContainer(ctx, run, ws)
	if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
		return err
	}

	isTerminal := term.IsTerminal(int(os.Stdin.Fd()))
	if len(args) > 1 && !isTerminal {
		return errdefs.InvalidArgument("too many arguments")
	}

	var commands []project.Command
	if len(args) > 1 {
		commands = scanLine(args[1])
	} else if !isTerminal {

		commands, err = readCommands(os.Stdin)
		if err != nil {
			return err
		}
	}

	atIndex := -1
	if createLayerInsert != "" {
		for i, l := range ws.Environment.Layers {
			if l.Name == createLayerInsert {
				atIndex = i
				break
			}
		}
		if atIndex == -1 {
			return errdefs.InvalidArgument("invalid index")
		}
	}

	rebuildContainer := createLayerHandler != ""
	if createLayerHandler != "" {
		err = support.CreateSystemLayer(ws, args[0], atIndex)
		if err != nil {
			return err
		}
	} else {
		layerName := args[0]
		for _, n := range project.LayerHandlers {
			if layerName == n {
				return errdefs.InvalidArgument("%s is a reserved layer name, use --handler",
					layerName)
			}
		}

		_, layer, err := ws.CreateLayer(layerName, createLayerInsert)
		layer.Handler = createLayerHandler
		layer.Commands = commands
		if err != nil {
			return err
		}
		rebuildContainer = len(commands) > 0
	}

	if rebuildContainer {
		_, err := buildContainer(ctx, run, ws, -1)
		if err != nil {
			return err
		}
	}

	err = prj.Write()
	if err != nil {
		return err
	}
	if oldCtr != nil {
		// Ignore any errors, TOOD: add warning
		oldCtr.Delete(ctx)
	}

	return nil
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

	tempConf, err := loadConfig()
	if err != nil {
		return err
	}

	confReg, err := tempConf.CreateRegistry(args[0])
	if err != nil {
		return err
	}

	type changeInfo struct {
		Configuration string
		Value         string
	}
	var changes []changeInfo

	if createRegistryDomain != "" {
		confReg.Domain = createRegistryDomain
		changes = append(changes, changeInfo{"Domain", createRegistryDomain})
	}
	if createRegistryRepoName != "" {
		confReg.RepoName = createRegistryRepoName
		changes = append(changes, changeInfo{"RepoName", createRegistryRepoName})
	}

	err = writeConfig(tempConf)
	if err != nil {
		return err
	}

	printList(changes, false)
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

	tempConf, err := loadConfig()
	if err != nil {
		return err
	}

	confRun, err := tempConf.CreateRuntime(args[0], args[1])
	if err != nil {
		return err
	}

	type changeInfo struct {
		Configuration string
		Value         string
	}
	var changes []changeInfo

	if createRuntimeEngine != "" {
		confRun.Engine = createRuntimeEngine
		changes = append(changes, changeInfo{"SocketName", createRuntimeSocketName})
	}
	if createRuntimeSocketName != "" {
		confRun.SocketName = createRuntimeSocketName
		changes = append(changes, changeInfo{"SocketName", createRuntimeSocketName})
	}
	if createRuntimeNamespace != "" {
		confRun.Namespace = createRuntimeNamespace
		changes = append(changes, changeInfo{"Namespace", createRuntimeNamespace})
	}

	err = writeConfig(tempConf)
	if err != nil {
		return err
	}

	printList(changes, false)
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
		&createContextRuntime, "runtime", "", "Context registry")
	createContextCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "Update system configuration")
	createContextCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Update project configuration")

	createCmd.AddCommand(createRegistryCmd)
	createRegistryCmd.Flags().StringVar(
		&createRegistryDomain, "domain", "", "Registry domain")
	createRegistryCmd.Flags().StringVar(
		&createRegistryRepoName, "reponame", "", "Registry repooname")
	createRegistryCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "Update system configuration")
	createRegistryCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Update project configuration")

	createCmd.AddCommand(createRuntimeCmd)
	createRuntimeCmd.Flags().StringVar(
		&createRuntimeEngine, "engine", "", "Container engine")
	createRuntimeCmd.Flags().StringVar(
		&createRuntimeSocketName, "socketname", "", "Socket name")
	createRuntimeCmd.Flags().StringVar(
		&createRuntimeNamespace, "namespace", "", "Namespace")
	createRuntimeCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "Update system configuration")
	createRuntimeCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Update project configuration")

	createCmd.AddCommand(createWorkspaceCmd)
	createWorkspaceCmd.Flags().StringVar(
		&createWorkspaceImage, "image", "", "Base image for the workspace")
	createWorkspaceCmd.Flags().StringVar(
		&createWorkspaceInsert, "insert", "", "Insert before this workspace")

	createCmd.AddCommand(createLayerCmd)
	createLayerCmd.Flags().StringVar(
		&createLayerInsert, "insert", "", "Insert before this layer")
	createLayerCmd.Flags().StringVarP(
		&createLayerHandler, "handler", "h", "",
		"Handler of the layer, such as apt")
}
