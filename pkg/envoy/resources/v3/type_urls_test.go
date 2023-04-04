package envoy

import (
	"reflect"
	"testing"

	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
)

func TestMappings(t *testing.T) {
	tests := []struct {
		name string
		want map[envoy.Type]string
	}{
		{
			name: "Returns the typeURL to resource types mapping",
			want: map[envoy.Type]string{
				"listener":        "type.googleapis.com/envoy.config.listener.v3.Listener",
				"route":           "type.googleapis.com/envoy.config.route.v3.RouteConfiguration",
				"scopedRoute":     "type.googleapis.com/envoy.config.route.v3.ScopedRouteConfiguration",
				"cluster":         "type.googleapis.com/envoy.config.cluster.v3.Cluster",
				"endpoint":        "type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment",
				"secret":          "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret",
				"runtime":         "type.googleapis.com/envoy.service.runtime.v3.Runtime",
				"virtualHost":     "type.googleapis.com/envoy.config.route.v3.VirtualHost",
				"extensionConfig": "type.googleapis.com/envoy.config.core.v3.TypedExtensionConfig",
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
