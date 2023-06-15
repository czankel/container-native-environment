package cli

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/container"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

func getLayer(prj *project.Project,
	wsName, layerName string) (*project.Workspace, *project.Layer, error) {

	var ws *project.Workspace
	var err error
	if wsName != "" {
		ws, err = prj.Workspace(wsName)
	} else {
		ws, err = prj.CurrentWorkspace()
	}
	if err != nil {
		return nil, nil, err
	}

	if len(ws.Environment.Layers) == 0 {
		return nil, nil, errdefs.InvalidArgument("No layers in workspace")
	}

	index := len(ws.Environment.Layers) - 1
	if layerName != "" {
		index, _ = ws.FindLayer(layerName)
		if index == -1 {
			index, err = strconv.Atoi(layerName)
			if err != nil {
				return nil, nil,
					errdefs.InvalidArgument("No such layer: %s", layerName)
			}
			if index < 0 || index > len(ws.Environment.Layers)-1 {
				return nil, nil,
					errdefs.InvalidArgument("Layer index %d out of range", index)
			}
		}
	}
	return ws, &ws.Environment.Layers[index], nil
}

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List resources",
	Aliases: []string{"l"},
	Args:    cobra.MinimumNArgs(1),
}

var listRuntimeCmd = &cobra.Command{
	Use:     "runtimes",
	Aliases: []string{"runtime", "r"},
	Short:   "list runtimes",
	Args:    cobra.NoArgs,
	RunE:    listRuntimeRunE,
}

func listRuntimeRunE(cmd *cobra.Command, args []string) error {
	printList(runtime.Runtimes(), false)
	return nil
}

var listCommandsCmd = &cobra.Command{
	Use:     "commands [name]",
	Short:   "List all commands",
	Aliases: []string{"command", "cmd"},
	RunE:    listCommandsRunE,
	Args:    cobra.NoArgs,
}

var listCommandsWorkspace string
var listCommandsLayer string

func listCommandsRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}

	_, layer, err := getLayer(prj, listCommandsWorkspace, listCommandsLayer)
	if err != nil {
		return err
	}

	printList(layer.Commands, true)

	return nil
}

var listImagesCmd = &cobra.Command{
	Use:     "images",
	Aliases: []string{"image", "i"},
	Short:   "list images",
	Args:    cobra.NoArgs,
	RunE:    listImagesRunE,
}

const displayHashLength = 8

// splitRepoNameTag splits the provided full name to the image name and tag
// and resolves any respository aliases from the registered repositories.
// The default repository is omitted in the name.
func splitRepoNameTag(name string) (string, string) {

	tPos := strings.Index(name, ":")
	dispName := name[:tPos]

	for n, r := range conf.Registry {
		p := r.Domain + "/" + r.RepoName
		if strings.HasPrefix(dispName, p) {
			dispName = name[len(p)+1 : tPos]
			if n != config.DefaultRegistryName {
				dispName = n + "/" + dispName
			}
		}
	}

	return dispName, name[tPos+1:]
}

func listImages(ctx context.Context, run runtime.Runtime) error {

	images, err := run.Images(ctx)
	if err != nil {
		return err
	}

	imgList := make([]struct {
		Name      string
		Tag       string
		ID        string
		CreatedAt string
		Size      string
	}, len(images), len(images))
	for i, img := range images {
		name, tag := splitRepoNameTag(img.Name())
		imgList[i].Name = name
		imgList[i].Tag = tag
		digest := img.Digest().String()
		dPos := strings.Index(digest, ":")
		if len(digest) >= dPos+1+displayHashLength {
			imgList[i].ID = digest[dPos+1 : dPos+1+displayHashLength]
		} else {
			imgList[i].ID = ""
		}
		imgList[i].CreatedAt = timeToAgoString(img.CreatedAt())
		imgList[i].Size = sizeToSIString(img.Size())
	}
	printList(imgList, true)

	return nil
}

func listImagesRunE(cmd *cobra.Command, args []string) error {

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

	return listImages(ctx, run)
}

