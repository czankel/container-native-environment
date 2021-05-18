package containerd

import (
	"context"
	"encoding/hex"
	"strings"
	"time"

	"github.com/containerd/containerd"
	ctrderr "github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/snapshots"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/runtime"
)

type snapshot struct {
	ctrdRuntime *containerdRuntime
	info        snapshots.Info
}

// The snapshot name consists of the domain and the containerID
func activeSnapshotName(domain, ctrID [16]byte) string {
	domStr := hex.EncodeToString(domain[:])
	cidStr := hex.EncodeToString(ctrID[:])
	return domStr + "-" + cidStr
}

// getSnapshot returns the requested snapshot
// It returns an error if the snapshot doesn't exist
func getSnapshot(ctrdRun *containerdRuntime, snapName string) (runtime.Snapshot, error) {

	ctrdCtx := ctrdRun.context
	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	info, err := snapSvc.Stat(ctrdCtx, snapName)
	if err != nil && ctrderr.IsNotFound(err) {
		return nil, errdefs.NotFound("snapshot", snapName)
	} else if err != nil {
		return nil, err
	}

	return &snapshot{ctrdRuntime: ctrdRun, info: info}, nil
}

func getActiveSnapshot(ctrdRun *containerdRuntime, domain, id [16]byte) (runtime.Snapshot, error) {
	return getSnapshot(ctrdRun, activeSnapshotName(domain, id))
}

func getSnapshots(ctrdRun *containerdRuntime) ([]runtime.Snapshot, error) {
	var snaps []runtime.Snapshot

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	err := snapSvc.Walk(ctrdRun.context, func(ctx context.Context, info snapshots.Info) error {
		snaps = append(snaps, &snapshot{info: info})
		return nil
	})
	return snaps, err
}

func getSnapshotDomains(ctrdRun *containerdRuntime) ([][16]byte, error) {

	var domains [][16]byte

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	err := snapSvc.Walk(ctrdRun.context, func(ctx context.Context, info snapshots.Info) error {

		name := string(info.Name)
		idx := strings.Index(name, "-")
		if idx == 32 {
			str, err := hex.DecodeString(name[:32])
			if err != nil {
				return runtime.Errorf("failed to decode domain '%s': $v", name, err)
			}

			var dom [16]byte
			copy(dom[:], str)

			found := false
			for _, d := range domains {
				if d == dom {
					found = true
					break
				}
			}
			if !found {
				domains = append(domains, dom)
			}
		}
		return nil
	})

	return domains, err
}

func deleteSnapshot(ctrdRun *containerdRuntime, name string) error {

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	err := snapSvc.Remove(ctrdRun.context, name)
	if err != nil {
		return runtime.Errorf("delete snapshot '%s' failed: %v", name, err)
	}

	return nil
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
