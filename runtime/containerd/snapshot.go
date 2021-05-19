package containerd

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/containerd/containerd"
	ctrderr "github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/snapshots"
	"github.com/opencontainers/image-spec/identity"

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

func createSnapshot(ctrdRun *containerdRuntime,
	snapName, parentName string, mutable bool) ([]mount.Mount, *snapshot, error) {

	ctrdCtx := ctrdRun.context

	var mounts []mount.Mount

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)

	// check if the snapshot already exists, take mutable flag into account
	info, err := snapSvc.Stat(ctrdCtx, snapName)
	if err != nil && !ctrderr.IsNotFound(err) {
		return nil, nil, err
	}
	if err == nil && (info.Kind == snapshots.KindActive && mutable ||
		info.Kind == snapshots.KindView && !mutable) {
		mounts, err = snapSvc.Mounts(ctrdCtx, snapName)
		if err != nil {
			return nil, nil, runtime.Errorf("failed to get snapshot mounts: %v", err)
		}
	}

	// otherwise, create snapshot
	if mounts == nil {

		labels := map[string]string{}
		labels["containerd.io/gc.root"] = time.Now().UTC().Format(time.RFC3339)

		if mutable {
			mounts, err = snapSvc.Prepare(ctrdCtx, snapName, parentName,
				snapshots.WithLabels(labels))
		} else {
			mounts, err = snapSvc.View(ctrdCtx, snapName, parentName,
				snapshots.WithLabels(labels))
		}
		if err != nil {
			return nil, nil,
				runtime.Errorf("failed to create snapshot '%s': %v", snapName, err)
		}

		info, err = snapSvc.Stat(ctrdCtx, snapName)
		if err != nil {
			return nil, nil, runtime.Errorf("failed to create snapshot")
		}
	}

	return mounts, &snapshot{ctrdRuntime: ctrdRun, info: info}, nil
}

func createActiveSnapshot(ctrdRun *containerdRuntime,
	img *image, domain, id [16]byte, snap runtime.Snapshot) error {

	activeSnapName := activeSnapshotName(domain, id)
	var rootFsSnapName string
	if snap != nil {
		rootFsSnapName = snap.Name()
	} else {
		diffIDs, err := img.ctrdImage.RootFS(ctrdRun.context)
		if err != nil {
			return runtime.Errorf("failed to get rootfs: %v", err)
		}
		rootFsSnapName = identity.ChainID(diffIDs).String()
	}

	// delete all 'old' snapshots down to the new rootfs or the image
	if rootFsSnapName == activeSnapName {
		return errdefs.InternalError("Cannot set rootfs to active layer")
	}

	snapName := activeSnapName
	for snapName != rootFsSnapName {
		snap, err := getSnapshot(ctrdRun, snapName)
		if err != nil && errors.Is(err, errdefs.ErrNotFound) {
			break
		}
		if err != nil {
			return err
		}
		err = deleteSnapshot(ctrdRun, snapName)
		if err != nil && errors.Is(err, errdefs.ErrNotFound) {
			break
		}
		if err != nil && errors.Is(err, errdefs.ErrInUse) {
			break
		}
		if err != nil {
			break
		}
		snapName = snap.Parent()
	}

	// create active snapshot based on the new rootFs
	_, snap, err := createSnapshot(ctrdRun, activeSnapName, rootFsSnapName, true /* mutable */)
	if err != nil {
		return err
	}

	return nil
}

func getActiveSnapMounts(ctrdRun *containerdRuntime, dom, cid [16]byte) ([]mount.Mount, error) {

	snapName := activeSnapshotName(dom, cid)

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	return snapSvc.Mounts(ctrdRun.context, snapName)
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

// delete all unrefeenced containers for the provided container starting with the active snapshot
func deleteContainerSnapshots(ctrdRun *containerdRuntime, domain, id [16]byte) error {

	snapName := activeSnapshotName(domain, id)

	snapMap := make(map[string]*snapshot)
	snapRefs := make(map[string]int)
	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	err := snapSvc.Walk(ctrdRun.context, func(ctx context.Context, info snapshots.Info) error {

		snapMap[info.Name] = &snapshot{ctrdRuntime: ctrdRun, info: info}

		if info.Parent != "" {
			snapRefs[info.Parent] = snapRefs[info.Parent] + 1
		}
		return nil
	})
	if err != nil {
		return err
	}

	for refC := 0; refC == 0; {
		snap, ok := snapMap[snapName]
		if !ok {
			break
		}

		parent := snap.info.Parent

		err = snapSvc.Remove(ctrdRun.context, snapName)
		if err != nil && !ctrderr.IsNotFound(err) {
			return err
		}

		if parent == "" {
			break
		}

		refC = snapRefs[parent] - 1
		if refC == 0 {
			snapRefs[parent] = -1 // mark already handled
		} else {
			snapRefs[parent] = refC
		}
		snapName = parent
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
