// Package containerd implements the runtime interface for the ContainerD Dameon containerd.io
package containerd

import (
	"encoding/json"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/czankel/cne/runtime"
)

type image struct {
	ctrdRuntime *containerdRuntime
	ctrdImage   containerd.Image
}

func (img *image) Config() (*ocispec.ImageConfig, error) {

	ctrdRun := img.ctrdRuntime
	ociDesc, err := img.ctrdImage.Config(ctrdRun.context)
	if err != nil {
		return nil, runtime.Errorf("failed to get image configuration: %v", err)
	}

	var (
		ociimage ocispec.Image
		config   ocispec.ImageConfig
	)

	switch ociDesc.MediaType {
	case ocispec.MediaTypeImageConfig, images.MediaTypeDockerSchema2Config:
		store := img.ctrdImage.ContentStore()
		blob, err := content.ReadBlob(ctrdRun.context, store, ociDesc)
		if err != nil {
			return nil, runtime.Errorf("failed to read image configuration: %v", err)
		}

		if err := json.Unmarshal(blob, &ociimage); err != nil {
			return nil, runtime.Errorf("error in image YAML configuration: %v", err)
		}
		config = ociimage.Config
	default:
		return nil, runtime.Errorf("unknown image config media type %s", ociDesc.MediaType)
	}

	return &config, nil
}

func (img *image) Digest() digest.Digest {
	imgConf, _ := img.ctrdImage.Config(img.ctrdRuntime.context)
	return imgConf.Digest
}

func (img *image) Name() string {
	return img.ctrdImage.Name()
}

func (img *image) CreatedAt() time.Time {
	/* TODO: Image.Metadata()Is supposed to be available in containerd 1.3.3
	   return img.ctrdImage.Metadata().CreatedAt
	*/
	return time.Now()
}

func (img *image) Size() int64 {
	size, _ := img.ctrdImage.Size(img.ctrdRuntime.context)
	return size
}
