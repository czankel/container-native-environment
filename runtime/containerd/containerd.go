// Package containerd implements the runtime interface for the ContainerD Dameon containerd.io
package containerd

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/reference"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/runtime"
)

// containerdRuntime provides the runtime implementation for the containerd daemon
// For more information about containerd, see: https://github.com/containerd/containerd
type containerdRuntime struct {
	client    *containerd.Client
	context   context.Context
	namespace string
}

type containerdRuntimeType struct {
}

const contextName = "cne"

func init() {
	runtime.Register("containerd", &containerdRuntimeType{})
}

// Runtime Interface

func (r *containerdRuntimeType) Open(confRun config.Runtime) (runtime.Runtime, error) {

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

	ctrdCtx := namespaces.WithNamespace(context.Background(), confRun.Namespace)

	return &containerdRuntime{
		client:    client,
		context:   ctrdCtx,
		namespace: confRun.Namespace,
	}, nil
}

func (ctrdRun *containerdRuntime) Namespace() string {
	return ctrdRun.namespace
}

func (ctrdRun *containerdRuntime) Close() {
	ctrdRun.client.Close()
}

func (ctrdRun *containerdRuntime) Images() ([]runtime.Image, error) {

	ctrdImgs, err := ctrdRun.client.ListImages(ctrdRun.context)
	if err != nil {
		return nil, runtime.Errorf("ListImages failed: %v", err)
	}

	runImgs := make([]runtime.Image, len(ctrdImgs))
	for i, ctrdImg := range ctrdImgs {
		runImgs[i] = &image{
			ctrdRuntime: ctrdRun,
			ctrdImage:   ctrdImg,
		}
	}

	return runImgs, nil
}

// TODO: ContainerD is not really stable when interrupting an image pull (e.g. using CTRL-C)
// TODO: Snapshots can stay in extracting stage and never complete.

func (ctrdRun *containerdRuntime) PullImage(name string,
	progress chan<- []runtime.ProgressStatus) (runtime.Image, error) {

	var mutex sync.Mutex
	descs := []ocispec.Descriptor{}

	var wg sync.WaitGroup
	wg.Add(1)

	h := images.HandlerFunc(func(ctrdCtx context.Context,
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

	pctx, stopProgress := context.WithCancel(ctrdRun.context)
	if progress != nil {
		go func() {
			defer wg.Done()
			defer close(progress)
			updateImageProgress(ctrdRun, pctx, &mutex, &descs, progress)
		}()
	}

	// ignore signals while pulling - see comment above
	signal.Ignore()

	ctrdImg, err := ctrdRun.client.Pull(ctrdRun.context, name,
		containerd.WithPullUnpack, containerd.WithImageHandler(h))

	signal.Reset()

	if progress != nil {
		stopProgress()
		wg.Wait()
	}

	if err == reference.ErrObjectRequired {
		return nil, runtime.Errorf("invalid image name '%s': %v", name, err)
	} else if err != nil {
		return nil, runtime.Errorf("pull image '%s' failed: %v", name, err)
	}

	return &image{
		ctrdRuntime: ctrdRun,
		ctrdImage:   ctrdImg,
	}, nil
}

func (ctrdRun *containerdRuntime) DeleteImage(name string) error {
	imgSvc := ctrdRun.client.ImageService()

	err := imgSvc.Delete(ctrdRun.context, name, images.SynchronousDelete())
	if err != nil {
		return runtime.Errorf("delete image '%s' failed: %v", name, err)
	}

	return nil
}
