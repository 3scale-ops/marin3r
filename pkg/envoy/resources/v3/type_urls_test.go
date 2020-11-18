package envoy

import (
	"reflect"
	"testing"

	envoy "github.com/3scale/marin3r/pkg/envoy"
)

func TestMappings(t *testing.T) {
	tests := []struct {
		name string
		want map[envoy.Type]string
	}{
		{
			name: "Returns the typeURL to resource types mapping",
			want: map[envoy.Type]string{
				"Listener": "type.googleapis.com/envoy.config.listener.v3.Listener",
				"Route":    "type.googleapis.com/envoy.config.route.v3.RouteConfiguration",
				"Cluster":  "type.googleapis.com/envoy.config.cluster.v3.Cluster",
				"Endpoint": "type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment",
				"Secret":   "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret",
				"Runtime":  "type.googleapis.com/envoy.service.runtime.v3.Runtime",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Mappings(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Mappings() = %v, want %v", got, tt.want)
			}
		})
	}
}
