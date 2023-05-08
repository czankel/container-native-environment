// Remote provides a remote runtime service
package service

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/runtime"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	empty "google.golang.org/protobuf/types/known/emptypb"
	timestamp "google.golang.org/protobuf/types/known/timestamppb"
)

type server struct {
	UnimplementedRemoteServer
	run runtime.Runtime
}

// ConvGoTimeToPb is a helper function to convert go's time.Time to protocol
// buffer's timestamp.Timestamp.
func ConvGoTimeToPb(ts time.Time) *timestamp.Timestamp {
	// FIXME: check if valid
	return &timestamp.Timestamp{
		Seconds: ts.Unix(),
		Nanos:   int32(ts.UnixNano() % 1000000000),
	}
}

// ConvGoTimeToPb is a helper function to convert protocol buffer's
// timestamp.Timestamp to go's time.Time.
func ConvPbTimeToGo(ts *timestamp.Timestamp) time.Time {
	// FIXME: check if valid
	return time.Unix(ts.Seconds, ts.Seconds*1000000000+int64(ts.Nanos))
}

// FIXME: rename all convert functions to align
func ConvGoErrorToPb(err error) error {

	// FIXME: keep, convert to ignore , etc.
	if err == nil {
		panic("nil error client")
	}

	if errdefs.IsCneError(err) {
		st := status.New(codes.Unknown, err.Error())
		ds, e := st.WithDetails(
			// FIXME: unwrap errors
			&CneError{
				Cause:    errdefs.Cause(err),
				Resource: errdefs.Resource(err),
				Name:     errdefs.Name(err),
				Msg:      errdefs.Message(err),
			})
		if e != nil {
			err = st.Err()
		} else {
			err = ds.Err()
		}
	}
	return err
}

func ConvPbErrorToGo(err error) error {

	s := status.Convert(err)

	// FIXME: remove?
	if err == nil {
		panic("nil error")
	}

	fmt.Printf("CONVERT ERROR len %d %v\n", len(s.Details()), s)
	if len(s.Details()) > 0 {
		// FIXME range
		cneErr := s.Details()[0].(*CneError)
		switch cneErr.Cause {
		case errdefs.ErrInvalidArgument.Error():
			err = errdefs.InvalidArgument(cneErr.Msg)
		case errdefs.ErrAlreadyExists.Error():
			err = errdefs.AlreadyExists(cneErr.Resource, cneErr.Name)
		case errdefs.ErrNotFound.Error():
			err = errdefs.NotFound(cneErr.Resource, cneErr.Name)
			/* FIXME
			case errdefs.ErrRemoteError.Error():
				err = errdefs.ErrRemoteError
			*/
		case errdefs.ErrNotImplemented.Error():
			err = errdefs.NotImplemented() // FIXME instantiate with info
		case errdefs.ErrInternalError.Error():
			err = errdefs.InternalError(cneErr.Msg)
		case errdefs.ErrInUse.Error():
			err = errdefs.InUse(cneErr.Resource, cneErr.Name)
		case errdefs.ErrNotConnected.Error():
			err = errdefs.NotConnected()
			/*
				case errdefs.ErrCommandFailed.Error():
					err = errdefs.ErrCommandFailed
				case errdefs.ErrCommandNotFound.Error():
					err = errdefs.ErrCommandNotFound
			*/
		}
	} else {
		panic("ERROR LEN")
	}
	fmt.Printf("Return ConvertError %v\n", err)
	return err
}

// pbSnapshot is a helper function to convert a remote snapshot to a protobuf snapshot
func pbSnapshot(runSnap runtime.Snapshot) *Snapshot {
	return &Snapshot{
		Name:    runSnap.Name(),
		Parent:  runSnap.Parent(),
		Created: ConvGoTimeToPb(runSnap.CreatedAt()),
		Size:    runSnap.Size(),
		Inodes:  runSnap.Inodes(),
	}
}