var listSnapshotsCmd = &cobra.Command{
	Use:     "snapshots",
	Aliases: []string{"snapshot", "s"},
	Short:   "list snapshots",
	Args:    cobra.NoArgs,
	RunE:    listSnapshotsRunE,
}

func listSnapshots(ctx context.Context, run runtime.Runtime) error {

	snapshots, err := run.Snapshots(ctx)
	if err != nil {
		return err
	}

	snapList := make([]struct {
		Name      string
		Parent    string
		CreatedAt string
		Size      int64
		Inodes    int64
	}, len(snapshots), len(snapshots))

	for i, snap := range snapshots {
		snapList[i].Name = snap.Name()
		snapList[i].Parent = snap.Parent()
		snapList[i].CreatedAt = timeToAgoString(snap.CreatedAt())
		snapList[i].Size = snap.Size()
		snapList[i].Inodes = snap.Inodes()
	}
	printList(snapList, false)

	return nil
}

func listSnapshotsRunE(cmd *cobra.Command, args []string) error {

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

	return listSnapshots(ctx, run)
}

var listContainersCmd = &cobra.Command{
	Use:     "containers",
	Aliases: []string{"c"},
	Short:   "list containers",
	Args:    cobra.NoArgs,
	RunE:    listContainersRunE,
}

var listContainersAll bool

func listContainers(ctx context.Context, run runtime.Runtime, prj *project.Project) error {

	ctrs, err := container.Containers(ctx, run, prj, &user)
	if err != nil {
		return err
	}

	ctrList := make([]struct {
		Name      string
		CreatedAt string
	}, len(ctrs), len(ctrs))

	for i, c := range ctrs {
		ctrList[i].Name = c.Name()
		ctrList[i].CreatedAt = timeToAgoString(c.CreatedAt())
	}

	printList(ctrList, false)

	return nil
}

func listContainersRunE(cmd *cobra.Command, args []string) error {

	var prj *project.Project

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

	if !listContainersAll {
		prj, err = loadProject()
		if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
			return err
		}
	}

	return listContainers(ctx, run, prj)
}

var listConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "list configurations",
}

var listConfigContextCmd = &cobra.Command{
	Use:   "context",
	Short: "list available contexts",
	Args:  cobra.NoArgs,
	RunE:  listConfigContextRunE,
}

var listConfigRuntimeCmd = &cobra.Command{
	Use:   "registry",
	Short: "list available registries",
	Args:  cobra.NoArgs,
	RunE:  listConfigRegistryRunE,
}
var listConfigRegistryCmd = &cobra.Command{
	Use:   "runtime",
	Short: "list available runtimes",
	Args:  cobra.NoArgs,
	RunE:  listConfigRuntimeRunE,
}

func listConfigContextRunE(cmd *cobra.Command, args []string) error {
	printList(conf.Context, false)
	return nil
}
func listConfigRegistryRunE(cmd *cobra.Command, args []string) error {
	printList(conf.Registry, false)
	return nil
}
func listConfigRuntimeRunE(cmd *cobra.Command, args []string) error {
	printList(conf.Runtime, false)
	return nil
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listRuntimeCmd)
	listCmd.AddCommand(listImagesCmd)
	listCmd.AddCommand(listContainersCmd)
	listContainersCmd.Flags().BoolVarP(
		&listContainersAll, "all", "A", false, "list containers of all projects")
	listCmd.AddCommand(listSnapshotsCmd)
	listCmd.AddCommand(listCommandsCmd)
	listCommandsCmd.Flags().StringVarP(
		&listCommandsWorkspace, "workspace", "w", "", "Name of the workspace")
	listCommandsCmd.Flags().StringVarP(
		&listCommandsLayer, "layer", "l", "", "Name or index of the layer")
	listCmd.AddCommand(listConfigCmd)
	listConfigCmd.AddCommand(listConfigContextCmd)
	listConfigCmd.AddCommand(listConfigRuntimeCmd)
	listConfigCmd.AddCommand(listConfigRegistryCmd)
}
