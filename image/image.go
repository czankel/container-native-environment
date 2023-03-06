// Package image manages images and registries.

package image

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/czankel/cne/project"
)

// AddPath adds a path to the current image
func AddPath(prj *project.Project, paths []string) error {
	fmt.Println("AddPath", paths[0])
	//rootPath := filepath.Dir(prj.Path) + "/"

	return nil
}

// WalkDir walks the specified root dir and filters against path while ignoring
// paths listed in ignore. Paths in the filter are included if include is true,
// or excluded, otherwise.
func WalkDir(root string, include bool, filter []string, ignore []string,
	cb func(path string, info os.FileInfo) error) error {

	rootLen := len(root)
	err := filepath.Walk(root,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			found := false
			for i := 0; i < len(filter) && !found; i++ {
				found = strings.Contains(filter[i], path[rootLen:])
			}
			if found != include {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			return cb(path, info)
		})
	return err
}

// WalkOne is a helper to return if at least one path will be returned in Walk.
func WalkOne(root string, include bool, filter []string, ignore []string) (bool, error) {
	found := false
	err := image.Walk(root, untracked, ignore, func(path string, info os.FileInfo) error {
		found = true
		return path.End
	})
	return found, err
}

/*
name := path[rootLen:]
fmt.Printf("    %s %d\n", name+"/", info.Size())
fmt.Printf("    %s %d\n", name, info.Size())

rootPath := filepath.Dir(prj.Path) + "/"

fmt.Println("Changes not committed to the image:\n  (use 'cne update ...' to include it in the image)")
*/
