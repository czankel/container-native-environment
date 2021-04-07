// Package containerd implements the runtime interface for the ContainerD Dameon containerd.io
package containerd

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/containerd/containerd"
	ctrderr "github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/reference"
	"github.com/containerd/containerd/snapshots"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	runspecs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/runtime"
)

const containerdGenerationLabel = "CNE-GEN"

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

// getGeneration returns the generation stored in a label
func getGeneration(ctrdRun *containerdRuntime, ctrdCtr containerd.Container) ([16]byte, error) {

	var gen [16]byte

	labels, err := ctrdCtr.Labels(ctrdRun.context)
	if err != nil {
		return [16]byte{}, runtime.Errorf("failed to get generation: %v", err)
	}

	val := labels[containerdGenerationLabel]
	str, err := hex.DecodeString(val)
	if err != nil {
		return [16]byte{}, runtime.Errorf("failed to decode generation '%s': $v", val, err)
	}
	copy(gen[:], str)

	return gen, nil
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

func (ctrdRun *containerdRuntime) Domains() ([][16]byte, error) {

	var domains [][16]byte

	ctrdCtrs, err := ctrdRun.client.Containers(ctrdRun.context)
	if err != nil {
		return domains, err
	}

	for _, c := range ctrdCtrs {
		dom, _, err := splitCtrdID(c.ID())
		if err != nil {
			return domains, err
		}
		found := false
		for _, d := range domains {
			if d == dom {
				found = true
				break
			}
		}
		if !found {
			domains = append(domains, dom)
		}
	}

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	err = snapSvc.Walk(ctrdRun.context, func(ctx context.Context, info snapshots.Info) error {

		name := string(info.Name)
		idx := strings.Index(name, "-")
		if idx == 32 {
			str, err := hex.DecodeString(name[:32])
			if err != nil {
				return runtime.Errorf("failed to decode domain '%s': $v", name, err)
			}

			var dom [16]byte
			copy(dom[:], str)

			found := false
			for _, d := range domains {
				if d == dom {
					found = true
					break
				}
			}
			if !found {
				domains = append(domains, dom)
			}
		}
		return nil
	})

	return domains, nil
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

func (ctrdRun *containerdRuntime) GetImage(name string) (runtime.Image, error) {

	ctrdImg, err := ctrdRun.client.GetImage(ctrdRun.context, name)
	if errors.Is(err, ctrderr.ErrNotFound) {
		return nil, errdefs.NotFound("image", name)
	} else if err != nil {
		return nil, err
	}

	return &image{
		ctrdRuntime: ctrdRun,
		ctrdImage:   ctrdImg,
	}, nil
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

func (ctrdRun *containerdRuntime) Snapshots() ([]runtime.Snapshot, error) {

	var snaps []runtime.Snapshot

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	err := snapSvc.Walk(ctrdRun.context, func(ctx context.Context, info snapshots.Info) error {
		snaps = append(snaps, &snapshot{info: info})
		return nil
	})
	return snaps, err
}

func deleteSnapshot(ctrdRun *containerdRuntime, name string) error {

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	err := snapSvc.Remove(ctrdRun.context, name)
	if err != nil {
		return runtime.Errorf("delete snapshot '%s' failed: %v", name, err)
	}

	return nil
}

func (ctrdRun *containerdRuntime) Containers(domain [16]byte) ([]runtime.Container, error) {

	var runCtrs []runtime.Container

	ctrdCtrs, err := ctrdRun.client.Containers(ctrdRun.context)
	if err != nil {
		return nil, runtime.Errorf("failed to get containers: %v", err)
	}

	for _, c := range ctrdCtrs {

		dom, id, err := splitCtrdID(c.ID())
		if err != nil {
			return nil, err
		}
		if dom != domain {
			continue
		}

		gen, err := getGeneration(ctrdRun, c)
		if err != nil {
			return nil, err
		}

		img, err := c.Image(ctrdRun.context)
		if err != nil {
			return nil, runtime.Errorf("failed to get image: %v", err)
		}
		spec, err := c.Spec(ctrdRun.context)
		if err != nil {
			return nil, runtime.Errorf("failed to get image spec: %v", err)
		}

		runCtrs = append(runCtrs, &container{
			domain:        dom,
			id:            id,
			generation:    gen,
			image:         &image{ctrdRun, img},
			spec:          spec,
			ctrdRuntime:   ctrdRun,
			ctrdContainer: c,
		})
	}
	return runCtrs, nil
}

func (ctrdRun *containerdRuntime) NewContainer(domain [16]byte, id [16]byte, generation [16]byte,
	img runtime.Image, spec *runspecs.Spec) (runtime.Container, error) {

	return &container{
		domain:        domain,
		id:            id,
		generation:    generation,
		image:         img.(*image),
		spec:          spec,
		ctrdRuntime:   ctrdRun,
		ctrdContainer: nil,
	}, nil
}
