package remote

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	runspecs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/runtime"
	"github.com/czankel/cne/service"
)

type container struct {
	remRuntime    *remoteRuntime
	remDomain     [16]byte
	remID         [16]byte
	remGeneration [16]byte
	remUID        uint32
	remCreatedAt  time.Time
	remUpdatedAt  time.Time
	image         string
}

// pbContainer is a helper function to convert a remote container to a protobuf container
func pbContainer(ns string, remCtr *container) *service.Container {
	return &service.Container{
		Namespace: ns,
		Id: &service.ContainerID{
			Domain:     remCtr.remDomain[:],
			Id:         remCtr.remID[:],
			Generation: remCtr.remGeneration[:],
			Uid:        remCtr.remUID,
		},
		Image: remCtr.image,
	}

}

// composeID composes an ID from the domain and container ID
func composeID(domain [16]byte, id [16]byte) string {
	return hex.EncodeToString(domain[:]) + "-" + hex.EncodeToString(id[:])
}

func (ctr *container) Name() string {
	return composeID(ctr.remDomain, ctr.remID) + "-" +
		hex.EncodeToString(ctr.remGeneration[:])
}

func (ctr *container) Runtime() runtime.Runtime {
	return ctr.remRuntime
}

func (ctr *container) Domain() [16]byte {
	return ctr.remDomain
}

func (ctr *container) ID() [16]byte {
	return ctr.remID
}

func (ctr *container) Generation() [16]byte {
	return ctr.remGeneration
}

func (ctr *container) UID() uint32 {
	return ctr.remUID
}

func (ctr *container) CreatedAt() time.Time {
	return ctr.remCreatedAt
}

func (ctr *container) UpdatedAt() time.Time {
	return ctr.remUpdatedAt
}

