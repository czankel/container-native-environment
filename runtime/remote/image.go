package remote

import (
	"fmt"
	"time"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/czankel/cne/errdefs"
)

type image struct {
}

func (img *image) Config() (*ocispec.ImageConfig, error) {
	fmt.Println("image.Config")

	var (
		//ociimage ocispec.Image
		config ocispec.ImageConfig
	)

	return &config, nil
}

func (img *image) Digest() digest.Digest {
	fmt.Println("image.Digest")
	return "" // digest.Digest{}
}

func (img *image) RootFS() ([]digest.Digest, error) {
	fmt.Println("image.RootFS")
	return nil, errdefs.NotImplemented()
}

func (img *image) Name() string {
	fmt.Println("image.Name")
	return ""
}

func (img *image) CreatedAt() time.Time {
	fmt.Println("image.CreatedAt")
	return time.Now()
}

func (img *image) Size() int64 {
	fmt.Println("image.Size")
	return 0
}

func (img *image) Mount(path string) error {
	fmt.Println("image.Umount")
	return errdefs.NotImplemented()
}

func (img *image) Unmount(path string) error {
	fmt.Println("image.Umount")
	return errdefs.NotImplemented()
}
