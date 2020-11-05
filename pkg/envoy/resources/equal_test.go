package envoy

import (
	"testing"

	"github.com/3scale/marin3r/pkg/envoy"
	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
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
					"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoyapi.Cluster{Name: "cluster1"},
				},
				b: map[string]envoy.Resource{
					"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoyapi.Cluster{Name: "cluster1"},
				},
			},
			want: true,
		},
		{
			name: "Returns false, different number of resources",
			args: args{
				a: map[string]envoy.Resource{
					"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoyapi.Cluster{Name: "cluster1"},
					"cluster2": &envoyapi.Cluster{Name: "cluster2"},
				},
				b: map[string]envoy.Resource{
					"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoyapi.Cluster{Name: "cluster1"},
				},
			},
			want: false,
		},
		{
			name: "Returns false, different resource name",
			args: args{
				a: map[string]envoy.Resource{
					"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoyapi.Cluster{Name: "cluster1"},
					"cluster2": &envoyapi.Cluster{Name: "cluster2"},
				},
				b: map[string]envoy.Resource{
					"endpoint":  &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1":  &envoyapi.Cluster{Name: "cluster1"},
					"different": &envoyapi.Cluster{Name: "cluster2"},
				},
			},
			want: false,
		},
		{
			name: "Returns false, different proto message",
			args: args{
				a: map[string]envoy.Resource{
					"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoyapi.Cluster{Name: "cluster1"},
					"cluster2": &envoyapi.Cluster{Name: "cluster2"},
				},
				b: map[string]envoy.Resource{
					"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					"cluster1": &envoyapi.Cluster{Name: "cluster1"},
					"cluster2": &envoyapi.Cluster{Name: "different"},
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
