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
	"reflect"
	"testing"

	envoy_api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache"
)

func newTestCache() Cache {
	return Cache{
		"node1": &xds_cache.Snapshot{
			Resources: [6]xds_cache.Resources{
				{Version: "789", Items: map[string]xds_cache.Resource{
					"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
				}},
				{Version: "789", Items: map[string]xds_cache.Resource{
					"cluster1": &envoy_api.Cluster{Name: "cluster1"},
					"cluster2": &envoy_api.Cluster{Name: "cluster2"},
					"cluster3": &envoy_api.Cluster{Name: "cluster3"},
				}},
				{Version: "789", Items: map[string]xds_cache.Resource{}},
				{Version: "789", Items: map[string]xds_cache.Resource{}},
				{Version: "789", Items: map[string]xds_cache.Resource{}},
				{Version: "789", Items: map[string]xds_cache.Resource{}},
			},
		},
		"node2": &xds_cache.Snapshot{
			Resources: [6]xds_cache.Resources{
				{Version: "43", Items: map[string]xds_cache.Resource{
					"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
				}},
				{Version: "43", Items: map[string]xds_cache.Resource{
					"cluster1": &envoy_api.Cluster{Name: "cluster1"},
				}},
				{Version: "43", Items: map[string]xds_cache.Resource{}},
				{Version: "43", Items: map[string]xds_cache.Resource{}},
				{Version: "43", Items: map[string]xds_cache.Resource{}},
				{Version: "43", Items: map[string]xds_cache.Resource{}},
			},
		},
	}
}

func TestNewCache(t *testing.T) {
	tests := []struct {
		name string
		want Cache
	}{
		{
			"Bootstraps a new cache",
			Cache{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCache(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCache() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCache_NewNodeCache(t *testing.T) {
	type args struct {
		nodeID string
	}
	type want struct {
		cache Cache
	}
	tests := []struct {
		name  string
		cache Cache
		args  args
		want  want
	}{
		{
			"Booststraps a new node snapshot in the cache",
			newTestCache(),
			args{
				nodeID: "node3",
			},
			want{Cache{
				"node1": &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "789", Items: map[string]xds_cache.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "789", Items: map[string]xds_cache.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
							"cluster2": &envoy_api.Cluster{Name: "cluster2"},
							"cluster3": &envoy_api.Cluster{Name: "cluster3"},
						}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
					},
				},
				"node2": &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
						}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
					},
				},
				"node3": &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "1", Items: map[string]xds_cache.Resource{}},
						{Version: "1", Items: map[string]xds_cache.Resource{}},
						{Version: "1", Items: map[string]xds_cache.Resource{}},
						{Version: "1", Items: map[string]xds_cache.Resource{}},
						{Version: "1", Items: map[string]xds_cache.Resource{}},
						{Version: "1", Items: map[string]xds_cache.Resource{}},
					},
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cache.NewNodeCache(tt.args.nodeID)
			if !reflect.DeepEqual(tt.cache, tt.want.cache) {
				t.Errorf("Cache.NewNodeCache() = %v, want %v", tt.cache, tt.want.cache)
			}
		})
	}
}

func TestCache_DeleteNodeCache(t *testing.T) {
	type args struct {
		nodeID string
	}
	type want struct {
		cache Cache
	}
	tests := []struct {
		name  string
		cache Cache
		args  args
		want  want
	}{
		{
			"Deletes a node's snapshot from the cache",
			newTestCache(),
			args{
				nodeID: "node1",
			},
			want{Cache{
				"node2": &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
						}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
					},
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cache.DeleteNodeCache(tt.args.nodeID)
			if !reflect.DeepEqual(tt.cache, tt.want.cache) {
				t.Errorf("Cache.DeleteNodeCache() = %v, want %v", tt.cache, tt.want.cache)
			}
		})
	}
}