func pbImage(ctx context.Context, runImg runtime.Image) (*Image, error) {

	rootDigests, err := runImg.RootFS(ctx)
	if err != nil {
		return nil, err
	}

	rootfs := make([]string, len(rootDigests))
	for i, r := range rootDigests {
		rootfs[i] = r.String()
	}

	return &Image{
		Name:    runImg.Name(),
		Digest:  runImg.Digest().String(),
		Size:    runImg.Size(),
		Created: ConvGoTimeToPb(runImg.CreatedAt()),
		Rootfs:  rootfs}, nil

}

// Listen listens on the port
func Listen(run runtime.Runtime) error {

	listener, err := net.Listen("tcp", "localhost:50000")
	if err != nil {
		return err
	}

	srv := grpc.NewServer()
	RegisterRemoteServer(srv, &server{run: run})

	if err := srv.Serve(listener); err != nil {
		return err
	}
	return nil
}

//
// Image
//

func (s *server) OnImages(ctx context.Context, namespace *Namespace) (*Images, error) {

	ctx = s.run.WithNamespace(ctx, namespace.Namespace)
	runImgs, err := s.run.Images(ctx)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	var pbImgs = make([]Image, len(runImgs))
	var pbPtrs = make([]*Image, len(runImgs))

	for i, img := range runImgs {
		pbImgs[i].Name = img.Name()
		pbImgs[i].Digest = img.Digest().String()
		pbImgs[i].Size = img.Size()
		digests, err := img.RootFS(ctx)
		if err == nil {
			rootfs := make([]string, len(digests))
			for i, d := range digests {
				rootfs[i] = d.String()
			}
			pbImgs[i].Rootfs = rootfs
		}
		pbImgs[i].Created = ConvGoTimeToPb(img.CreatedAt())
		pbPtrs[i] = &pbImgs[i]
	}

	return &Images{Images: pbPtrs}, nil
}

func (s *server) OnGetImage(ctx context.Context, name *Name) (*Image, error) {

	ctx = s.run.WithNamespace(ctx, name.Namespace)
	runImg, err := s.run.GetImage(ctx, name.Value)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	pbImg, err := pbImage(ctx, runImg)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}
	return pbImg, nil
}

// FIXME: Implemnet progress in PullImage
func (s *server) OnPullImage(ctx context.Context, name *Name) (*Image, error) {

	ctx = s.run.WithNamespace(ctx, name.Namespace)
	runImg, err := s.run.PullImage(ctx, name.Value, nil)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	pbImg, err := pbImage(ctx, runImg)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}
	return pbImg, nil
}

//
// Snapshots
//

func (s *server) OnSnapshots(ctx context.Context, namespace *Namespace) (*Snapshots, error) {

	ctx = s.run.WithNamespace(ctx, namespace.Namespace)
	runSnaps, err := s.run.Snapshots(ctx)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	var pbSnap = make([]Snapshot, len(runSnaps))
	var pbPtrs = make([]*Snapshot, len(runSnaps))

	for i, snap := range runSnaps {
		pbSnap[i].Name = snap.Name()
		pbSnap[i].Parent = snap.Parent()
		pbSnap[i].Created = ConvGoTimeToPb(snap.CreatedAt())
		pbSnap[i].Size = snap.Size()
		pbSnap[i].Inodes = snap.Inodes()
		pbPtrs[i] = &pbSnap[i]
	}

	return &Snapshots{Snapshots: pbPtrs}, nil
}

func (s *server) OnGetSnapshot(ctx context.Context, name *Name) (*Snapshot, error) {

	ctx = s.run.WithNamespace(ctx, name.Namespace)
	runSnap, err := s.run.GetSnapshot(ctx, name.Value)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	return &Snapshot{
		Name:    name.Value,
		Parent:  runSnap.Parent(),
		Created: ConvGoTimeToPb(runSnap.CreatedAt()),
		Size:    runSnap.Size(),
		Inodes:  runSnap.Inodes(),
	}, nil
}

