package remote

import (
	"time"

	"github.com/czankel/cne/service"
)

type snapshot struct {
	remSnapshot *service.Snapshot
}

func (snap *snapshot) Name() string {
	return snap.remSnapshot.Name
}

func (snap *snapshot) Parent() string {
	return snap.remSnapshot.Parent
}

func (snap *snapshot) CreatedAt() time.Time {
	return service.ConvPbTimeToGo(snap.remSnapshot.Created)
}

func (snap *snapshot) Size() int64 {
	return snap.remSnapshot.Size
}

func (snap *snapshot) Inodes() int64 {
	return snap.remSnapshot.Inodes
}
