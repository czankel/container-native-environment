package remote

import (
	"fmt"
	"os"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/runtime"
)

type process struct {
	container *container
}

func (proc *process) Wait() (<-chan runtime.ExitStatus, error) {
	fmt.Println("process.Wait")
	return nil, errdefs.NotImplemented()
}

func (proc *process) Signal(sig os.Signal) error {
	fmt.Println("process.Signal")
	return errdefs.NotImplemented()
}