func (s *server) OnDeleteSnapshot(ctx context.Context, name *Name) (*empty.Empty, error) {

	ctx = s.run.WithNamespace(ctx, name.Namespace)
	err := s.run.DeleteSnapshot(ctx, name.Value)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	return &empty.Empty{}, nil
}

//
// Container
//

func (s *server) OnContainers(ctx context.Context, pbFilter *Filter) (*Containers, error) {

	// TODO: currently ignoring other filters
	var filters []interface{}
	if pbFilter.Type == Filter_DOMAIN {
		filters = append(filters, pbFilter.Domain)
	}
	ctx = s.run.WithNamespace(ctx, pbFilter.Namespace)
	runCtrs, err := s.run.Containers(ctx, filters...)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	var pbCtrs = make([]ContainerInfo, len(runCtrs))
	var pbrPtrs = make([]*ContainerInfo, len(runCtrs))

	for i, c := range runCtrs {
		domain := c.Domain()
		id := c.ID()
		generation := c.Generation()
		pbCtrs[i] = ContainerInfo{
			Id: &ContainerID{
				Domain:     domain[:],
				Id:         id[:],
				Generation: generation[:],
				Uid:        c.UID(),
			},
			Created: ConvGoTimeToPb(c.CreatedAt()),
			Updated: ConvGoTimeToPb(c.UpdatedAt()),
		}

		pbrPtrs[i] = &pbCtrs[i]
	}

	return &Containers{Containers: pbrPtrs}, nil
}

func (s *server) OnGetContainer(ctx context.Context, pbCtr *Container) (*ContainerInfo, error) {

	ctx = s.run.WithNamespace(ctx, pbCtr.Namespace)
	runCtr, err := s.run.GetContainer(
		ctx,
		[16]byte(pbCtr.Id.Domain),
		[16]byte(pbCtr.Id.Id),
		[16]byte(pbCtr.Id.Generation))
	if err != nil {
		fmt.Printf("GetContainer failed %v\n", err)
		return nil, ConvGoErrorToPb(err)
	}

	return &ContainerInfo{
		Id:      pbCtr.Id,
		Created: ConvGoTimeToPb(runCtr.CreatedAt()),
		Updated: ConvGoTimeToPb(runCtr.UpdatedAt()),
	}, nil
}

func (s *server) OnDeleteContainer(ctx context.Context, pbCtr *Container) (*empty.Empty, error) {

	ctx = s.run.WithNamespace(ctx, pbCtr.Namespace)
	err := s.run.DeleteContainer(
		ctx,
		[16]byte(pbCtr.Id.Domain),
		[16]byte(pbCtr.Id.Id),
		[16]byte(pbCtr.Id.Generation))
	if err != nil {
		fmt.Printf("XX OnDeleteContainer convert %v\n", err)
		return nil, ConvGoErrorToPb(err)
	}

	fmt.Printf("OnDeletContainer return nil\n")
	return &empty.Empty{}, nil
}

func (s *server) OnPurgeContainer(ctx context.Context, pbCtr *Container) (*empty.Empty, error) {

	ctx = s.run.WithNamespace(ctx, pbCtr.Namespace)
	err := s.run.PurgeContainer(
		ctx,
		[16]byte(pbCtr.Id.Domain),
		[16]byte(pbCtr.Id.Id),
		[16]byte(pbCtr.Id.Generation))
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	return &empty.Empty{}, nil
}

func (s *server) OnSetRootFS(ctx context.Context, rootfs *RootFS) (*empty.Empty, error) {

	ctx = s.run.WithNamespace(ctx, rootfs.Namespace)
	pbCtr := rootfs.Container
	runCtr, err := s.run.GetContainer(
		ctx,
		[16]byte(pbCtr.Domain),
		[16]byte(pbCtr.Id),
		[16]byte(pbCtr.Generation))
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	var runSnap runtime.Snapshot = nil
	if rootfs.Snapshot != "" {
		runSnap, err = s.run.GetSnapshot(ctx, rootfs.Snapshot)
		if err != nil {
			return nil, ConvGoErrorToPb(err)
		}
	}

	err = runCtr.SetRootFS(ctx, runSnap)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	return &empty.Empty{}, nil
}

