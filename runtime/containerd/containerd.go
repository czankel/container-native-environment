package containerd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/reference"

	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/runtime"
)

// containerdRuntime provides the runtime implementation for the containerd daemon
// For more information about containerd, see: https://github.com/containerd/containerd
type containerdRuntime struct {
	client  *containerd.Client
	context context.Context
}

type image struct {
	contdRuntime *containerdRuntime
	contdImage   containerd.Image
}

type containerdRuntimeType struct {
}

const contextName = "cne"

func init() {
	runtime.Register("containerd", &containerdRuntimeType{})
}

// Runtime Interface

// Open opens the containerd runtime under the default context name
func (r *containerdRuntimeType) Open(confRun config.Runtime) (runtime.Runtime, error) {

	c, err := containerd.New(confRun.SocketName)
	if err != nil {
		return nil, err
	}

	ctx := namespaces.WithNamespace(context.Background(), confRun.Namespace)

	return &containerdRuntime{
		client:  c,
		context: ctx,
	}, nil
}

// Close closes the client to containerd
func (run *containerdRuntime) Close() {
	run.client.Close()
}

// Images returns a list of all images available on the system
func (run *containerdRuntime) Images() ([]runtime.Image, error) {

	contdImgs, err := run.client.ListImages(run.context)
	if err != nil {
		return nil, err
	}

	imgs := make([]runtime.Image, len(contdImgs))
	for i, img := range contdImgs {
		imgs[i] = &image{
			contdRuntime: run,
			contdImage:   img,
		}
	}

	return imgs, nil
}

// Name returns the full image name
func (img image) Name() string {
	return img.contdImage.Name()
}

func (img image) CreatedAt() time.Time {
	/* Will be in containerd 1.3.3
	return img.contdImage.Metadata().CreatedAt
	*/
	return time.Now()
}

func (img image) Size() int64 {
	size, _ := img.contdImage.Size(img.contdRuntime.context)
	return size
}

// PullImage pulls the specified image by name from the default registry
func (run *containerdRuntime) PullImage(name string) (runtime.Image, error) {

	img, err := run.client.Pull(run.context, name, containerd.WithPullUnpack)
	if err == reference.ErrObjectRequired {
		return nil, runtime.Errorf("invalid image name '%s': %v", name, err)
	} else if err != nil {
		return nil, runtime.Errorf("pull image '%s' failed: %v", name, err)
	}

	return &image{
		contdRuntime: run,
		contdImage:   img,
	}, nil
}

// Config returns the image configuration
func (img *image) Config() (*v1.ImageConfig, error) {

	contdRun := img.contdRuntime
	ociDesc, err := img.contdImage.Config(contdRun.context)
	if err != nil {
		return nil, err
	}

	var (
		ociimage v1.Image
		config   v1.ImageConfig
	)

	switch ociDesc.MediaType {
	case v1.MediaTypeImageConfig, images.MediaTypeDockerSchema2Config:
		store := img.contdImage.ContentStore()
		blob, err := content.ReadBlob(contdRun.context, store, ociDesc)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(blob, &ociimage); err != nil {
			return nil, err
		}
		config = ociimage.Config
	default:
		return nil, fmt.Errorf("unknown image config media type %s", ociDesc.MediaType)
	}

	return &config, nil
}

// Digest returns the digest of the image
func (img *image) Digest() digest.Digest {

	imgConf, _ := img.contdImage.Config(img.contdRuntime.context)
	return imgConf.Digest
}
