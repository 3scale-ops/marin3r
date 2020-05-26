package nodeconfigcache

import (
	"testing"

	envoy_api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	xds_cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

func Test_snapshotIsEqual(t *testing.T) {
	type args struct {
		newSnap *xds_cache.Snapshot
		oldSnap *xds_cache.Snapshot
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Returns true if snapshot resources are equal",
			args: args{
				newSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					},
				},
				oldSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					},
				},
			},
			want: true,
		},
		{
			name: "Returns true if snapshot resources are equal, even with different versions",
			args: args{
				newSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					},
				},
				oldSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					},
				},
			},
			want: true,
		},
		{
			name: "Returns false, different number of resources",
			args: args{
				newSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
							"cluster2": &envoy_api.Cluster{Name: "cluster2"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					},
				},
				oldSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					},
				},
			},
			want: false,
		},
		{
			name: "Returns false, different resource name",
			args: args{
				newSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
							"cluster2": &envoy_api.Cluster{Name: "cluster2"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					},
				},
				oldSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"cluster1":  &envoy_api.Cluster{Name: "cluster1"},
							"different": &envoy_api.Cluster{Name: "cluster2"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					},
				},
			},
			want: false,
		},
		{
			name: "Returns false, different proto message",
			args: args{
				newSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
							"cluster2": &envoy_api.Cluster{Name: "cluster2"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					},
				},
				oldSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoy_api.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoy_api.Cluster{Name: "cluster1"},
							"cluster2": &envoy_api.Cluster{Name: "different"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := snapshotIsEqual(tt.args.newSnap, tt.args.oldSnap); got != tt.want {
				t.Errorf("snapshotIsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
