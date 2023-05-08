// Package remote implements the runtime interface for a remote CNE server
package remote

import (
	"context"
	"fmt"

	"github.com/czankel/cne/config"
	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/runtime"
	"github.com/czankel/cne/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const ContextNamespace = "CNE_NS"

// remoteRuntime provides the runtime implementation for a remote CNE instance
type remoteRuntime struct {
	namespace string
	client    service.RemoteClient
	conn      *grpc.ClientConn
}

type remoteRuntimeType struct {
}

func init() {
	runtime.Register("remote", &remoteRuntimeType{})
}

func (r *remoteRuntimeType) Open(ctx context.Context,
	confRun *config.Runtime) (runtime.Runtime, error) {

	// FIXME: need to pass host information
	conn, err := grpc.Dial("localhost:50000",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := service.NewRemoteClient(conn)

	return &remoteRuntime{
		client: client,
		conn:   conn,
	}, nil
}

func (remRun *remoteRuntime) WithNamespace(ctx context.Context, ns string) context.Context {
	return context.WithValue(ctx, ContextNamespace, ns)
}

func (remRun *remoteRuntime) Close() {
	if remRun.conn != nil {
		remRun.conn.Close()
		remRun.conn = nil
	}
}

func (remRun *remoteRuntime) Images(ctx context.Context) ([]runtime.Image, error) {

	if remRun.conn == nil {
		return nil, errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	pbImgs, err := remRun.client.OnImages(ctx, &service.Namespace{Namespace: ns})
	if err != nil {
		return nil, service.ConvPbErrorToGo(err)
	}

	remImgs := pbImgs.Images
	runImgs := make([]runtime.Image, len(remImgs))
	for i, img := range remImgs {
		runImgs[i] = &image{
			remRuntime: remRun,
			remImage:   img,
		}
	}

	return runImgs, nil
}

func (remRun *remoteRuntime) GetImage(ctx context.Context, name string) (runtime.Image, error) {

	if remRun.conn == nil {
		return nil, errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	pbImg, err := remRun.client.OnGetImage(ctx, &service.Name{Namespace: ns, Value: name})
	if err != nil {
		fmt.Printf("GETIMAGE ERROR %v\n", err)
		return nil, service.ConvPbErrorToGo(err)
	}
	return &image{
		remRuntime: remRun,
		remImage:   pbImg,
	}, nil
}

// TODO: implement progress feedback
func (remRun *remoteRuntime) PullImage(ctx context.Context, name string,
	progress chan<- []runtime.ProgressStatus) (runtime.Image, error) {

	if remRun.conn == nil {
		return nil, errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	pbImg, err := remRun.client.OnPullImage(ctx, &service.Name{Namespace: ns, Value: name})
	if err != nil {
		return nil, service.ConvPbErrorToGo(err)
	}
	return &image{
		remRuntime: remRun,
		remImage:   pbImg,
	}, nil
}

func (remRun *remoteRuntime) DeleteImage(ctx context.Context, name string) error {

	if remRun.conn == nil {
		return errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	_, err := remRun.client.OnDeleteImage(ctx, &service.Name{Namespace: ns, Value: name})
	if err != nil {
		return service.ConvPbErrorToGo(err)
	}

	return nil
}

func (remRun *remoteRuntime) Snapshots(ctx context.Context) ([]runtime.Snapshot, error) {

	if remRun.conn == nil {
		return nil, errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	pbSnaps, err := remRun.client.OnSnapshots(ctx, &service.Namespace{Namespace: ns})
	if err != nil {
		return nil, service.ConvPbErrorToGo(err)
	}

	// FIXME: extract information instead of using pb?
	remSnaps := pbSnaps.Snapshots
	runSnaps := make([]runtime.Snapshot, len(remSnaps))
	for i, s := range pbSnaps.Snapshots {
		runSnaps[i] = &snapshot{
			remSnapshot: s,
		}
	}

	return runSnaps, nil
}

func (remRun *remoteRuntime) GetSnapshot(ctx context.Context, name string) (runtime.Snapshot, error) {

	if remRun.conn == nil {
		return nil, errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	pbSnap, err := remRun.client.OnGetSnapshot(ctx, &service.Name{Namespace: ns, Value: name})
	if err != nil {
		return nil, service.ConvPbErrorToGo(err)
	}
	return &snapshot{
		remSnapshot: pbSnap,
	}, nil
}

func (remRun *remoteRuntime) DeleteSnapshot(ctx context.Context, name string) error {

	if remRun.conn == nil {
		return errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	_, err := remRun.client.OnDeleteSnapshot(ctx, &service.Name{Namespace: ns, Value: name})
	if err != nil {
		return service.ConvPbErrorToGo(err)
	}

	return nil
}

func (remRun *remoteRuntime) Containers(ctx context.Context,
	filters ...interface{}) ([]runtime.Container, error) {
	if remRun.conn == nil {
		return nil, errdefs.NotConnected()
	}

	hasDomain := false
	var domain [16]byte

	if len(filters) > 1 {
		err := errdefs.InvalidArgument("too many arguments to get containers")
		return nil, service.ConvPbErrorToGo(err)
	}
	if len(filters) == 1 {
		domain, hasDomain = filters[0].([16]byte)
		if !hasDomain {
			err := errdefs.InvalidArgument("invalid arguments for getting containers")
			return nil, service.ConvPbErrorToGo(err)
		}
	}

	ns := ctx.Value(ContextNamespace).(string)
	pbFilters := &service.Filter{Namespace: ns, Type: service.Filter_NONE}
	if hasDomain {
		pbFilters = &service.Filter{
			Type:   service.Filter_DOMAIN,
			Domain: domain[:],
		}
	}

	pbCtrs, err := remRun.client.OnContainers(ctx, pbFilters)
	if err != nil {
		return nil, service.ConvPbErrorToGo(err)
	}

	remCtrs := pbCtrs.Containers
	runCtrs := make([]runtime.Container, len(remCtrs))
	for i, c := range remCtrs {
		// FIXME: use helper function?
		runCtrs[i] = &container{
			remRuntime:    remRun,
			remDomain:     [16]byte(c.Id.Domain),
			remID:         [16]byte(c.Id.Id),
			remGeneration: [16]byte(c.Id.Generation),
			remUID:        c.Id.Uid,
			remCreatedAt:  service.ConvPbTimeToGo(c.Created),
			remUpdatedAt:  service.ConvPbTimeToGo(c.Updated),
		}
	}

	return runCtrs, nil
}

func (remRun *remoteRuntime) GetContainer(ctx context.Context,
	domain, id, generation [16]byte) (runtime.Container, error) {

	if remRun.conn == nil {
		return nil, errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	pbCtr := &service.Container{
		Namespace: ns,
		Id: &service.ContainerID{
			Domain:     domain[:],
			Id:         id[:],
			Generation: generation[:],
		},
	}

	pbCtrInfo, err := remRun.client.OnGetContainer(ctx, pbCtr)
	if err != nil {
		return nil, service.ConvPbErrorToGo(err)
	}

	return &container{
		remRuntime:    remRun,
		remDomain:     [16]byte(pbCtrInfo.Id.Domain),
		remID:         [16]byte(pbCtrInfo.Id.Id),
		remGeneration: [16]byte(pbCtrInfo.Id.Generation),
		remUID:        pbCtrInfo.Id.Uid,
		remCreatedAt:  service.ConvPbTimeToGo(pbCtrInfo.Created),
		remUpdatedAt:  service.ConvPbTimeToGo(pbCtrInfo.Updated)}, nil
}

func (remRun *remoteRuntime) NewContainer(
	ctx context.Context,
	domain, id, generation [16]byte, uid uint32,
	img runtime.Image) (runtime.Container, error) {

	// FIXME set created to now ??
	return &container{
		remRuntime:    remRun,
		remDomain:     domain,
		remID:         id,
		remGeneration: generation,
		remUID:        uid,
		image:         img.Name()}, nil // FIXME store digest of image??
}

func (remRun *remoteRuntime) DeleteContainer(ctx context.Context, domain, id, generation [16]byte) error {

	if remRun.conn == nil {
		return errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	pbCtr := &service.Container{
		Namespace: ns,
		Id: &service.ContainerID{
			Domain:     domain[:],
			Id:         id[:],
			Generation: generation[:],
		},
	}

	_, err := remRun.client.OnDeleteContainer(ctx, pbCtr)
	if err != nil {
		return service.ConvPbErrorToGo(err)
	}

	return nil
}

func (remRun *remoteRuntime) PurgeContainer(ctx context.Context, domain, id, generation [16]byte) error {

	if remRun.conn == nil {
		return errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	pbCtr := &service.Container{
		Namespace: ns,
		Id: &service.ContainerID{
			Domain:     domain[:],
			Id:         id[:],
			Generation: generation[:],
		},
	}

	_, err := remRun.client.OnPurgeContainer(ctx, pbCtr)
	if err != nil {
		return service.ConvPbErrorToGo(err)
	}

	return nil
}
