package runtime

import (
	"context"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"

	"github.com/czankel/cne/config"
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
	Register("containerd", &containerdRuntimeType{})
}

// Runtime Interface

// Open opens the containerd runtime under the default context name
func (r *containerdRuntimeType) Open(confRun config.Runtime) (Runtime, error) {

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
