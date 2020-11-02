package discoveryservice

import (
	xdss "github.com/3scale/marin3r/pkg/discoveryservice/xdss"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

type Cache struct {
	v3 cache_v3.SnapshotCache
}

func NewCache(v3 cache_v3.SnapshotCache) Cache {
	return Cache{v3: v3}
}

func (c Cache) SetSnapshot(nodeID string, snap xdss.Snapshot) error {

	return c.v3.SetSnapshot(nodeID, *snap.(Snapshot).v3)
}

func (c Cache) GetSnapshot(nodeID string) (xdss.Snapshot, error) {

	snap, err := c.v3.GetSnapshot(nodeID)
	if err != nil {
		return &Snapshot{}, err
	}
	return &Snapshot{v3: &snap}, nil
}

func (c Cache) ClearSnapshot(nodeID string) {

	c.v3.ClearSnapshot(nodeID)
}

func (c Cache) NewSnapshot(resourcesVersion string) xdss.Snapshot {

	snap := &cache_v3.Snapshot{Resources: [6]cache_v3.Resources{}}
	snap.Resources[cache_types.Listener] = cache_v3.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Endpoint] = cache_v3.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Cluster] = cache_v3.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Route] = cache_v3.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Secret] = cache_v3.NewResources(resourcesVersion, []cache_types.Resource{})
	snap.Resources[cache_types.Runtime] = cache_v3.NewResources(resourcesVersion, []cache_types.Resource{})

	return Snapshot{v3: snap}
}
