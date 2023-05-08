// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.21.12
// source: service/runtime.proto

package service

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Remote_OnImages_FullMethodName            = "/cne.Remote/OnImages"
	Remote_OnGetImage_FullMethodName          = "/cne.Remote/OnGetImage"
	Remote_OnPullImage_FullMethodName         = "/cne.Remote/OnPullImage"
	Remote_OnDeleteImage_FullMethodName       = "/cne.Remote/OnDeleteImage"
	Remote_OnSnapshots_FullMethodName         = "/cne.Remote/OnSnapshots"
	Remote_OnGetSnapshot_FullMethodName       = "/cne.Remote/OnGetSnapshot"
	Remote_OnDeleteSnapshot_FullMethodName    = "/cne.Remote/OnDeleteSnapshot"
	Remote_OnContainers_FullMethodName        = "/cne.Remote/OnContainers"
	Remote_OnGetContainer_FullMethodName      = "/cne.Remote/OnGetContainer"
	Remote_OnCreateContainer_FullMethodName   = "/cne.Remote/OnCreateContainer"
	Remote_OnContainerSnapshot_FullMethodName = "/cne.Remote/OnContainerSnapshot"
	Remote_OnDeleteContainer_FullMethodName   = "/cne.Remote/OnDeleteContainer"
	Remote_OnPurgeContainer_FullMethodName    = "/cne.Remote/OnPurgeContainer"
	Remote_OnSetRootFS_FullMethodName         = "/cne.Remote/OnSetRootFS"
	Remote_OnAmend_FullMethodName             = "/cne.Remote/OnAmend"
	Remote_OnCommit_FullMethodName            = "/cne.Remote/OnCommit"
	Remote_OnExec_FullMethodName              = "/cne.Remote/OnExec"
)

// RemoteClient is the client API for Remote service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RemoteClient interface {
	// image
	OnImages(ctx context.Context, in *Namespace, opts ...grpc.CallOption) (*Images, error)
	OnGetImage(ctx context.Context, in *Name, opts ...grpc.CallOption) (*Image, error)
	OnPullImage(ctx context.Context, in *Name, opts ...grpc.CallOption) (*Image, error)
	OnDeleteImage(ctx context.Context, in *Name, opts ...grpc.CallOption) (*emptypb.Empty, error)
	// snapshot
	OnSnapshots(ctx context.Context, in *Namespace, opts ...grpc.CallOption) (*Snapshots, error)
	OnGetSnapshot(ctx context.Context, in *Name, opts ...grpc.CallOption) (*Snapshot, error)
	OnDeleteSnapshot(ctx context.Context, in *Name, opts ...grpc.CallOption) (*emptypb.Empty, error)
	// container
	OnContainers(ctx context.Context, in *Filter, opts ...grpc.CallOption) (*Containers, error)
	OnGetContainer(ctx context.Context, in *Container, opts ...grpc.CallOption) (*ContainerInfo, error)
	OnCreateContainer(ctx context.Context, in *Container, opts ...grpc.CallOption) (*emptypb.Empty, error)
	OnContainerSnapshot(ctx context.Context, in *Container, opts ...grpc.CallOption) (*Snapshot, error)
	OnDeleteContainer(ctx context.Context, in *Container, opts ...grpc.CallOption) (*emptypb.Empty, error)
	OnPurgeContainer(ctx context.Context, in *Container, opts ...grpc.CallOption) (*emptypb.Empty, error)
	OnSetRootFS(ctx context.Context, in *RootFS, opts ...grpc.CallOption) (*emptypb.Empty, error)
	OnAmend(ctx context.Context, in *Container, opts ...grpc.CallOption) (*Snapshot, error)
	OnCommit(ctx context.Context, in *Container, opts ...grpc.CallOption) (*emptypb.Empty, error)
	OnExec(ctx context.Context, opts ...grpc.CallOption) (Remote_OnExecClient, error)
}

type remoteClient struct {
	cc grpc.ClientConnInterface
}

func NewRemoteClient(cc grpc.ClientConnInterface) RemoteClient {
	return &remoteClient{cc}
}

