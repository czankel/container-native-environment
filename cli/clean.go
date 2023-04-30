package cli

import (
	"context"

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

	ctx := context.Background()
	run, err := runtime.Open(ctx, &conf.Runtime)
	if err != nil {
		return err
	}
	defer run.Close()
	ctx = run.WithNamespace(ctx, conf.Runtime.Name)

	if !cleanProjectAll {
		prj, err = loadProject()
		if err != nil {
			return err
		}
	}
	ctrs, err := container.Containers(ctx, run, prj, &user)
	if err != nil {
		return err
	}
	for _, c := range ctrs {
		c.Purge(ctx) // Ignore errors
	}

	// add all snapshots to a map
	tree := make(map[string]runtime.Snapshot)
	leaves := []string{}
	snaps, err := run.Snapshots(ctx)
	if err != nil {
		return err
	}
	for _, s := range snaps {
		tree[s.Name()] = s
		leaves = append(leaves, s.Name())
	}

	// exclude snapshots created extracting images
	imgs, err := run.Images(ctx)
	if err != nil {
		return err
	}
	for _, i := range imgs {
		rootfs, err := i.RootFS(ctx)
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
			err = run.DeleteSnapshot(ctx, s.Name())
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