func TestCache_GetNodeCache(t *testing.T) {
	type args struct {
		nodeID string
	}
	type want struct {
		snapshot *xds_cache.Snapshot
	}
	tests := []struct {
		name  string
		cache Cache
		args  args
		want  want
	}{
		{
			"Gets the snapshot for a given node",
			newTestCache(),
			args{nodeID: "node2"},
			want{&xds_cache.Snapshot{
				Resources: [6]xds_cache.Resources{
					{Version: "43", Items: map[string]xds_cache.Resource{
						"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "43", Items: map[string]xds_cache.Resource{
						"cluster1": &envoy_api.Cluster{Name: "cluster1"},
					}},
					{Version: "43", Items: map[string]xds_cache.Resource{}},
					{Version: "43", Items: map[string]xds_cache.Resource{}},
					{Version: "43", Items: map[string]xds_cache.Resource{}},
					{Version: "43", Items: map[string]xds_cache.Resource{}},
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cache.GetNodeCache(tt.args.nodeID); !reflect.DeepEqual(got, tt.want.snapshot) {
				t.Errorf("Cache.GetNodeCache() = %v, want %v", got, tt.want.snapshot)
			}
		})
	}
}

func TestCache_SetResource(t *testing.T) {
	type args struct {
		nodeID string
		name   string
		rtype  xds_cache.ResponseType
		value  xds_cache.Resource
	}
	type want struct {
		cache Cache
	}
	tests := []struct {
		name  string
		cache Cache
		args  args
		want  want
	}{
		{
			"Writes a named resource of a given resource type in a node's cache",
			newTestCache(),
			args{
				nodeID: "node1",
				name:   "listener1",
				rtype:  xds_cache.Listener,
				value:  &envoy_api.Listener{Name: "listener1"},
			},
			want{Cache{
				"node1": &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "789", Items: map[string]xds_cache.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "789", Items: map[string]xds_cache.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
							"cluster2": &envoy_api.Cluster{Name: "cluster2"},
							"cluster3": &envoy_api.Cluster{Name: "cluster3"},
						}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
						{Version: "789", Items: map[string]xds_cache.Resource{
							"listener1": &envoy_api.Listener{Name: "listener1"},
						}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
					},
				},
				"node2": &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
						}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
					},
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cache.SetResource(tt.args.nodeID, tt.args.name, tt.args.rtype, tt.args.value)
			if !reflect.DeepEqual(tt.cache, tt.want.cache) {
				t.Errorf("Cache.DeleteResource() = %v, want %v", tt.cache, tt.want.cache)
			}
		})
	}
}

func TestCache_GetResource(t *testing.T) {
	type args struct {
		nodeID string
		name   string
		rtype  xds_cache.ResponseType
	}
	type want struct {
		resource xds_cache.Resource
	}
	tests := []struct {
		name  string
		cache Cache
		args  args
		want  want
	}{
		{
			"Gets a named resource of a given resource type from a node's cache",
			newTestCache(),
			args{
				nodeID: "node2",
				name:   "endpoint",
				rtype:  xds_cache.Endpoint,
			},
			want{&envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"}},
		},
		{
			"Returns 'nil' if the named resource is not found",
			newTestCache(),
			args{
				nodeID: "node2",
				name:   "xxxx",
				rtype:  xds_cache.Endpoint,
			},
			want{nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cache.GetResource(tt.args.nodeID, tt.args.name, tt.args.rtype); !reflect.DeepEqual(got, tt.want.resource) {
				t.Errorf("Cache.GetResource() = %v, want %v", got, tt.want.resource)
			}
		})
	}
}

func TestCache_DeleteResource(t *testing.T) {
	type args struct {
		nodeID string
		name   string
		rtype  xds_cache.ResponseType
	}
	type want struct {
		cache Cache
	}
	tests := []struct {
		name  string
		cache Cache
		args  args
		want  want
	}{
		{
			"Deletes a named resource of a given resource type from a node's cache",
			newTestCache(),
			args{
				nodeID: "node1",
				name:   "cluster2",
				rtype:  xds_cache.Cluster,
			},
			want{Cache{
				"node1": &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "789", Items: map[string]xds_cache.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "789", Items: map[string]xds_cache.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
							"cluster3": &envoy_api.Cluster{Name: "cluster3"},
						}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
					},
				},
				"node2": &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
						}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
					},
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cache.DeleteResource(tt.args.nodeID, tt.args.name, tt.args.rtype)
			if !reflect.DeepEqual(tt.cache, tt.want.cache) {
				t.Errorf("Cache.DeleteResource() = %v, want %v", tt.cache, tt.want.cache)
			}
		})
	}
}

func TestCache_ClearResources(t *testing.T) {
	type args struct {
		nodeID string
		rtype  xds_cache.ResponseType
	}
	type want struct {
		cache Cache
	}
	tests := []struct {
		name  string
		cache Cache
		args  args
		want  want
	}{
		{
			"Deletes resources of a given resource type from a node's cache, only for the given node",
			newTestCache(),
			args{
				nodeID: "node1",
				rtype:  xds_cache.Cluster,
			},
			want{Cache{
				"node1": &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "789", Items: map[string]xds_cache.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
						{Version: "789", Items: map[string]xds_cache.Resource{}},
					},
				},
				"node2": &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
						}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
						{Version: "43", Items: map[string]xds_cache.Resource{}},
					},
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cache.ClearResources(tt.args.nodeID, tt.args.rtype)
			if !reflect.DeepEqual(tt.cache, tt.want.cache) {
				t.Errorf("Cache.ClearResources() = %v, want %v", tt.cache, tt.want.cache)
			}
		})
	}
}

