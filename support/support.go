// Package support provides fuections for supporting the operating system of the underlying image.

package support

import (
	"bufio"
	"context"
	"fmt"
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

func SetupWorkspace(ctx context.Context, ws *project.Workspace, img runtime.Image) error {

	info, err := GetImageInfo(ctx, img)
	if err != nil {
		return err
	}

	_, layer, err := ws.CreateLayer(project.LayerNameOS, "")
	if err != nil {
		return nil
	}

	switch info.ID {
	case "ubuntu":
		err = UbuntuOSLayerInit(layer)
	case "debian":
		err = DebianOSLayerInit(layer)
	default:
		fmt.Printf("Uknown OS: %v\n", info.ID)
	}
	if err != nil {
		return err
	}

	return nil
}

// Try to identify the OS from the image.
// Returns nil if the OS couldn't be identified.
func GetImageInfo(ctx context.Context, img runtime.Image) (*ImageInfo, error) {

	tmpDir, err := os.MkdirTemp("/tmp", "cne-mount")
	if err != nil {
		return nil, errdefs.InternalError("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// FIXME: not working always ...
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

// InitHandler initializes a newly created layer for the specified handler name
func InitHandler(layer *project.Layer, handler string) error {
	switch handler {
	case project.LayerHandlerApt:
		return AptLayerInit(layer)
	case project.LayerHandlerUbuntu:
		return UbuntuOSLayerInit(layer)
	case project.LayerHandlerDebian:
		return UbuntuOSLayerInit(layer)
	default:
		return errdefs.InvalidArgument("handler: '%s' not supported", handler)
	}
}
