package envoy

import (
	"testing"

	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	xds_cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

func Test_ResourcesEqual(t *testing.T) {
	type args struct {
		a [6]xds_cache.Resources
		b [6]xds_cache.Resources
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Returns true if snapshot resources are equal",
			args: args{
				a: [6]xds_cache.Resources{
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
				},
				b: [6]xds_cache.Resources{
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
				},
			},
			want: true,
		},
		{
			name: "Returns true if snapshot resources are equal, even with different versions",
			args: args{
				a: [6]xds_cache.Resources{
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
				},
				b: [6]xds_cache.Resources{
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
					}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
				},
			},
			want: true,
		},
		{
			name: "Returns false, different number of resources",
			args: args{
				a: [6]xds_cache.Resources{
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
						"cluster2": &envoyapi.Cluster{Name: "cluster2"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
				},
				b: [6]xds_cache.Resources{
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
					}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
				},
			},
			want: false,
		},
		{
			name: "Returns false, different resource name",
			args: args{
				a: [6]xds_cache.Resources{
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
						"cluster2": &envoyapi.Cluster{Name: "cluster2"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
				},
				b: [6]xds_cache.Resources{
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"cluster1":  &envoyapi.Cluster{Name: "cluster1"},
						"different": &envoyapi.Cluster{Name: "cluster2"},
					}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
				},
			},
			want: false,
		},
		{
			name: "Returns false, different proto message",
			args: args{
				a: [6]xds_cache.Resources{
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
						"cluster2": &envoyapi.Cluster{Name: "cluster2"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
				},
				b: [6]xds_cache.Resources{
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
						"cluster2": &envoyapi.Cluster{Name: "different"},
					}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
				},
			},
			want: false,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResourcesEqual(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("resourcesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