// FIXME: Have NewContainer above with image, this doesn't
func (s *server) OnCreateContainer(ctx context.Context, pbCtr *Container) (*empty.Empty, error) {

	ctx = s.run.WithNamespace(ctx, pbCtr.Namespace)

	runImg, err := s.run.GetImage(ctx, pbCtr.Image)
	if err != nil {
		fmt.Printf("XX GetImage rror %v\n", err)
		return nil, ConvGoErrorToPb(err)
	}

	runCtr, err := s.run.NewContainer(
		ctx,
		[16]byte(pbCtr.Id.Domain),
		[16]byte(pbCtr.Id.Id),
		[16]byte(pbCtr.Id.Generation),
		pbCtr.Id.Uid,
		runImg)
	if err != nil {
		fmt.Printf("XX Newcontainer failed %v\n", err)
		return nil, ConvGoErrorToPb(err)
	}

	err = runCtr.Create(ctx)
	if err != nil {
		fmt.Printf("XX Create failed %v\n", err)
		return nil, ConvGoErrorToPb(err)
	}

	return &empty.Empty{}, nil
}

func (s *server) OnContainerSnapshot(ctx context.Context, pbCtr *Container) (*Snapshot, error) {

	ctx = s.run.WithNamespace(ctx, pbCtr.Namespace)

	runCtr, err := s.run.GetContainer(
		ctx,
		[16]byte(pbCtr.Id.Domain),
		[16]byte(pbCtr.Id.Id),
		[16]byte(pbCtr.Id.Generation))
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	runSnap, err := runCtr.Snapshot(ctx)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	return pbSnapshot(runSnap), nil
}

func (s *server) OnAmend(ctx context.Context, pbCtr *Container) (*Snapshot, error) {

	ctx = s.run.WithNamespace(ctx, pbCtr.Namespace)
	runCtr, err := s.run.GetContainer(
		ctx,
		[16]byte(pbCtr.Id.Domain),
		[16]byte(pbCtr.Id.Id),
		[16]byte(pbCtr.Id.Generation))
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	runSnap, err := runCtr.Amend(ctx)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	return pbSnapshot(runSnap), nil
}

func (s *server) OnCommit(ctx context.Context, pbCtr *Container) (*empty.Empty, error) {

	ctx = s.run.WithNamespace(ctx, pbCtr.Namespace)
	gen := [16]byte(pbCtr.Id.Generation)
	runCtr, err := s.run.GetContainer(
		ctx,
		[16]byte(pbCtr.Id.Domain),
		[16]byte(pbCtr.Id.Id),
		gen)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	err = runCtr.Commit(ctx, gen)
	if err != nil {
		return nil, ConvGoErrorToPb(err)
	}

	return &empty.Empty{}, nil
}

func (s *server) OnExec(srv Remote_OnExecServer) error {
	/*
		ctx := srv.Context()

		// FIXME: create timeout ctx? if <-ctx.Timeout??
		msg, err := getMessage(s)
		if err != nil {
			return ConvGoErrorToPb(err) // FIXME?
		}
		if msg == nil {
			return nil
		}

		if msg.Command != Session_EXEC {
			return ConvGoErrorToPb(errdefs.InvalidArgument())
		}

		procSpec := &runtime.ProcessSpec{
			UID:  msg.UID,
			GID:  msg.GID,
			Args: msg.Args,
			// FIXME Envs: msg.Envs,
		}
			stream := &runtime.Stream{
				Stdin:    reader,
				Stdout:   writer,
				Stderr:   writererr,
				Terminal: true, // FIXME
			}
			proc, err = ctr.Exec(ctx, stream, procSpec)

			go func() {
				streamLoop(ctx)
			}()
	*/
	return nil
}

