package discoveryservice

import (
	"context"

	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resource_v3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
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

// NewSnapshot returns a Snapshot object
func (c Cache) NewSnapshot(resourcesVersion string) xdss.Snapshot {

	snap, _ := cache_v3.NewSnapshot(resourcesVersion,
		map[resource_v3.Type][]cache_types.Resource{
			resource_v3.EndpointType:        {},
			resource_v3.ClusterType:         {},
			resource_v3.RouteType:           {},
			resource_v3.ScopedRouteType:     {},
			resource_v3.VirtualHostType:     {},
			resource_v3.ListenerType:        {},
			resource_v3.SecretType:          {},
			resource_v3.RuntimeType:         {},
			resource_v3.ExtensionConfigType: {},
		},
	)

	return Snapshot{v3: snap}
}
