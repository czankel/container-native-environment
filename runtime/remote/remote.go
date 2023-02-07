// Package remote implements the runtime interface for a remote CNE server
package remote

import (
	runspecs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/runtime"
)

// remoteRuntime provides the runtime implementation for a remote CNE instance
type remoteRuntime struct {
	namespace string
}

type remoteRuntimeType struct {
}

func init() {
	runtime.Register("remote", &remoteRuntimeType{})
}

// Runtime Interface

func (r *remoteRuntimeType) Open(confRun config.Runtime) (runtime.Runtime, error) {
	return &remoteRuntime{
		namespace: confRun.Namespace,
	}, nil
}

func (remoteRun *remoteRuntime) Namespace() string {
	return ""
}

func (remoteRun *remoteRuntime) Close() {
}

func (remoteRun *remoteRuntime) Images() ([]runtime.Image, error) {
	return nil, errdefs.NotImplemented()
}

func (remoteRun *remoteRuntime) GetImage(name string) (runtime.Image, error) {
	return nil, errdefs.NotImplemented()
}

func (remoteRun *remoteRuntime) PullImage(name string,
	progress chan<- []runtime.ProgressStatus) (runtime.Image, error) {

	return nil, errdefs.NotImplemented()
}

func (remoteRun *remoteRuntime) DeleteImage(name string) error {
	return errdefs.NotImplemented()
}

func (remoteRun *remoteRuntime) Snapshots() ([]runtime.Snapshot, error) {
	return nil, errdefs.NotImplemented()
}

func (remoteRun *remoteRuntime) DeleteSnapshot(name string) error {
	return errdefs.NotImplemented()
}

func (remoteRun *remoteRuntime) Containers(filters ...interface{}) ([]runtime.Container, error) {
	return nil, errdefs.NotImplemented()
}

func (remoteRun *remoteRuntime) GetContainer(
	domain, id, generation [16]byte) (runtime.Container, error) {

	return nil, errdefs.NotImplemented()
}

func (remoteRun *remoteRuntime) NewContainer(domain, id, generation [16]byte, uid uint32,
	img runtime.Image, spec *runspecs.Spec) (runtime.Container, error) {

	return nil, errdefs.NotImplemented()
}

func (remoteRun *remoteRuntime) DeleteContainer(domain, id, generation [16]byte) error {
	return errdefs.NotImplemented()
}

func (remoteRun *remoteRuntime) PurgeContainer(domain, id, generation [16]byte) error {
	return errdefs.NotImplemented()
}
