package discoveryservice

import (
	xdss "github.com/3scale/marin3r/pkg/discoveryservice/xdss"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

// Cache implements "github.com/3scale/marin3r/pkg/discoveryservice/xdss".Cache for envoy API v2.
type Cache struct {
	v2 cache_v2.SnapshotCache
}

// NewCache returns a Cache object.
func NewCache(v2 cache_v2.SnapshotCache) Cache {
	return Cache{v2: v2}
}

// SetSnapshot updates a snapshot for a node.
func (c Cache) SetSnapshot(nodeID string, snap xdss.Snapshot) error {

	return c.v2.SetSnapshot(nodeID, *snap.(Snapshot).v2)
}

// GetSnapshot gets the snapshot for a node, and returns an error if not found.
func (c Cache) GetSnapshot(nodeID string) (xdss.Snapshot, error) {

	snap, err := c.v2.GetSnapshot(nodeID)
	if err != nil {
		return &Snapshot{}, err
	}
	return &Snapshot{v2: &snap}, nil
}

// ClearSnapshot clears snapshot and info for a node.
func (c Cache) ClearSnapshot(nodeID string) {

	c.v2.ClearSnapshot(nodeID)
}

// NewSnapshot returns a Snapshot object
func (c Cache) NewSnapshot(resourcesVersion string) xdss.Snapshot {

	snap := &cache_v2.Snapshot{Resources: [6]cache_v2.Resources{}}
	snap.Resources[cache_types.Listener] = cache_v2.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Endpoint] = cache_v2.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Cluster] = cache_v2.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Route] = cache_v2.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Secret] = cache_v2.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Runtime] = cache_v2.NewResources(resourcesVersion, []cache_types.Resource{})

	return Snapshot{v2: snap}
}
