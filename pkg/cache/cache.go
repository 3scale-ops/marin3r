// Copyright 2020 rvazquez@redhat.com
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package cache

import (
	"strconv"

	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache"
)

/*
Package cache offers a simple implementation of a cache to store
envoy xDS resources. It offers a set of methods to set/get resources
in the cache and to push the cache to the xDS server for publishing.

The structure of this cache copies the structure of the cache.SnapshotCache
struct of the go-control-plane package, the one that is used in the end to
push resources to the xDS server. This is so to avoid unnecessary
transformations between cache objects.

For reference, an example cache struct:

	c := map[string][6]xds_cache.Resources{
			"my-node-id": [6]xds_cache.Resources{
				xds_cache.Resources{Version: "1", Items: map[string]xds_cache.Resource{}}, // Endpoint
				xds_cache.Resources{Version: "1", Items: map[string]xds_cache.Resource{}}, // Cluster
				xds_cache.Resources{Version: "1", Items: map[string]xds_cache.Resource{}}, // Route
				xds_cache.Resources{Version: "1", Items: map[string]xds_cache.Resource{}}, // Listener
				xds_cache.Resources{Version: "1", Items: map[string]xds_cache.Resource{}}, // Secret
				xds_cache.Resources{Version: "1", Items: map[string]xds_cache.Resource{}}, // Runtime
		},
	}


*/

const (
	startingVersion = 1
	// Endpoint cache resource type
	Endpoint xds_cache.ResponseType = xds_cache.Endpoint
	// Cluster cache resource type
	Cluster xds_cache.ResponseType = xds_cache.Cluster
	// Route cache resource type
	Route xds_cache.ResponseType = xds_cache.Route
	// Listener cache resource type
	Listener xds_cache.ResponseType = xds_cache.Listener
	// Secret cache resurce type
	Secret xds_cache.ResponseType = xds_cache.Secret
	// Runtime cache resource type
	Runtime xds_cache.ResponseType = xds_cache.Runtime
)

// Cache ...
type Cache map[string]*xds_cache.Snapshot

// NewCache ...
func NewCache() Cache {
	return Cache{}
}

// NewNodeCache ...
func (cache Cache) NewNodeCache(nodeID string) {

	version := strconv.Itoa(startingVersion)

	ncache := xds_cache.Snapshot{Resources: [6]xds_cache.Resources{}}
	ncache.Resources[Listener] = xds_cache.NewResources(version, []xds_cache.Resource{})
	ncache.Resources[Endpoint] = xds_cache.NewResources(version, []xds_cache.Resource{})
	ncache.Resources[Cluster] = xds_cache.NewResources(version, []xds_cache.Resource{})
	ncache.Resources[Route] = xds_cache.NewResources(version, []xds_cache.Resource{})
	ncache.Resources[Secret] = xds_cache.NewResources(version, []xds_cache.Resource{})
	ncache.Resources[Runtime] = xds_cache.NewResources(version, []xds_cache.Resource{})

	cache[nodeID] = &ncache
}

// DeleteNodeCache ...
func (cache Cache) DeleteNodeCache(nodeID string) {
	delete(cache, nodeID)
}

// GetNodeCache ...
func (cache Cache) GetNodeCache(nodeID string) *xds_cache.Snapshot {
	return cache[nodeID]
}

// SetResource ...
func (cache Cache) SetResource(nodeID, name string, rtype xds_cache.ResponseType, value xds_cache.Resource) {
	cache[nodeID].Resources[rtype].Items[name] = value
}

// GetResource ...
func (cache Cache) GetResource(nodeID, name string, rtype xds_cache.ResponseType) xds_cache.Resource {
	return cache[nodeID].Resources[rtype].Items[name]
}

// DeleteResource ...
func (cache Cache) DeleteResource(nodeID, name string, rtype xds_cache.ResponseType) {
	delete(cache[nodeID].Resources[rtype].Items, name)
}

// ClearResources ...
func (cache Cache) ClearResources(nodeID string, rtype xds_cache.ResponseType) {
	cache[nodeID].Resources[rtype].Items = map[string]xds_cache.Resource{}
}

// SetSnapshot ...
func (cache Cache) SetSnapshot(nodeID string, snapshotCache xds_cache.SnapshotCache) {
	snapshotCache.SetSnapshot(nodeID, *cache[nodeID])
}

// GetCurrentVersion ...
func (cache Cache) GetCurrentVersion(nodeID string) (int, error) {
	version, err := strconv.Atoi(cache[nodeID].Resources[0].Version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// BumpCacheVersion ...
func (cache Cache) BumpCacheVersion(nodeID string) (int, error) {
	version, err := strconv.Atoi(cache[nodeID].Resources[0].Version)
	if err != nil {
		return 0, err
	}
	version++
	sversion := strconv.Itoa(version)
	for i := 0; i < 6; i++ {
		// snap := cache[nodeID]
		cache[nodeID].Resources[i].Version = sversion
	}
	return version, nil
}
