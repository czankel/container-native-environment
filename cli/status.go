package cli

import (
	"fmt"
	//	"os"
	//"path/filepath"
	//	"strings"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the local file and directories",
	Long: `
Output status of CNE`,
	Args: cobra.NoArgs,
	RunE: statusRunE,
}

func statusRunE(cmd *cobra.Command, args []string) error {

	prj, err := loadProject()
	if err != nil {
		return err
	}

	err = printImageStatus(prj)

	return nil
}

// internal helper to print the status of the files included in an image
// returns if any changes are pending
func imageStatus(prj *project.Project) (bool, error) {

	// FIXME: this is also called with "cne run"??

	root := filepath.Dir(prj.Path) + "/"
	ignore := []string{}

	if image.WalkOne(root, ignore) || image.WalkOne(mayberemoved) {
		fmt.Println("Changes not committed to the image:")
		fmt.Println("(use 'cne rebuild image' to rebuild the image with the changes)")

		// Walk1
		fmt.Printf("%s changed ...")
		// Walk2
		fmt.Printf("%s removed ...")
	}

	if image.WalkOne(root, ignore) {
		fmt.Println("Untracked files:")
		fmt.Println("(use 'cne add <file|path> ...' to include it in the image)")
		err := image.Walk(root, true, filter, ignore, func(path string, info os.FileInfo) {

			if info.IsDir() {
				fmt.Printf("    %s %d\n", name+"/", info.Size())
			} else {
				fmt.Printf("    %s %d\n", name, info.Size())
			}
		})
	}
	/*
	On branch remote
	nothing to commit, working tree clean
	*/
	return nil
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
