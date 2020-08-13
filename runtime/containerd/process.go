// Package containerd implements the runtime interface for the ContainerD Dameon containerd.io
package containerd

import (
	"github.com/containerd/containerd"
	ctrderr "github.com/containerd/containerd/errdefs"

	"github.com/czankel/cne/runtime"
)

type process struct {
	container *container
	ctrdProc  containerd.Process
}

func (proc *process) Wait() (<-chan runtime.ExitStatus, error) {

	ctrdRun := proc.container.ctrdRuntime
	runExitStatus := make(chan runtime.ExitStatus)

	ctrdExitStatus, err := proc.ctrdProc.Wait(ctrdRun.context)
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
