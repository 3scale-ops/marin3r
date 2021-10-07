package envoy

import (
	"testing"

	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
)

func Test_ResourcesEqual(t *testing.T) {
	type args struct {
		a map[string]envoy.Resource
		b map[string]envoy.Resource
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Returns true if snapshot resources are equal",
			args: args{
				a: map[string]envoy.Resource{
					"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoy_config_cluster_v3.Cluster{Name: "cluster1"},
				},
				b: map[string]envoy.Resource{
					"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoy_config_cluster_v3.Cluster{Name: "cluster1"},
				},
			},
			want: true,
		},
		{
			name: "Returns false, different number of resources",
			args: args{
				a: map[string]envoy.Resource{
					"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoy_config_cluster_v3.Cluster{Name: "cluster1"},
					"cluster2": &envoy_config_cluster_v3.Cluster{Name: "cluster2"},
				},
				b: map[string]envoy.Resource{
					"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoy_config_cluster_v3.Cluster{Name: "cluster1"},
				},
			},
			want: false,
		},
		{
			name: "Returns false, different resource name",
			args: args{
				a: map[string]envoy.Resource{
					"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoy_config_cluster_v3.Cluster{Name: "cluster1"},
					"cluster2": &envoy_config_cluster_v3.Cluster{Name: "cluster2"},
				},
				b: map[string]envoy.Resource{
					"endpoint":  &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1":  &envoy_config_cluster_v3.Cluster{Name: "cluster1"},
					"different": &envoy_config_cluster_v3.Cluster{Name: "cluster2"},
				},
			},
			want: false,
		},
		{
			name: "Returns false, different proto message",
			args: args{
				a: map[string]envoy.Resource{
					"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoy_config_cluster_v3.Cluster{Name: "cluster1"},
					"cluster2": &envoy_config_cluster_v3.Cluster{Name: "cluster2"},
				},
				b: map[string]envoy.Resource{
					"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoy_config_cluster_v3.Cluster{Name: "cluster1"},
					"cluster2": &envoy_config_cluster_v3.Cluster{Name: "different"},
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
