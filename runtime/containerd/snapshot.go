//go:build linux

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
	size        int64
	inodes      int64
}

// The snapshot name consists of the domain and the containerID
func activeSnapshotName(domain, ctrID [16]byte) string {
	domStr := hex.EncodeToString(domain[:])
	cidStr := hex.EncodeToString(ctrID[:])
	return domStr + "-" + cidStr
}

func getSnapshotDomains(ctx context.Context, ctrdRun *containerdRuntime) ([][16]byte, error) {

	var domains [][16]byte

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	err := snapSvc.Walk(ctx, func(ctx context.Context, info snapshots.Info) error {

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

func getSnapshots(ctx context.Context,
	ctrdRun *containerdRuntime) ([]runtime.Snapshot, error) {
	var snaps []runtime.Snapshot

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	err := snapSvc.Walk(ctx, func(ctx context.Context, info snapshots.Info) error {

		usage, err := snapSvc.Usage(ctx, string(info.Name))
		if err != nil {
			return runtime.Errorf("failed to get snapshot usage: %v", err)
		}

		snaps = append(snaps, &snapshot{
			ctrdRuntime: ctrdRun,
			info:        info,
			size:        usage.Size,
			inodes:      usage.Inodes})

		return nil
	})
	if err != nil {
		return snaps, runtime.Errorf("failed to get snapshots: %v", err)
	}
	return snaps, nil
}

// getSnapshot returns the requested snapshot
// It returns an error if the snapshot doesn't exist
func getSnapshot(ctx context.Context,
	ctrdRun *containerdRuntime, snapName string) (runtime.Snapshot, error) {

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	info, err := snapSvc.Stat(ctx, snapName)
	if err != nil && ctrderr.IsNotFound(err) {
		return nil, errdefs.NotFound("snapshot", snapName)
	} else if err != nil {
		return nil, runtime.Errorf("failed to get snapshot: %v", err)
	}

	usage, err := snapSvc.Usage(ctx, string(info.Name))
	if err != nil {
		return nil, runtime.Errorf("failed to get snapshot usage: %v", err)
	}

	return &snapshot{
		ctrdRuntime: ctrdRun,
		info:        info,
		size:        usage.Size,
		inodes:      usage.Inodes}, nil
}

func getActiveSnapshot(ctx context.Context,
	ctrdRun *containerdRuntime, domain, id [16]byte) (runtime.Snapshot, error) {
	return getSnapshot(ctx, ctrdRun, activeSnapshotName(domain, id))
}

func commitSnapshot(ctx context.Context, ctrdRun *containerdRuntime,
	snap runtime.Snapshot, amend bool) (runtime.Snapshot, error) {

	activeSnapName := snap.Name()
	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	diffSvc := ctrdRun.client.DiffService()

	parentName := snap.Parent()
	if parentName == "" {
		return nil, runtime.Errorf("snapshot %s doesn't have a parent", snap.Name())
	}

	parentSnap, err := getSnapshot(ctx, ctrdRun, parentName)
	if err != nil {
		return nil, runtime.Errorf("parent snapshot '%v' not found: %v", parentName, err)
	}

	parentMnts, err := snapSvc.View(ctx, parentName+"-view", parentName)
	if err != nil {
		return nil, runtime.Errorf("creating snapshot '%v' failed: %v", parentName, err)
	}
	defer snapSvc.Remove(ctx, parentName+"-view")

	snapMnts, err := snapSvc.Mounts(ctx, snap.Name())
	if err != nil {
		return nil, runtime.Errorf("failed to mount snapshot: %v", err)
	}

	desc, err := diffSvc.Compare(ctx, parentMnts, snapMnts)
	if err != nil {
		return nil, runtime.Errorf("failed to create diff between snapshots: %v", err)
	}
	digest := desc.Digest.String()

	if parentName == digest {
		return nil, nil
	}

	labels := map[string]string{}
	labels["containerd.io/gc.root"] = time.Now().UTC().Format(time.RFC3339)

	if amend {

		amendName := parentSnap.Parent()
		if amendName == "" {
			return nil, runtime.Errorf("no snapshot to amend snapshot %s", snap.Name())
		}

		amendMnts, err := snapSvc.Prepare(ctx, amendName+"-amend", amendName)
		if err != nil {
			return nil, runtime.Errorf("failed to create temporary snapshot: %v", err)
		}

		desc, err = diffSvc.Apply(ctx, desc, amendMnts)
		if err != nil {
			snapSvc.Remove(ctx, amendName+"-amend")
			return nil, runtime.Errorf("failed to apply snapshot: %v", err)
		}

		digest = desc.Digest.String()
		err = snapSvc.Commit(ctx, digest, amendName+"-amend",
			snapshots.WithLabels(labels))
		if err != nil {
			snapSvc.Remove(ctx, amendName+"-amend")
			return nil, runtime.Errorf("failed to commit snapshot: %v", err)
		}

		// TODO: log a warning on errors for these
		snapSvc.Remove(ctx, activeSnapName)
		snapSvc.Remove(ctx, parentName)

	} else {

		err = snapSvc.Commit(ctx, digest, activeSnapName, snapshots.WithLabels(labels))
		if err != nil && ctrderr.IsAlreadyExists(err) {
			return nil, errdefs.AlreadyExists("snapshot", digest)
		}
		if err != nil {
			return nil, runtime.Errorf("failed to commit snapshot: %v", err)
		}
	}

	info, err := snapSvc.Stat(ctx, digest)
	return &snapshot{ctrdRuntime: ctrdRun, info: info}, nil
}

func createSnapshot(ctx context.Context, ctrdRun *containerdRuntime,
	snapName, parentName string, mutable bool) ([]mount.Mount, *snapshot, error) {

	var mounts []mount.Mount

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)

	// check if the snapshot already exists, take mutable flag into account
	info, err := snapSvc.Stat(ctx, snapName)
	if err != nil && ctrderr.IsAlreadyExists(err) {
		return nil, nil, errdefs.AlreadyExists("snapshot", snapName)
	}
	if err != nil && !ctrderr.IsNotFound(err) {
		return nil, nil, runtime.Errorf("failed to create snapshot: %v", err)
	}
	if err == nil && (info.Kind == snapshots.KindActive && mutable ||
		info.Kind == snapshots.KindView && !mutable) {
		mounts, err = snapSvc.Mounts(ctx, snapName)
		if err != nil {
			return nil, nil, runtime.Errorf("failed to get snapshot mounts: %v", err)
		}
	}

	// otherwise, create snapshot
	if mounts == nil {

		labels := map[string]string{}
		labels["containerd.io/gc.root"] = time.Now().UTC().Format(time.RFC3339)

		if mutable {
			mounts, err = snapSvc.Prepare(ctx, snapName, parentName,
				snapshots.WithLabels(labels))
		} else {
			mounts, err = snapSvc.View(ctx, snapName, parentName,
				snapshots.WithLabels(labels))
		}
		if err != nil {
			return nil, nil,
				runtime.Errorf("failed to create snapshot '%s': %v", snapName, err)
		}

		info, err = snapSvc.Stat(ctx, snapName)
		if err != nil {
			return nil, nil, runtime.Errorf("failed to create snapshot")
		}
	}

	return mounts, &snapshot{ctrdRuntime: ctrdRun, info: info}, nil
}

type mounter struct{}

func (mounter) Mount(target string, mounts ...mount.Mount) error {
	return mount.All(mounts, target)
}

func (mounter) Unmount(target string) error {
	return mount.UnmountAll(target, 0)
}

func createActiveSnapshot(ctx context.Context, ctrdRun *containerdRuntime,
	img *image, domain, id [16]byte, snap runtime.Snapshot) error {

	activeSnapName := activeSnapshotName(domain, id)
	var rootFsSnapName string
	if snap != nil {
		rootFsSnapName = snap.Name()
	} else {
		diffIDs, err := img.ctrdImage.RootFS(ctx)
		if err != nil {
			return runtime.Errorf("failed to get rootfs: %v", err)
		}
		rootFsSnapName = identity.ChainID(diffIDs).String()
		_, err = getSnapshot(ctx, ctrdRun, rootFsSnapName)

		// unpack 'image' if root snapshot was removed
		if err != nil && errors.Is(err, errdefs.ErrNotFound) {
			img.Unpack(ctx)
			digest := identity.ChainID(diffIDs).String()
			_, _, err = createSnapshot(ctx, img.ctrdRuntime, digest, digest, false)
		}
		if err != nil {
			return err
		}
	}

	// delete all 'old' snapshots down to the new rootfs or the image
	if rootFsSnapName == activeSnapName {
		return errdefs.InternalError("Cannot set rootfs to active layer")
	}

	snapName := activeSnapName
	for snapName != rootFsSnapName {
		snap, err := getSnapshot(ctx, ctrdRun, snapName)
		if err != nil && errors.Is(err, errdefs.ErrNotFound) {
			break
		}
		if err != nil {
			return err
		}
		err = deleteSnapshot(ctx, ctrdRun, snapName)
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
	_, snap, err := createSnapshot(ctx, ctrdRun, activeSnapName, rootFsSnapName, true /* mutable */)
	if err != nil {
		return err
	}

	return nil
}

func updateSnapshot(ctx context.Context, ctrdRun *containerdRuntime,
	domain, id [16]byte, amend bool) (runtime.Snapshot, error) {

	activeSnapName := activeSnapshotName(domain, id)
	snap, err := getSnapshot(ctx, ctrdRun, activeSnapName)
	if err != nil && errors.Is(err, errdefs.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	snap, err = commitSnapshot(ctx, ctrdRun, snap, false /* amend */)
	if err != nil {
		return nil, err
	}

	_, _, err = createSnapshot(ctx, ctrdRun, activeSnapName, snap.Name(), true /* mutable */)
	return snap, err
}

func getActiveSnapMounts(ctx context.Context,
	ctrdRun *containerdRuntime, dom, cid [16]byte) ([]mount.Mount, error) {

	snapName := activeSnapshotName(dom, cid)

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	return snapSvc.Mounts(ctx, snapName)
}

// delete the specified snapshot; return ErrNotFound if the snapshot doesn exist and
// ErrInUse if it is still in use and referenced.
func deleteSnapshot(ctx context.Context, ctrdRun *containerdRuntime, snapName string) error {

	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	err := snapSvc.Remove(ctx, snapName)
	if err != nil && ctrderr.IsNotFound(err) {
		return errdefs.NotFound("snapshot", snapName)
	}
	if err != nil && ctrderr.IsFailedPrecondition(err) {
		return errdefs.InUse("snapshot", snapName)
	}

	return nil
}

func deleteActiveSnapshot(ctx context.Context, ctrdRun *containerdRuntime, domain, id [16]byte) error {
	activeSnapName := activeSnapshotName(domain, id)
	err := deleteSnapshot(ctx, ctrdRun, activeSnapName)
	if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
		return err
	}

	return nil
}

// delete all unrefeenced containers for the provided container starting with the active snapshot
func deleteContainerSnapshots(ctx context.Context,
	ctrdRun *containerdRuntime, domain, id [16]byte) error {

	snapName := activeSnapshotName(domain, id)

	snapMap := make(map[string]*snapshot)
	snapRefs := make(map[string]int)
	snapSvc := ctrdRun.client.SnapshotService(containerd.DefaultSnapshotter)
	err := snapSvc.Walk(ctx, func(ctx context.Context, info snapshots.Info) error {

		snapMap[info.Name] = &snapshot{ctrdRuntime: ctrdRun, info: info}

		if info.Parent != "" {
			snapRefs[info.Parent] = snapRefs[info.Parent] + 1
		}
		return nil
	})
	if err != nil {
		return runtime.Errorf("failed to get snapshot list: %v", err)
	}

	for refC := 0; refC == 0; {
		snap, ok := snapMap[snapName]
		if !ok {
			break
		}

		parent := snap.info.Parent

		err = snapSvc.Remove(ctx, snapName)
		if err != nil && !ctrderr.IsNotFound(err) {
			return runtime.Errorf("failed to remove snapshot: %v", err)
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

func (snap *snapshot) Size() int64 {
	return snap.size
}

func (snap *snapshot) Inodes() int64 {
	return snap.inodes
}