func TestCache_SetSnapshot(t *testing.T) {
	type args struct {
		nodeID        string
		snapshotCache xds_cache.SnapshotCache
	}
	type want struct {
		snapshots map[string]xds_cache.Snapshot
	}
	tests := []struct {
		name    string
		cache   Cache
		args    args
		want    want
		wantErr bool
	}{
		{
			"Writes a node's snapshot to xds_cache.SnapshotCache",
			newTestCache(),
			args{
				nodeID:        "node1",
				snapshotCache: xds_cache.NewSnapshotCache(true, cache.IDHash{}, nil),
			},
			want{
				map[string]xds_cache.Snapshot{
					"node1": {
						Resources: [6]xds_cache.Resources{
							{Version: "789", Items: map[string]xds_cache.Resource{
								"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
							}},
							{Version: "789", Items: map[string]xds_cache.Resource{
								"cluster1": &envoy_api.Cluster{Name: "cluster1"},
								"cluster2": &envoy_api.Cluster{Name: "cluster2"},
								"cluster3": &envoy_api.Cluster{Name: "cluster3"},
							}},
							{Version: "789", Items: map[string]xds_cache.Resource{}},
							{Version: "789", Items: map[string]xds_cache.Resource{}},
							{Version: "789", Items: map[string]xds_cache.Resource{}},
							{Version: "789", Items: map[string]xds_cache.Resource{}},
						},
					},
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cache.SetSnapshot(tt.args.nodeID, tt.args.snapshotCache)
			got, err := tt.args.snapshotCache.GetSnapshot(tt.args.nodeID)
			if err != nil {
				t.Errorf("Could not get snapshot from xds_cache.SnapshotCache: %s", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want.snapshots[tt.args.nodeID]) {
				t.Errorf("Cache.SetSnapshot() = %v, want %v", got, tt.want.snapshots[tt.args.nodeID])
			}
		})
	}
}

func TestCache_GetCurrentVersion(t *testing.T) {
	type args struct {
		nodeID string
	}

	type want struct {
		version int
	}
	tests := []struct {
		name    string
		cache   Cache
		args    args
		want    want
		wantErr bool
	}{
		{
			"Gets current version for a node's cache",
			newTestCache(),
			args{nodeID: "node1"},
			want{version: 789},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cache.GetCurrentVersion(tt.args.nodeID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cache.GetCurrentVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want.version {
				t.Errorf("Cache.GetCurrentVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCache_BumpCacheVersion(t *testing.T) {
	type args struct {
		nodeID string
	}

	type want struct {
		version int
		cache   Cache
	}
	tests := []struct {
		name    string
		cache   Cache
		args    args
		want    want
		wantErr bool
	}{
		{
			"Bumps up version for all resource types",
			newTestCache(),
			args{nodeID: "node2"},
			want{
				version: 44,
				cache: Cache{
					"node1": &xds_cache.Snapshot{
						Resources: [6]xds_cache.Resources{
							{Version: "789", Items: map[string]xds_cache.Resource{
								"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
							}},
							{Version: "789", Items: map[string]xds_cache.Resource{
								"cluster1": &envoy_api.Cluster{Name: "cluster1"},
								"cluster2": &envoy_api.Cluster{Name: "cluster2"},
								"cluster3": &envoy_api.Cluster{Name: "cluster3"},
							}},
							{Version: "789", Items: map[string]xds_cache.Resource{}},
							{Version: "789", Items: map[string]xds_cache.Resource{}},
							{Version: "789", Items: map[string]xds_cache.Resource{}},
							{Version: "789", Items: map[string]xds_cache.Resource{}},
						},
					},
					"node2": &xds_cache.Snapshot{
						Resources: [6]xds_cache.Resources{
							{Version: "44", Items: map[string]xds_cache.Resource{
								"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
							}},
							{Version: "44", Items: map[string]xds_cache.Resource{
								"cluster1": &envoy_api.Cluster{Name: "cluster1"},
							}},
							{Version: "44", Items: map[string]xds_cache.Resource{}},
							{Version: "44", Items: map[string]xds_cache.Resource{}},
							{Version: "44", Items: map[string]xds_cache.Resource{}},
							{Version: "44", Items: map[string]xds_cache.Resource{}},
						},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVersion, err := tt.cache.BumpCacheVersion(tt.args.nodeID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cache.BumpCacheVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotVersion != tt.want.version {
				t.Errorf("Cache.BumpCacheVersion() = %v, want %v", gotVersion, tt.want.version)
			}

			gotCache := tt.cache
			if !reflect.DeepEqual(gotCache, tt.want.cache) {
				t.Errorf("Cache.BumpCacheVersion() = %v, want %v", gotCache, tt.want.cache)
			}
		})
	}
}
