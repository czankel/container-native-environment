// Package containerd implements the runtime interface for the ContainerD Dameon containerd.io
package containerd

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/content"
	ctrderr "github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/remotes"

	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/identity"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/czankel/cne/runtime"
)

const updateIntervalMsecs = 100

// updateImageProgress sends the current image download status in a regular 100ms interval
// to the provided progress channel.
func updateImageProgress(ctx context.Context,
	ctrdRun *containerdRuntime, mutex *sync.Mutex, descs *[]ocispec.Descriptor,
	progress chan<- []runtime.ProgressStatus) {

	var (
		ticker = time.NewTicker(time.Duration(updateIntervalMsecs) * time.Millisecond)
		start  = time.Now()
	)

	defer ticker.Stop()
	cs := ctrdRun.client.ContentStore()

	for loop := true; loop; {
		var statuses []runtime.ProgressStatus
		var err error

		select {
		case <-ticker.C:
			statuses, err = updateImageStatus(ctx, start, cs, mutex, descs)

		case <-ctx.Done():
			statuses, err = updateImageStatus(ctx, start, cs, mutex, descs)
			loop = false
		}

		if err != nil {
			break
		}

		progress <- statuses
	}
}

// updateImageStatus sends the status of the current image download to the progress channel.
func updateImageStatus(ctx context.Context, start time.Time, cs content.Store,
	mutex *sync.Mutex, descs *[]ocispec.Descriptor) ([]runtime.ProgressStatus, error) {

	statuses := []runtime.ProgressStatus{}
	actStats := map[string]*runtime.ProgressStatus{}

	active, err := cs.ListStatuses(ctx, "")
	if err == nil {
		for _, active := range active {
			if !strings.HasPrefix(active.Ref, "layer-") {
				continue
			}
			statuses = append(statuses, runtime.ProgressStatus{
				Reference: active.Ref,
				Status:    runtime.StatusRunning,
				Offset:    active.Offset,
				Total:     active.Total,
				StartedAt: active.StartedAt,
				UpdatedAt: active.UpdatedAt,
			})
			actStats[active.Ref] = &statuses[len(statuses)-1]
		}
	}

	mutex.Lock()
	defer mutex.Unlock()

	for _, desc := range *descs {

		ref := remotes.MakeRefKey(ctx, desc)
		if !strings.HasPrefix(ref, "layer-") {
			continue
		}
		if _, isActive := actStats[ref]; isActive {
			continue
		}

		info, err := cs.Info(ctx, desc.Digest)
		if err != nil && !ctrderr.IsNotFound(err) {
			continue
		}

		stat := runtime.ProgressStatus{
			Reference: ref,
			Status:    runtime.StatusUnknown,
		}
		if err != nil && ctrderr.IsNotFound(err) {
			stat.Status = runtime.StatusPending
		} else if err == nil {
			if info.CreatedAt.After(start) {
				if _, done := info.Labels["containerd.io/uncompressed"]; done {
					stat.Status = runtime.StatusComplete
				} else {
					stat.Status = runtime.StatusRunning
				}
			} else {
				stat.Status = runtime.StatusExists
			}
			stat.Offset = info.Size
			stat.Total = info.Size
			stat.UpdatedAt = info.CreatedAt
		}
		statuses = append(statuses, stat)
	}

	return statuses, nil
}

type image struct {
	ctrdRuntime *containerdRuntime
	ctrdImage   containerd.Image
	digest      digest.Digest
	size        int64
}

func getImage(ctx context.Context,
	ctrdRun *containerdRuntime, ctrdImg containerd.Image) (*image, error) {

	imgConf, err := ctrdImg.Config(ctx)
	if err != nil {
		return nil, err
	}

	return &image{
		ctrdRuntime: ctrdRun,
		ctrdImage:   ctrdImg,
		digest:      imgConf.Digest,
		size:        imgConf.Size,
	}, nil
}

func (img *image) Config(ctx context.Context) (*ocispec.ImageConfig, error) {

	ociDesc, err := img.ctrdImage.Config(ctx)
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
		blob, err := content.ReadBlob(ctx, store, ociDesc)
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
	return img.digest
}

func (img *image) RootFS(ctx context.Context) ([]digest.Digest, error) {

	rootFS, err := img.ctrdImage.RootFS(ctx)
	if err != nil {
		return nil, runtime.Errorf("failed to get image rootfs %v", err)
	}

	return rootFS, nil
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
	return img.size
}

func (img *image) Mount(ctx context.Context, path string) error {

	var mounts []mount.Mount
	var err error

	diffIDs, err := img.ctrdImage.RootFS(ctx)
	if err != nil {
		return runtime.Errorf("failed to get rootfs: %v", err)
	}

	digest := identity.ChainID(diffIDs).String()
	snapName := digest + "-image"
	mounts, _, err = createSnapshot(ctx, img.ctrdRuntime, snapName, digest, false)
	if err != nil {
		return err
	}

	err = mount.All(mounts, path)
	if err != nil {
		return err
	}

	return nil
}

func (img *image) Unmount(ctx context.Context, path string) error {

	err := mount.UnmountAll(path, 0)
	if err != nil {
		return err
	}

	diffIDs, err := img.ctrdImage.RootFS(ctx)
	if err != nil {
		return runtime.Errorf("failed to get rootfs: %v", err)
	}

	digest := identity.ChainID(diffIDs).String()
	snapName := digest + "-image"
	return deleteSnapshot(ctx, img.ctrdRuntime, snapName)
}

func (img *image) Unpack(ctx context.Context) error {
	return img.ctrdImage.Unpack(ctx, containerd.DefaultSnapshotter)
}
