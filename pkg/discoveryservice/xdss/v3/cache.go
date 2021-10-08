package discoveryservice

import (
	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

// Cache implements "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss".Cache for envoy API v3.
type Cache struct {
	v3 cache_v3.SnapshotCache
}

// NewCache returns a Cache object.
func NewCache(v3 cache_v3.SnapshotCache) Cache {
	return Cache{v3: v3}
}

// SetSnapshot updates a snapshot for a node.
func (c Cache) SetSnapshot(nodeID string, snap xdss.Snapshot) error {

	return c.v3.SetSnapshot(nodeID, *snap.(Snapshot).v3)
}

// GetSnapshot gets the snapshot for a node, and returns an error if not found.
func (c Cache) GetSnapshot(nodeID string) (xdss.Snapshot, error) {

	snap, err := c.v3.GetSnapshot(nodeID)
	if err != nil {
		return &Snapshot{}, err
	}
	return &Snapshot{v3: &snap}, nil
}

// ClearSnapshot clears snapshot and info for a node.
func (c Cache) ClearSnapshot(nodeID string) {

	c.v3.ClearSnapshot(nodeID)
}

// NewSnapshot returns a Snapshot object
func (c Cache) NewSnapshot(resourcesVersion string) xdss.Snapshot {

	snap := &cache_v3.Snapshot{Resources: [7]cache_v3.Resources{}}
	snap.Resources[cache_types.Listener] = cache_v3.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Endpoint] = cache_v3.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Cluster] = cache_v3.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Route] = cache_v3.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Secret] = cache_v3.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Runtime] = cache_v3.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.ExtensionConfig] = cache_v3.NewResources(resourcesVersion, []cache_types.Resource{})

	return Snapshot{v3: snap}
}