func (ctr *container) SetRootFS(ctx context.Context,
	snap runtime.Snapshot) error {

	remRun := ctr.remRuntime
	if remRun.conn == nil {
		return errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	snapName := ""
	if snap != nil {
		snapName = snap.Name()
	}
	pbRFS := &service.RootFS{
		Namespace: ns,
		Container: &service.ContainerID{
			Domain:     ctr.remDomain[:],
			Id:         ctr.remID[:],
			Generation: ctr.remGeneration[:],
			Uid:        ctr.remUID,
		},
		Snapshot: snapName,
	}

	_, err := remRun.client.OnSetRootFS(ctx, pbRFS)
	if err != nil {
		return service.ConvPbErrorToGo(err)
	}

	return nil
}

func (ctr *container) Create(ctx context.Context) error {

	remRun := ctr.remRuntime
	if remRun.conn == nil {
		return errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	fmt.Printf("XX call OnCreateContainer\n")
	_, err := remRun.client.OnCreateContainer(ctx, pbContainer(ns, ctr))
	fmt.Printf("XX returned from OnCreateContainer\n")
	if err != nil {
		fmt.Printf("XX have error, convert %v\n", err)
		x := service.ConvPbErrorToGo(err)
		fmt.Printf("XX error converted %v\n", x)
		return x
	}

	return nil
}

func (ctr *container) UpdateSpec(newSpec *runspecs.Spec) error {
	fmt.Println("container.UpdateSpec")
	return errdefs.NotImplemented()
}

func (ctr *container) Snapshot(ctx context.Context) (runtime.Snapshot, error) {

	remRun := ctr.remRuntime
	if remRun.conn == nil {
		return nil, errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	remSnap, err := remRun.client.OnContainerSnapshot(ctx, pbContainer(ns, ctr))
	if err != nil {
		return nil, service.ConvPbErrorToGo(err)
	}

	return &snapshot{remSnapshot: remSnap}, nil
}

func (ctr *container) Amend(ctx context.Context) (runtime.Snapshot, error) {

	remRun := ctr.remRuntime
	if remRun.conn == nil {
		return nil, errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	pbCtr := &service.Container{
		Namespace: ns,
		Id: &service.ContainerID{
			Domain:     ctr.remDomain[:],
			Id:         ctr.remID[:],
			Generation: ctr.remGeneration[:],
			Uid:        ctr.remUID,
		},
	}

	pbSnap, err := remRun.client.OnAmend(ctx, pbCtr)
	if err != nil {
		return nil, service.ConvPbErrorToGo(err)
	}
	return &snapshot{
		remSnapshot: pbSnap,
	}, nil
}

func (ctr *container) Commit(ctx context.Context, gen [16]byte) error {

	remRun := ctr.remRuntime
	if remRun.conn == nil {
		return errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	pbCtr := &service.Container{
		Namespace: ns,
		Id: &service.ContainerID{
			Domain:     ctr.remDomain[:],
			Id:         ctr.remID[:],
			Generation: ctr.remGeneration[:],
			Uid:        ctr.remUID,
		},
	}

	_, err := remRun.client.OnCommit(ctx, pbCtr)
	if err != nil {
		return service.ConvPbErrorToGo(err)
	}
	return nil
}

func (ctr *container) Mount(ctx context.Context,
	destination string, source string) error {

	// FIXME: remote Mount
	return errdefs.NotImplemented()
}

type process struct {
	container     *container
	context       context.Context
	client        service.Remote_OnExecClient
	code          uint32
	runExitStatus chan runtime.ExitStatus
}

// FIXME: pass context to exec instead of using the runExitStatus?
func (ctr *container) Exec(ctx context.Context, stream runtime.Stream,
	runProcSpec *runtime.ProcessSpec) (runtime.Process, error) {

	fmt.Printf("XX EXEC\n")

	// FIXME ns := ctx.Value(ContextNamespace).(string)

	remRun := ctr.remRuntime
	client, err := remRun.client.OnExec(ctx)
	if err != nil {
		fmt.Printf("XX Returned error %v\n", err)
		return nil, service.ConvPbErrorToGo(err)
	}

	// FIXME: is that the same context?? ctx := client.Context()
	runExitStatus := make(chan runtime.ExitStatus)

	// local <- remote
	go func() {
		for {
			resp, err := client.Recv()
			fmt.Printf("XX Exec RECV %v\n", resp)
			if err == io.EOF {
				runExitStatus <- runtime.ExitStatus{
					Error: errdefs.NotConnected(),
					Code:  1,
				}
				return
			}

			if resp.Command == service.Session_STDERR {
				_, err = stream.Stderr.Write(resp.Data)
			} else if resp.Command == service.Session_STDOUT {
				_, err = stream.Stdout.Write(resp.Data)
			}
			// FIXME: send anything before the error
			if err != nil {
				fmt.Printf("XX Exec got error %v\n", err)
				runExitStatus <- runtime.ExitStatus{
					Error: err,
					Code:  1,
				}
				return
			}

			if resp.Command == service.Session_EXIT {
				runExitStatus <- runtime.ExitStatus{
					ExitTime: service.ConvPbTimeToGo(resp.Timestamp),
					Error:    errors.New(resp.Error),
					Code:     resp.Code,
				}
				return
			}
		}

	}()

	msg := &service.Session{
		Command: service.Session_EXEC,
		UID:     runProcSpec.UID,
		GID:     runProcSpec.GID,
		Args:    runProcSpec.Args,
		Envs:    runProcSpec.Env,
	}

	err = client.Send(msg)
	if err != nil {
		return nil, service.ConvPbErrorToGo(err)
	}

	// local -> remote
	go func() {
		var buffer []byte

		for {
			len, err := stream.Stdin.Read(buffer)
			if len > 0 {
				fmt.Printf("XX Send %d %v\n", len, err)
				msg := &service.Session{
					Command: service.Session_STDIN,
					Data:    buffer}
				err = client.Send(msg)

			} else if err == io.EOF {
				fmt.Printf("XX Send EOF\n")
				if err := client.CloseSend(); err != nil {
				}
				return
			}

			// FIXME notify Wait??
			if err != nil {
				// FIXME close(remExitStatus)
				// FIXME: close(remExitStatus)
				return
			}
		}
	}()
	/*
		// handle context exits
		go func() {

			//  		select {
			//  		case <-ctx.Done():
			//  			return ctx.Err()
			//  		case out <- v:
			//  		}

			<-ctx.Done()
			fmt.Printf("XX Exec got Done\n")
			if err := ctx.Err(); err != nil {
				fmt.Println(err)
			}
			// FIXME close(done)
			// FIXME: close(remExitStatus)
		}()
	*/

	return &process{
		container:     ctr,
		client:        client,
		runExitStatus: runExitStatus,
	}, nil

}

// FIXME: rename to SigInt and add SigKill; not OS independent
func (proc *process) Signal(ctx context.Context, sig os.Signal) error {

	fmt.Printf("XX Signal!\n")

	err := proc.client.Send(&service.Session{
		Command: service.Session_SIGINT,
	})

	if err != nil {
		return service.ConvPbErrorToGo(err)
	}

	return nil

}

// FIXME: wrong?? callee cannot cancel... Exec will just wait, so caller has to handle cancellation ...
func (proc *process) Wait(ctx context.Context) error {
	fmt.Printf("XX Wait, which just exists\n")

	<-ctx.Done()
	fmt.Printf("XX Exec got Done\n")
	if err := ctx.Err(); err != nil {
		fmt.Println(err)
	}

	return nil
}

func (proc *process) ExitCode() uint32 {
	return proc.code
}

func (proc *process) ExitTime() time.Time {
	return time.Now()
}

func (ctr *container) Processes(ctx context.Context) ([]runtime.Process, error) {

	// FIXME: remote Processes
	return nil, errdefs.NotImplemented()
}

func (ctr *container) Delete(ctx context.Context) error {

	fmt.Printf("XX ctr.Delete\n")

	remRun := ctr.remRuntime
	if remRun.conn == nil {
		return errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	_, err := remRun.client.OnDeleteContainer(ctx, pbContainer(ns, ctr))
	if err != nil {
		fmt.Printf("XX OnDelete returned %v\n", err)
		return service.ConvPbErrorToGo(err)
	}

	return nil
}

func (ctr *container) Purge(ctx context.Context) error {
	remRun := ctr.remRuntime
	if remRun.conn == nil {
		return errdefs.NotConnected()
	}

	ns := ctx.Value(ContextNamespace).(string)
	_, err := remRun.client.OnPurgeContainer(ctx, pbContainer(ns, ctr))
	if err != nil {
		return service.ConvPbErrorToGo(err)
	}

	return nil
}