/* ORIGINAL

func (s *server) OnExec(srv Remote_OnExecServer) error {


	var proc runtime.Process

	// FIXME go func(reader) {} (reader)?

	for {
		// exit if context is done
		// or continue
		select {
		case <-ctx.Done():
			fmt.Printf("XX OnExec Done\n")
			return ConvGoErrorToPb(ctx.Err())
		default:
		}

		// receive data from stream
		msg, err := srv.Recv()
		fmt.Printf("XX OnExec RECV %v\n", msg)
		if err == io.EOF {
			// return will close stream from server side
			fmt.Println("XX OnExec EOF")
			return nil
		}
		if err != nil {
			fmt.Printf("XX OnExec receive error %v", err)
			continue
		}

		//runCtx = s.run.WithNamespace(ctx, msg.Namespace)

}

	return nil
}
*/

/*
// FIXME: would be same as exec?? except Session_EXEC i
func (s *server) OnAttach(srv Remote_OnAttachServer) error {

	loop()
}
*/

//
// Streaming
//

type remoteReader struct {
}

type remoteWriter struct {
}

type remoteWriterErr struct {
}

/*
func (r *remoteReader) Read(p []byte) (int, error) {
	fmt.Printf("READ\n")
	return 0, nil
}

func (w *remoteWriter) write(p []byte, stderr bool) (int, error) {
	fmt.Printf("WRITE\n")
	return 0, nil
	//return write(p, false)
	// FIXME: how will this work?
	// copy to internal rr-buffer (allow empty line to "flush")
	// if connected, send all buffer lines
}

func (w *remoteWriterErr) Write(p []byte) (int, error) {
	return write(p, true)
	return 0, nil
}
*/

// FIXME: why return error just a simple looper?
func streamLoop(ctx context.Context) {
	/*
		for {
			// exit if context is done
			// or continue
			select {
			case <-ctx.Done():
				fmt.Printf("XX OnExec Done\n")
				return
			default:
			}

			msg, err := getMessage()
			if err != nil {
				cancel(err)
				return
			}
			if msg == nil {
				cancel(nil)
				return
			}

			var resp *Session = nil
			switch msg.Command {
			case Session_SIGINT:
				fmt.Printf("XX OnExec SIGINT\n")
				// FIXME: proc.Signal(
			case Session_SIGKILL:
				// FIXME: proc.SignalKlll(
			case Session_STDIN:
				// FIXME how will this work?
				// copy to queue and wake reader?
				text := "> " + string(msg.Data)
				resp = &Session{Data: []byte(text)}
			}

			if resp != nil {
				fmt.Printf("XX OnExec sending back\n")
				if err := srv.Send(resp); err != nil {
					fmt.Printf("XX OnExec Send returned error %v", err)
				}
			}
		}
	*/
}

// returns true if not exit...
func getMessage(srv Remote_OnExecServer) (*Session, error) {
	// FIXME timeout?

	// receive data from stream
	msg, err := srv.Recv()
	fmt.Printf("XX OnExec RECV %v\n", msg)
	if err == io.EOF {
		// return will close stream from server side
		fmt.Println("XX OnExec EOF")
		return msg, nil
	}
	if err != nil {
		fmt.Printf("XX OnExec receive error %v", err)
		return nil, err
	}
	return msg, nil
}

/*

SERVER


  	//use wait group to allow process to be concurrent
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(count int64) {
			defer wg.Done()

      			//time sleep to simulate server process time
			time.Sleep(time.Duration(count) * time.Second)
			resp := pb.Response{Result: fmt.Sprintf("Request #%d For Id:%d", count, in.Id)}
			if err := srv.Send(&resp); err != nil {
				log.Printf("send error %v", err)
			}
			log.Printf("finishing request number : %d", count)
		}(int64(i))
	}

	wg.Wait()
	return nil
}


func SendText() error {

	timer := time.NewTicker(2 * time.Second)
	for {
		select {

		// Exit on stream context done
		case <-stream.Context().Done():
			return nil
		case <-timer.C:
			// Grab stats and output
			hwStats, err := s.GetStats()
			if err != nil {
				log.Println(err.Error())
			} else {

			}
			// Send the Hardware stats on the stream
			err = stream.Send(hwStats)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
*/
