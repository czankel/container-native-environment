// Package containerd implements the runtime interface for the ContainerD Dameon containerd.io
package containerd

import (
	"context"
	"os"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/runtime"
)

const containerdGenerationLabel = "CNE-GEN"
const containerdUIDLabel = "CNE-UID"

// containerdRuntime provides the runtime implementation for the containerd daemon
// For more information about containerd, see: https://github.com/containerd/containerd
type containerdRuntime struct {
	client *containerd.Client
}

type containerdRuntimeType struct {
}

const contextName = "cne"

func init() {
	runtime.Register("containerd", &containerdRuntimeType{})
}

func (r *containerdRuntimeType) Open(ctx context.Context,
	confRun *config.Runtime) (runtime.Runtime, error) {

	// Validate the provided port
	_, err := os.Stat(confRun.SocketName)
	if err != nil {
		return nil, runtime.Errorf("failed to open runtime socket '%s': %v",
			confRun.SocketName, err)
	}

	client, err := containerd.New(confRun.SocketName)
	if err != nil {
		return nil, runtime.Errorf("failed to open runtime socket '%s': %v",
			confRun.SocketName, err)
	}

	return &containerdRuntime{
		client: client,
	}, nil
}

// Runtime Interface

func (ctrdRun *containerdRuntime) WithNamespace(ctx context.Context, ns string) context.Context {
	return namespaces.WithNamespace(ctx, ns)
}

func (ctrdRun *containerdRuntime) Close() {
	ctrdRun.client.Close()
}

func (ctrdRun *containerdRuntime) Images(ctx context.Context) ([]runtime.Image, error) {

	ctrdImgs, err := ctrdRun.client.ListImages(ctx)
	if err != nil {
		return nil, runtime.Errorf("ListImages failed: %v", err)
	}

	runImgs := make([]runtime.Image, len(ctrdImgs))
	for i, ctrdImg := range ctrdImgs {
		runImg, err := newImage(ctx, ctrdRun, ctrdImg)
		if err != nil {
			return nil, err
		}
		runImgs[i] = runImg
	}

	return runImgs, nil
}

func (ctrdRun *containerdRuntime) GetImage(ctx context.Context, name string) (runtime.Image, error) {
	return getImage(ctx, *ctrdRun, name)
}

func (ctrdRun *containerdRuntime) PullImage(ctx context.Context, name string,
	progress chan<- []runtime.ProgressStatus) (runtime.Image, error) {
	return pullImage(ctx, ctrdRun, name, progress)
}

func (ctrdRun *containerdRuntime) DeleteImage(ctx context.Context, name string) error {
	imgSvc := ctrdRun.client.ImageService()

	err := imgSvc.Delete(ctx, name, images.SynchronousDelete())
	if err != nil {
		return runtime.Errorf("delete image '%s' failed: %v", name, err)
	}

	return nil

}

func (ctrdRun *containerdRuntime) Snapshots(ctx context.Context) ([]runtime.Snapshot, error) {
	return getSnapshots(ctx, ctrdRun)
}

func (ctrdRun *containerdRuntime) GetSnapshot(
	ctx context.Context, name string) (runtime.Snapshot, error) {
	return getSnapshot(ctx, ctrdRun, name)
}

func (ctrdRun *containerdRuntime) DeleteSnapshot(ctx context.Context,
	name string) error {

	return deleteSnapshot(ctx, ctrdRun, name)
}

func (ctrdRun *containerdRuntime) Containers(ctx context.Context,
	filters ...interface{}) ([]runtime.Container, error) {
	return getContainers(ctx, ctrdRun, filters...)
}

func (ctrdRun *containerdRuntime) GetContainer(ctx context.Context,
	domain, id, generation [16]byte) (runtime.Container, error) {
	return getContainer(ctx, ctrdRun, domain, id, generation)
}

func (ctrdRun *containerdRuntime) NewContainer(ctx context.Context,
	domain, id, generation [16]byte, uid uint32,
	img runtime.Image) (runtime.Container, error) {

	// start with a base container
	spec, err := runtime.DefaultSpec(ctx)
	if err != nil {
		return nil, err
	}

	return newContainer(ctrdRun, nil, domain, id, generation, uid, img.(*image), &spec), nil
}

func (ctrdRun *containerdRuntime) DeleteContainer(ctx context.Context,
	domain, id, generation [16]byte) error {

	return deleteContainer(ctx, ctrdRun, domain, id, false /*purge*/)
}

func (ctrdRun *containerdRuntime) PurgeContainer(ctx context.Context,
	domain, id, generation [16]byte) error {

	return deleteContainer(ctx, ctrdRun, domain, id, true /*purge*/)
}
