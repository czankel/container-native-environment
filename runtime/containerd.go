package runtime

import (
	"context"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
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
func (r *containerdRuntimeType) Open(sockName string) (Runtime, error) {

	c, err := containerd.New(sockName)
	if err != nil {
		return nil, err
	}

	ctx := namespaces.WithNamespace(context.Background(), contextName)

	return &containerdRuntime{
		client:  c,
		context: ctx,
	}, nil
}

// Close closes the client to containerd
func (run *containerdRuntime) Close() {
	run.client.Close()
}
