//go:build linux

package containerd

import (
	"context"
	"os"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	ctrderr "github.com/containerd/containerd/errdefs"

	"github.com/czankel/cne/runtime"
)

type process struct {
	container *container
	ctrdProc  containerd.Process
	code      uint32
	exitedAt  time.Time
}

// Wait waits for the process to complete and returns the result or
// the error for any context operation.
func (proc *process) Wait(ctx context.Context) error {

	ctrdExitStatus, err := proc.ctrdProc.Wait(ctx)
	if err != nil && ctrderr.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return runtime.Errorf("wait failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case exitStatus := <-ctrdExitStatus:
			code, exitedAt, _err := exitStatus.Result()
			proc.code = code
			proc.exitedAt = exitedAt
			err = _err
			return nil
		}
	}

	return err
}

func (proc *process) SigInt(ctx context.Context) error {

	s := sig.(syscall.Signal)
	err := proc.ctrdProc.Kill(ctx, s)
	if err != nil {
		return runtime.Errorf("kill failed: %v", err)
	}
	return nil
}

func (proc *process) ExitCode() uint32 {
	return proc.code
}

func (proc *process) ExitTime() time.Time {
	return proc.exitedAt
}
