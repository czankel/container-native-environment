//go:build linux

// Package containerd implements the runtime interface for the ContainerD Dameon containerd.io
//
package containerd

import (
	"context"
	"os"
	"syscall"

	"github.com/containerd/containerd"
	ctrderr "github.com/containerd/containerd/errdefs"

	"github.com/czankel/cne/runtime"
)

type process struct {
	container *container
	ctrdProc  containerd.Process
}

// Wait waits for the process to complete and returns the result or
// the error for any context operation.
func (proc *process) Wait(ctx context.Context) (<-chan runtime.ExitStatus, error) {

	ctrdExitStatus, err := proc.ctrdProc.Wait(ctx)
	runExitStatus := make(chan runtime.ExitStatus)
	if err != nil && ctrderr.IsNotFound(err) {
		runExitStatus <- runtime.ExitStatus{}
		return runExitStatus, nil
	}
	if err != nil {
		return nil, runtime.Errorf("wait failed: %v", err)
	}

	go func() {
		defer close(runExitStatus)

		exitStatus := <-ctrdExitStatus
		code, exitedAt, err := exitStatus.Result()
		runExitStatus <- runtime.ExitStatus{
			ExitTime: exitedAt,
			Error:    err,
			Code:     code,
		}
	}()

	return runExitStatus, nil
}

func (proc *process) Signal(ctx context.Context, sig os.Signal) error {

	s := sig.(syscall.Signal)
	err := proc.ctrdProc.Kill(ctx, s)
	if err != nil {
		return runtime.Errorf("kill failed: %v", err)
	}
	return nil
}
