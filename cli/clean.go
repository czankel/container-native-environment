package cli

import (
	"github.com/spf13/cobra"

	"github.com/czankel/cne/container"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

var cleanCmd = &cobra.Command{
	Use: "clean",
}

var cleanProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Clean up all project resources, including images, containers and snapshots",
	Args:  cobra.NoArgs,
	RunE:  cleanProjectRunE,
}

var cleanProjectAll bool

func cleanProjectRunE(cmd *cobra.Command, args []string) error {

	var prj *project.Project

	run, err := runtime.Open(conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()

	if !cleanProjectAll {
		prj, err = project.Load()
		if err != nil {
			return err
		}
	}
	ctrs, err := container.Containers(run, prj, &user)
	if err != nil {
		return err
	}
	for _, c := range ctrs {
		c.Purge() // Ignore errors
	}

	// add all snapshots to a map
	tree := make(map[string]runtime.Snapshot)
	leaves := []string{}
	snaps, err := run.Snapshots()
	if err != nil {
		return err
	}
	for _, s := range snaps {
		tree[s.Name()] = s
		leaves = append(leaves, s.Name())
	}

	// exclude snapshots created extracting images
	imgs, err := run.Images()
	if err != nil {
		return err
	}
	for _, i := range imgs {
		rootfs, err := i.RootFS()
		if err != nil {
			return err
		}
		for _, r := range rootfs {
			delete(tree, r.String())
		}
	}

	// remove all non-leave nodes
	for _, s := range snaps {
		for i, l := range leaves {
			if l == s.Parent() {
				leaves = append(leaves[:i], leaves[i+1:]...)
				break
			}
		}
	}

	// delete snapshots
	for _, l := range leaves {
		for {
			s, ok := tree[l]
			if !ok {
				break
			}
			err = run.DeleteSnapshot(s.Name())
			if err != nil {
				break
			}
			l = s.Parent()
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.AddCommand(cleanProjectCmd)
	cleanProjectCmd.Flags().BoolVarP(
		&cleanProjectAll, "all", "A", false, "clean containers of all projects")
}