func (c *remoteClient) OnImages(ctx context.Context, in *Namespace, opts ...grpc.CallOption) (*Images, error) {
	out := new(Images)
	err := c.cc.Invoke(ctx, Remote_OnImages_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnGetImage(ctx context.Context, in *Name, opts ...grpc.CallOption) (*Image, error) {
	out := new(Image)
	err := c.cc.Invoke(ctx, Remote_OnGetImage_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnPullImage(ctx context.Context, in *Name, opts ...grpc.CallOption) (*Image, error) {
	out := new(Image)
	err := c.cc.Invoke(ctx, Remote_OnPullImage_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnDeleteImage(ctx context.Context, in *Name, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Remote_OnDeleteImage_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnSnapshots(ctx context.Context, in *Namespace, opts ...grpc.CallOption) (*Snapshots, error) {
	out := new(Snapshots)
	err := c.cc.Invoke(ctx, Remote_OnSnapshots_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnGetSnapshot(ctx context.Context, in *Name, opts ...grpc.CallOption) (*Snapshot, error) {
	out := new(Snapshot)
	err := c.cc.Invoke(ctx, Remote_OnGetSnapshot_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnDeleteSnapshot(ctx context.Context, in *Name, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Remote_OnDeleteSnapshot_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnContainers(ctx context.Context, in *Filter, opts ...grpc.CallOption) (*Containers, error) {
	out := new(Containers)
	err := c.cc.Invoke(ctx, Remote_OnContainers_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnGetContainer(ctx context.Context, in *Container, opts ...grpc.CallOption) (*ContainerInfo, error) {
	out := new(ContainerInfo)
	err := c.cc.Invoke(ctx, Remote_OnGetContainer_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnCreateContainer(ctx context.Context, in *Container, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Remote_OnCreateContainer_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnContainerSnapshot(ctx context.Context, in *Container, opts ...grpc.CallOption) (*Snapshot, error) {
	out := new(Snapshot)
	err := c.cc.Invoke(ctx, Remote_OnContainerSnapshot_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnDeleteContainer(ctx context.Context, in *Container, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Remote_OnDeleteContainer_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnPurgeContainer(ctx context.Context, in *Container, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Remote_OnPurgeContainer_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnSetRootFS(ctx context.Context, in *RootFS, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Remote_OnSetRootFS_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnAmend(ctx context.Context, in *Container, opts ...grpc.CallOption) (*Snapshot, error) {
	out := new(Snapshot)
	err := c.cc.Invoke(ctx, Remote_OnAmend_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnCommit(ctx context.Context, in *Container, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Remote_OnCommit_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteClient) OnExec(ctx context.Context, opts ...grpc.CallOption) (Remote_OnExecClient, error) {
	stream, err := c.cc.NewStream(ctx, &Remote_ServiceDesc.Streams[0], Remote_OnExec_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &remoteOnExecClient{stream}
	return x, nil
}

type Remote_OnExecClient interface {
	Send(*Session) error
	Recv() (*Session, error)
	grpc.ClientStream
}

type remoteOnExecClient struct {
	grpc.ClientStream
}

func (x *remoteOnExecClient) Send(m *Session) error {
	return x.ClientStream.SendMsg(m)
}

func (x *remoteOnExecClient) Recv() (*Session, error) {
	m := new(Session)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// RemoteServer is the server API for Remote service.
// All implementations must embed UnimplementedRemoteServer
// for forward compatibility
type RemoteServer interface {
	// image
	OnImages(context.Context, *Namespace) (*Images, error)
	OnGetImage(context.Context, *Name) (*Image, error)
	OnPullImage(context.Context, *Name) (*Image, error)
	OnDeleteImage(context.Context, *Name) (*emptypb.Empty, error)
	// snapshot
	OnSnapshots(context.Context, *Namespace) (*Snapshots, error)
	OnGetSnapshot(context.Context, *Name) (*Snapshot, error)
	OnDeleteSnapshot(context.Context, *Name) (*emptypb.Empty, error)
	// container
	OnContainers(context.Context, *Filter) (*Containers, error)
	OnGetContainer(context.Context, *Container) (*ContainerInfo, error)
	OnCreateContainer(context.Context, *Container) (*emptypb.Empty, error)
	OnContainerSnapshot(context.Context, *Container) (*Snapshot, error)
	OnDeleteContainer(context.Context, *Container) (*emptypb.Empty, error)
	OnPurgeContainer(context.Context, *Container) (*emptypb.Empty, error)
	OnSetRootFS(context.Context, *RootFS) (*emptypb.Empty, error)
	OnAmend(context.Context, *Container) (*Snapshot, error)
	OnCommit(context.Context, *Container) (*emptypb.Empty, error)
	OnExec(Remote_OnExecServer) error
	mustEmbedUnimplementedRemoteServer()
}

// UnimplementedRemoteServer must be embedded to have forward compatible implementations.
type UnimplementedRemoteServer struct {
}

func (UnimplementedRemoteServer) OnImages(context.Context, *Namespace) (*Images, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnImages not implemented")
}
func (UnimplementedRemoteServer) OnGetImage(context.Context, *Name) (*Image, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnGetImage not implemented")
}
func (UnimplementedRemoteServer) OnPullImage(context.Context, *Name) (*Image, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnPullImage not implemented")
}
func (UnimplementedRemoteServer) OnDeleteImage(context.Context, *Name) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnDeleteImage not implemented")
}
func (UnimplementedRemoteServer) OnSnapshots(context.Context, *Namespace) (*Snapshots, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnSnapshots not implemented")
}
func (UnimplementedRemoteServer) OnGetSnapshot(context.Context, *Name) (*Snapshot, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnGetSnapshot not implemented")
}
func (UnimplementedRemoteServer) OnDeleteSnapshot(context.Context, *Name) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnDeleteSnapshot not implemented")
}
func (UnimplementedRemoteServer) OnContainers(context.Context, *Filter) (*Containers, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnContainers not implemented")
}
func (UnimplementedRemoteServer) OnGetContainer(context.Context, *Container) (*ContainerInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnGetContainer not implemented")
}
func (UnimplementedRemoteServer) OnCreateContainer(context.Context, *Container) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnCreateContainer not implemented")
}
func (UnimplementedRemoteServer) OnContainerSnapshot(context.Context, *Container) (*Snapshot, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnContainerSnapshot not implemented")
}
func (UnimplementedRemoteServer) OnDeleteContainer(context.Context, *Container) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnDeleteContainer not implemented")
}
func (UnimplementedRemoteServer) OnPurgeContainer(context.Context, *Container) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnPurgeContainer not implemented")
}
func (UnimplementedRemoteServer) OnSetRootFS(context.Context, *RootFS) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnSetRootFS not implemented")
}
func (UnimplementedRemoteServer) OnAmend(context.Context, *Container) (*Snapshot, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnAmend not implemented")
}
func (UnimplementedRemoteServer) OnCommit(context.Context, *Container) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnCommit not implemented")
}
func (UnimplementedRemoteServer) OnExec(Remote_OnExecServer) error {
	return status.Errorf(codes.Unimplemented, "method OnExec not implemented")
}
func (UnimplementedRemoteServer) mustEmbedUnimplementedRemoteServer() {}

// UnsafeRemoteServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RemoteServer will
// result in compilation errors.
type UnsafeRemoteServer interface {
	mustEmbedUnimplementedRemoteServer()
}

func RegisterRemoteServer(s grpc.ServiceRegistrar, srv RemoteServer) {
	s.RegisterService(&Remote_ServiceDesc, srv)
}

func _Remote_OnImages_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Namespace)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnImages(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnImages_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnImages(ctx, req.(*Namespace))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnGetImage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Name)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnGetImage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnGetImage_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnGetImage(ctx, req.(*Name))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnPullImage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Name)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnPullImage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnPullImage_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnPullImage(ctx, req.(*Name))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnDeleteImage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Name)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnDeleteImage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnDeleteImage_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnDeleteImage(ctx, req.(*Name))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnSnapshots_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Namespace)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnSnapshots(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnSnapshots_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnSnapshots(ctx, req.(*Namespace))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnGetSnapshot_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Name)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnGetSnapshot(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnGetSnapshot_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnGetSnapshot(ctx, req.(*Name))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnDeleteSnapshot_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Name)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnDeleteSnapshot(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnDeleteSnapshot_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnDeleteSnapshot(ctx, req.(*Name))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnContainers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Filter)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnContainers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnContainers_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnContainers(ctx, req.(*Filter))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnGetContainer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Container)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnGetContainer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnGetContainer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnGetContainer(ctx, req.(*Container))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnCreateContainer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Container)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnCreateContainer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnCreateContainer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnCreateContainer(ctx, req.(*Container))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnContainerSnapshot_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Container)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnContainerSnapshot(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnContainerSnapshot_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnContainerSnapshot(ctx, req.(*Container))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnDeleteContainer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Container)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnDeleteContainer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnDeleteContainer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnDeleteContainer(ctx, req.(*Container))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnPurgeContainer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Container)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnPurgeContainer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnPurgeContainer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnPurgeContainer(ctx, req.(*Container))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnSetRootFS_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RootFS)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnSetRootFS(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnSetRootFS_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnSetRootFS(ctx, req.(*RootFS))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnAmend_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Container)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnAmend(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnAmend_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnAmend(ctx, req.(*Container))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnCommit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Container)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteServer).OnCommit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Remote_OnCommit_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteServer).OnCommit(ctx, req.(*Container))
	}
	return interceptor(ctx, in, info, handler)
}

func _Remote_OnExec_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(RemoteServer).OnExec(&remoteOnExecServer{stream})
}

type Remote_OnExecServer interface {
	Send(*Session) error
	Recv() (*Session, error)
	grpc.ServerStream
}

type remoteOnExecServer struct {
	grpc.ServerStream
}

func (x *remoteOnExecServer) Send(m *Session) error {
	return x.ServerStream.SendMsg(m)
}

func (x *remoteOnExecServer) Recv() (*Session, error) {
	m := new(Session)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Remote_ServiceDesc is the grpc.ServiceDesc for Remote service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Remote_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cne.Remote",
	HandlerType: (*RemoteServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "OnImages",
			Handler:    _Remote_OnImages_Handler,
		},
		{
			MethodName: "OnGetImage",
			Handler:    _Remote_OnGetImage_Handler,
		},
		{
			MethodName: "OnPullImage",
			Handler:    _Remote_OnPullImage_Handler,
		},
		{
			MethodName: "OnDeleteImage",
			Handler:    _Remote_OnDeleteImage_Handler,
		},
		{
			MethodName: "OnSnapshots",
			Handler:    _Remote_OnSnapshots_Handler,
		},
		{
			MethodName: "OnGetSnapshot",
			Handler:    _Remote_OnGetSnapshot_Handler,
		},
		{
			MethodName: "OnDeleteSnapshot",
			Handler:    _Remote_OnDeleteSnapshot_Handler,
		},
		{
			MethodName: "OnContainers",
			Handler:    _Remote_OnContainers_Handler,
		},
		{
			MethodName: "OnGetContainer",
			Handler:    _Remote_OnGetContainer_Handler,
		},
		{
			MethodName: "OnCreateContainer",
			Handler:    _Remote_OnCreateContainer_Handler,
		},
		{
			MethodName: "OnContainerSnapshot",
			Handler:    _Remote_OnContainerSnapshot_Handler,
		},
		{
			MethodName: "OnDeleteContainer",
			Handler:    _Remote_OnDeleteContainer_Handler,
		},
		{
			MethodName: "OnPurgeContainer",
			Handler:    _Remote_OnPurgeContainer_Handler,
		},
		{
			MethodName: "OnSetRootFS",
			Handler:    _Remote_OnSetRootFS_Handler,
		},
		{
			MethodName: "OnAmend",
			Handler:    _Remote_OnAmend_Handler,
		},
		{
			MethodName: "OnCommit",
			Handler:    _Remote_OnCommit_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "OnExec",
			Handler:       _Remote_OnExec_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "service/runtime.proto",
}
