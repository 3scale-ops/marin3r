package discoveryservice

import (
	"context"

	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

var _ xdss.Cache = Cache{}

// Cache implements "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss".Cache for envoy API v3.
type Cache struct {
	v3 cache_v3.SnapshotCache
}

// NewCache returns a Cache object.
func NewCache() Cache {
	return Cache{v3: cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)}
}

// NewCache returns a Cache object.
func NewCacheFromSnapshotCache(v3 cache_v3.SnapshotCache) Cache {
	return Cache{v3: v3}
}

// SetSnapshot updates a snapshot for a node.
func (c Cache) SetSnapshot(ctx context.Context, nodeID string, snap xdss.Snapshot) error {

	return c.v3.SetSnapshot(ctx, nodeID, snap.(Snapshot).v3)
}

// GetSnapshot gets the snapshot for a node, and returns an error if not found.
func (c Cache) GetSnapshot(nodeID string) (xdss.Snapshot, error) {

	snap, err := c.v3.GetSnapshot(nodeID)
	if err != nil {
		return &Snapshot{}, err
	}
	return &Snapshot{v3: snap.(*cache_v3.Snapshot)}, nil
}

// ClearSnapshot clears snapshot and info for a node.
func (c Cache) ClearSnapshot(nodeID string) {

	c.v3.ClearSnapshot(nodeID)
}

func (c Cache) NewSnapshot() xdss.Snapshot {

	return NewSnapshot()
}
