package containerd

import (
	"context"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/reference"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/runtime"
)

// containerdRuntime provides the runtime implementation for the containerd daemon
// For more information about containerd, see: https://github.com/containerd/containerd
type containerdRuntime struct {
	client  *containerd.Client
	context context.Context
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

	client, err := containerd.New(confRun.SocketName)
	if err != nil {
		return nil, err
	}

	ctrdCtx := namespaces.WithNamespace(context.Background(), confRun.Namespace)

	return &containerdRuntime{
		client:  client,
		context: ctrdCtx,
	}, nil
}

// Close closes the client to containerd
func (ctrdRun *containerdRuntime) Close() {
	ctrdRun.client.Close()
}

// Images returns a list of all images available on the system
func (ctrdRun *containerdRuntime) Images() ([]runtime.Image, error) {

	ctrdImgs, err := ctrdRun.client.ListImages(ctrdRun.context)
	if err != nil {
		return nil, err
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

// PullImage pulls the specified image by name from the default registry
func (ctrdRun *containerdRuntime) PullImage(name string) (runtime.Image, error) {

	ctrdImg, err := ctrdRun.client.Pull(ctrdRun.context, name, containerd.WithPullUnpack)
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
