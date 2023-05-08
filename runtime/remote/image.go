package remote

import (
	"context"
	"fmt"
	"time"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/service"
)

type image struct {
	remRuntime *remoteRuntime
	remImage   *service.Image
}

// FIXME: remote.image.Config not implemented??
func (img *image) Config(ctx context.Context) (*ocispec.ImageConfig, error) {
	fmt.Println("image.Config")

	var (
		//ociimage ocispec.Image
		config ocispec.ImageConfig
	)

	return &config, nil
}

func (img *image) Name() string {
	return img.remImage.Name
}

func (img *image) Size() int64 {
	return img.remImage.Size
}

func (img *image) Digest() digest.Digest {
	return digest.FromString(img.remImage.Digest)
}

func (img *image) CreatedAt() time.Time {
	fmt.Println("image.CreatedAt")
	return time.Now()
}

// FIXME: remote.image.RootFS not implemented
func (img *image) RootFS(ctx context.Context) ([]digest.Digest, error) {
	fmt.Println("image.RootFS")
	return nil, service.ConvPbErrorToGo(errdefs.NotImplemented())
}

// FIXME: remote.image.Mount not implemented
func (img *image) Mount(ctx context.Context, path string) error {
	fmt.Println("image.Umount")
	return service.ConvPbErrorToGo(errdefs.NotImplemented())
}

// FIXME: remote.image.Unmount not implemented
func (img *image) Unmount(ctx context.Context, path string) error {
	fmt.Println("image.Umount")
	return service.ConvPbErrorToGo(errdefs.NotImplemented())
}
