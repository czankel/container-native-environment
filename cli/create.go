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

var createCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new resource",
	Aliases: []string{"c"},
	Args:    cobra.MinimumNArgs(1),
}

var createConfigCmd = &cobra.Command{
	Use: "config",
}

var createConfigContextCmd = &cobra.Command{
	Use:   "context name",
	Short: "Create a new context",
	Args:  cobra.ExactArgs(1),
	RunE:  createConfigContextRunE,
}

var createConfigContextOptions string
var createConfigContextRegistry string
var createConfigContextRuntime string

func createConfigContextRunE(cmd *cobra.Command, args []string) error {

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

	if createConfigContextOptions != "" {
		err := confCtx.UpdateContextOptions(createConfigContextOptions)
		if err != nil {
			return err
		}
		changes = append(changes, changeInfo{"Options", createConfigContextOptions})
	}

	if createConfigContextRegistry != "" {
		if _, ok := tempConf.Registry[createConfigContextRegistry]; !ok {
			return errdefs.NotFound("runtime", createConfigContextRegistry)
		}
		confCtx.Registry = createConfigContextRegistry
		changes = append(changes, changeInfo{"Registry", createConfigContextRegistry})
	}

	if createConfigContextRuntime != "" {
		if _, ok := tempConf.Runtime[createConfigContextRuntime]; !ok {
			return errdefs.NotFound("runtime", createConfigContextRuntime)
		}
		confCtx.Runtime = createConfigContextRuntime
		changes = append(changes, changeInfo{"Runtime", createConfigContextRuntime})
	}

	err = writeConfig(tempConf)
	if err != nil {
		return err
	}

	printList(changes, false)
	return nil
}

var createConfigRegistryCmd = &cobra.Command{
	Use:   "registry name",
	Short: "Create a new registry",
	Args:  cobra.ExactArgs(1),
	RunE:  createConfigRegistryRunE,
}

var createConfigRegistryDomain string
var createConfigRegistryRepoName string

func createConfigRegistryRunE(cmd *cobra.Command, args []string) error {

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

	if createConfigRegistryDomain != "" {
		confReg.Domain = createConfigRegistryDomain
		changes = append(changes, changeInfo{"Domain", createConfigRegistryDomain})
	}
	if createConfigRegistryRepoName != "" {
		confReg.RepoName = createConfigRegistryRepoName
		changes = append(changes, changeInfo{"RepoName", createConfigRegistryRepoName})
	}

	err = writeConfig(tempConf)
	if err != nil {
		return err
	}

	printList(changes, false)
	return nil
}

var createConfigRuntimeCmd = &cobra.Command{
	Use:   "runtime name runtime",
	Short: "Create a new runtime",
	Args:  cobra.ExactArgs(2),
	RunE:  createConfigRuntimeRunE,
}

var createConfigRuntimeRuntime string
var createConfigRuntimeSocketName string
var createConfigRuntimeNamespace string

func createConfigRuntimeRunE(cmd *cobra.Command, args []string) error {

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

	if createConfigRuntimeRuntime != "" {
		confRun.Runtime = createConfigRuntimeRuntime
		changes = append(changes, changeInfo{"SocketName", createConfigRuntimeSocketName})
	}
	if createConfigRuntimeSocketName != "" {
		confRun.SocketName = createConfigRuntimeSocketName
		changes = append(changes, changeInfo{"SocketName", createConfigRuntimeSocketName})
	}
	if createConfigRuntimeNamespace != "" {
		confRun.Namespace = createConfigRuntimeNamespace
		changes = append(changes, changeInfo{"Namespace", createConfigRuntimeNamespace})
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

func initWorkspace(prj *project.Project, wsName, insert, imgName string) error {

	imgName, err := conf.FullImageName(imgName)
	if err != nil {
		return err
	}

	ws, err := prj.CreateWorkspace(wsName, imgName, insert)
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
			go func() {
				defer wg.Done()
				wg.Add(1)
				showProgress(progress)
			}()
			err = img.Unpack(ctx, progress)
		}
		if err != nil {
			return err
		}

		err = support.SetupWorkspace(ctx, ws, img)
		if err != nil {
			return err
		}
	}

	prj.CurrentWorkspaceName = wsName

	return prj.Write()
}

var createLayerSystem bool
var createLayerInsert string

var createLayerCmd = &cobra.Command{
	Use:     "layer [name] [cmdline]",
	Short:   "Create a new layer",
	Aliases: []string{"l"},
	Args:    cobra.MinimumNArgs(1),
	RunE:    createLayerRunE,
}

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

	rebuildContainer := createLayerSystem
	if createLayerSystem {
		err = support.CreateSystemLayer(ws, args[0], atIndex)
		if err != nil {
			return err
		}
	} else {
		layerName := args[0]
		for _, n := range project.SystemLayerTypes {
			if layerName == n {
				return errdefs.InvalidArgument("%s is a reserved layer name, use --system",
					layerName)
			}
		}

		layer, err := ws.CreateLayer(createLayerSystem, layerName, atIndex)
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

func init() {

	rootCmd.AddCommand(createCmd)

	createCmd.AddCommand(createConfigCmd)
	createConfigCmd.Flags().BoolVarP(
		&configSystem, "system", "", false, "Update system configuration")
	createConfigCmd.Flags().BoolVarP(
		&configProject, "project", "", false, "Update project configuration")

	createConfigCmd.AddCommand(createConfigContextCmd)
	createConfigContextCmd.Flags().StringVar(
		&createConfigContextOptions, "options", "", "Context options")
	createConfigContextCmd.Flags().StringVar(
		&createConfigContextRegistry, "registry", "", "Context registry")
	createConfigContextCmd.Flags().StringVar(
		&createConfigContextRuntime, "runtime", "", "Context registry")

	createConfigCmd.AddCommand(createConfigRegistryCmd)
	createConfigRegistryCmd.Flags().StringVar(
		&createConfigRegistryDomain, "domain", "", "Registry domain")
	createConfigRegistryCmd.Flags().StringVar(
		&createConfigRegistryRepoName, "reponame", "", "Registry repooname")

	createConfigCmd.AddCommand(createConfigRuntimeCmd)
	createConfigRuntimeCmd.Flags().StringVar(
		&createConfigRuntimeRuntime, "runtime", "", "Container runtime")
	createConfigRuntimeCmd.Flags().StringVar(
		&createConfigRuntimeSocketName, "socketname", "", "Socket name")
	createConfigRuntimeCmd.Flags().StringVar(
		&createConfigRuntimeNamespace, "namespace", "", "Namespace")

	createCmd.AddCommand(createWorkspaceCmd)
	createWorkspaceCmd.Flags().StringVar(
		&createWorkspaceImage, "image", "", "Base image for the workspace")
	createWorkspaceCmd.Flags().StringVar(
		&createWorkspaceInsert, "insert", "", "Insert before this workspace")

	createCmd.AddCommand(createLayerCmd)
	createLayerCmd.Flags().StringVar(
		&createLayerInsert, "insert", "", "Insert before this layer")
	createLayerCmd.Flags().BoolVarP(
		&createLayerSystem, "system", "s", false,
		"User the system handler of the same name")
}
