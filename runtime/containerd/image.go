//go:build linux

package containerd

import (
	"context"
	"encoding/json"
	"errors"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/content"
	ctrderr "github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/labels"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/reference"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/snapshots"

	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/identity"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/runtime"
)

const updateIntervalMsecs = 100

// updateImageProgress sends the current image download status in a regular 100ms interval
// to the provided progress channel.
// Note that this is the only function closing the progress channel
func updateImageProgress(cctx context.Context, ctx context.Context,
	ctrdRun *containerdRuntime, mutex *sync.Mutex, descs *[]ocispec.Descriptor,
	progress chan<- []runtime.ProgressStatus) {

	var (
		ticker = time.NewTicker(time.Duration(updateIntervalMsecs) * time.Millisecond)
		start  = time.Now()
	)

	defer ticker.Stop()
	cs := ctrdRun.client.ContentStore()
	sn := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)

	for loop := true; loop; {
		var statuses []runtime.ProgressStatus
		var err error

		select {
		case <-ticker.C:
			statuses, err = updateImageStatus(cctx, start, cs, sn, mutex, descs)

		case <-cctx.Done():
			statuses, err = updateImageStatus(ctx, start, cs, sn, mutex, descs)
			loop = false
		}

		if err != nil {
			break
		}

		progress <- statuses
	}
}

// updateImageStatus sends the status of the current image download to the progress channel.
func updateImageStatus(ctx context.Context, start time.Time,
	cs content.Store, sn snapshots.Snapshotter,
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
				Status:    runtime.StatusLoading,
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

	var chain []digest.Digest
	for _, desc := range *descs {
		ref := remotes.MakeRefKey(ctx, desc)
		if _, isActive := actStats[ref]; isActive {
			continue
		}
		stat := runtime.ProgressStatus{
			Reference: ref,
			Status:    runtime.StatusPending,
		}

		info, err := cs.Info(ctx, desc.Digest)
		if err != nil && ctrderr.IsNotFound(err) {
			stat.Status = runtime.StatusPending

		} else if err == nil {
			if strings.Contains(desc.MediaType, "image.layer") {
				if snapID, f := info.Labels[labels.LabelUncompressed]; f {
					chain = append(chain, digest.Digest(snapID))
					chainID := identity.ChainID(chain)
					if _, err := sn.Stat(ctx, chainID.String()); err == nil {
						stat.Status = runtime.StatusComplete
					}
				} else {
					stat.Status = runtime.StatusUnpacking
				}
			} else {
				stat.Status = runtime.StatusComplete
			}
			stat.UpdatedAt = info.CreatedAt
		} else {
			// ignore errors
			continue
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

func newImage(ctx context.Context,
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

func getImage(ctx context.Context, ctrdRun containerdRuntime, name string) (runtime.Image, error) {
	ctrdImg, err := ctrdRun.client.GetImage(ctx, name)
	if errors.Is(err, ctrderr.ErrNotFound) {
		return nil, errdefs.NotFound("image", name)
	} else if err != nil {
		return nil, err
	}

	return newImage(ctx, &ctrdRun, ctrdImg)
}

// TODO: ContainerD is not really stable when interrupting an image pull (e.g. using CTRL-C)
// TODO: Snapshots can stay in extracting stage and never complete.

func pullImage(ctx context.Context, ctrdRun *containerdRuntime, name string,
	progress chan<- []runtime.ProgressStatus) (runtime.Image, error) {

	var mutex sync.Mutex
	descs := []ocispec.Descriptor{}

	h := images.HandlerFunc(func(ctx context.Context,
		desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {

		if desc.MediaType != images.MediaTypeDockerSchema1Manifest {
			mutex.Lock()
			found := false
			for _, d := range descs {
				if desc.Digest == d.Digest {
					found = true
					break
				}
			}
			if !found {
				descs = append(descs, desc)
			}
			mutex.Unlock()
		}
		return nil, nil
	})
	if progress != nil {
		var wg sync.WaitGroup
		pctx, stopProgress := context.WithCancel(ctx)
		defer wg.Wait()
		defer stopProgress()
		wg.Add(1)
		go func() {
			defer wg.Done()
			updateImageProgress(pctx, ctx, ctrdRun, &mutex, &descs, progress)
		}()
	}

	// ignore signals while pulling - see comment above
	signal.Ignore()

	ctrdImg, err := ctrdRun.client.Pull(ctx, name, containerd.WithImageHandler(h))

	signal.Reset()

	if err == reference.ErrObjectRequired {
		return nil, runtime.Errorf("invalid image name '%s': %v", name, err)
	} else if err != nil {
		return nil, runtime.Errorf("pull image '%s' failed: %v", name, err)
	}

	return newImage(ctx, ctrdRun, ctrdImg)
}

// Image interface

func (img *image) Name() string {
	return img.ctrdImage.Name()
}

func (img *image) Size() int64 {
	return img.size
}

func (img *image) Digest() digest.Digest {
	return img.digest
}

func (img *image) CreatedAt() time.Time {
	/* TODO: Image.Metadata()Is supposed to be available in containerd 1.3.3
	   return img.ctrdImage.Metadata().CreatedAt
	*/
	return time.Now()
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

func (img *image) Unpack(ctx context.Context, progress chan<- []runtime.ProgressStatus) error {

	diffIDs, err := img.ctrdImage.RootFS(ctx)
	if err != nil {
		return err
	}

	cs := img.ctrdImage.ContentStore()
	manifest, _ := images.Manifest(ctx, cs, img.ctrdImage.Target(), img.ctrdImage.Platform())
	if len(diffIDs) != len(manifest.Layers) {
		return errors.New("mismatched image rootfs and manifest layers")
	}

	descs := make([]ocispec.Descriptor, len(diffIDs))
	for i := range diffIDs {
		descs[i] = ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageLayer,
			Digest:    manifest.Layers[i].Digest,
		}
	}

	if progress != nil {
		var wg sync.WaitGroup
		cctx, stopProgress := context.WithCancel(ctx)
		defer wg.Wait()
		defer stopProgress()
		wg.Add(1)

		go func() {
			defer wg.Done()
			// TODO: is the mutex still necessary?
			var mutex sync.Mutex
			updateImageProgress(cctx, ctx, img.ctrdRuntime, &mutex, &descs, progress)
		}()
	}
	return img.ctrdImage.Unpack(ctx, containerd.DefaultSnapshotter)
}

func (img *image) RootFS(ctx context.Context) ([]digest.Digest, error) {

	rootFS, err := img.ctrdImage.RootFS(ctx)
	if err != nil {
		return nil, runtime.Errorf("failed to get image rootfs %v", err)
	}

	return rootFS, nil
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
