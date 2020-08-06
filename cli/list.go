package cli

import (
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/container"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

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
	printValue("Index", "Runtime", "", runtime.Runtimes())
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

	conf := config.Load()

	tPos := strings.Index(name, ":")
	dispName := name[:tPos]

	for rn, r := range conf.Registry {
		p := r.Domain + "/" + r.RepoName
		if strings.HasPrefix(dispName, p) {
			dispName = name[len(p)+1 : tPos]
			if rn != config.DefaultRegistryName {
				dispName = rn + "/" + dispName
			}
		}
	}

	return dispName, name[tPos+1:]
}

func listImagesRunE(cmd *cobra.Command, args []string) error {

	conf := config.Load()

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	images, err := run.Images()
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
		imgList[i].ID = digest[dPos+1 : dPos+1+displayHashLength]
		imgList[i].CreatedAt = timeToAgoString(img.CreatedAt())
		imgList[i].Size = sizeToSIString(img.Size())
	}
	printList(imgList)

	return nil
}

var listSnapshotsCmd = &cobra.Command{
	Use:     "snapshots",
	Aliases: []string{"snapshot", "i"},
	Short:   "list snapshots",
	Args:    cobra.NoArgs,
	RunE:    listSnapshotsRunE,
}

func listSnapshotsRunE(cmd *cobra.Command, args []string) error {

	conf := config.Load()

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	prj, err := project.Load()
	if err != nil {
		return err
	}

	dom, err := uuid.Parse(prj.UUID)
	if err != nil {
		return errdefs.InvalidArgument("invalid project UUID in workspace: '%v'", prj.UUID)
	}

	snapshots, err := run.Snapshots(dom)
	if err != nil {
		return err
	}

	snapList := make([]struct {
		Name      string
		Parent    string
		CreatedAt string
	}, len(snapshots), len(snapshots))

	for i, snap := range snapshots {
		snapList[i].Name = snap.Name()
		snapList[i].Parent = snap.Parent()
		snapList[i].CreatedAt = timeToAgoString(snap.CreatedAt())
	}
	printList(snapList)

	return nil
}

var listContainersCmd = &cobra.Command{
	Use:     "containers",
	Aliases: []string{"containers", "c"},
	Short:   "list containers",
	Args:    cobra.NoArgs,
	RunE:    listContainersRunE,
}

func listContainersRunE(cmd *cobra.Command, args []string) error {

	conf := config.Load()

	prj, err := project.Load()
	if err != nil {
		return err
	}

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	ctrs, err := container.Containers(run, prj)
	if err != nil {
		return err
	}

	ctrList := make([]struct {
		Name      string
		CreatedAt string
	}, len(ctrs), len(ctrs))

	for i, c := range ctrs {
		ctrList[i].Name = c.Name
		ctrList[i].CreatedAt = timeToAgoString(c.CreatedAt)
	}

	printList(ctrList)

	return nil
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listRuntimeCmd)
	listCmd.AddCommand(listImagesCmd)
	listCmd.AddCommand(listContainersCmd)
	listCmd.AddCommand(listSnapshotsCmd)
}
