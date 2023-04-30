// Package support provides fuections for supporting the operating system of the underlying image.

package support

import (
	"bufio"
	"context"
	"io/ioutil"
	"os"
	"strings"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/project"
	"github.com/czankel/cne/runtime"
)

type ImageInfo struct {
	FullName string
	ID       string
	Version  string
}

func SetupWorkspace(ws *project.Workspace, img runtime.Image) error {

	info, err := GetImageInfo(img)
	if err != nil {
		return err
	}

	switch info.ID {
	case "ubuntu":
		err = UbuntuCreateOSLayer(ws, -1)
	}
	if err != nil {
		return err
	}

	return nil
}

// Try to identify the OS from the image.
// Returns nil if the OS couldn't be identified.
func GetImageInfo(img runtime.Image) (*ImageInfo, error) {

	tmpDir, err := ioutil.TempDir("/tmp", "cne-mount")
	if err != nil {
		return nil, errdefs.InternalError("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	err = img.Mount(ctx, tmpDir)
	if err != nil {
		return nil, err
	}
	defer img.Unmount(ctx, tmpDir)

	// scan os-release fields, returns nil if parsing fails
	f, err := os.Open(tmpDir + "/etc/os-release")
	if err != nil {
		return nil, nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var imageinfo ImageInfo
	for scanner.Scan() {

		line := scanner.Text()
		v := strings.Split(line, "=")
		if len(v) != 2 {
			return nil, nil
		}
		key := strings.Trim(v[0], " ")
		val := strings.Trim(v[1], " ")
		val = strings.Trim(val, "\"")

		switch key {
		case "ID":
			imageinfo.ID = val
		case "VERSION_ID":
			imageinfo.Version = val
		case "PRETTY_NAME":
			imageinfo.FullName = val
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil
	}

	return &imageinfo, nil
}

// CreateSystemLayer creates a system layer
func CreateSystemLayer(ws *project.Workspace, name string, atIndex int) error {
	switch name {
	case project.LayerTypeApt:
		return AptCreateLayer(ws, atIndex)
	case project.LayerTypeUbuntu:
		return UbuntuCreateOSLayer(ws, atIndex)
	default:
		return errdefs.InvalidArgument("system layer %s not supported", name)
	}
}
