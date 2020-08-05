package containerd

import (
	"time"

	"github.com/containerd/containerd/snapshots"
)

type snapshot struct {
	info snapshots.Info
}

func (snap *snapshot) Name() string {
	return snap.info.Name
}

func (snap *snapshot) Parent() string {
	return snap.info.Parent
}

func (snap *snapshot) CreatedAt() time.Time {
	return snap.info.Created
}
