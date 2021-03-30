package envoy

import (
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	resource_v2 "github.com/envoyproxy/go-control-plane/pkg/resource/v2"
)

// Mappings return a map associating "github.com/3scale-ops/marin3r/pkg/envoy/resources".Type to
// the v2 envoy API type URLs for each resource type
func Mappings() map[envoy.Type]string {
	return map[envoy.Type]string{
		envoy.Listener: resource_v2.ListenerType,
		envoy.Route:    resource_v2.RouteType,
		envoy.Cluster:  resource_v2.ClusterType,
		envoy.Endpoint: resource_v2.EndpointType,
		envoy.Secret:   resource_v2.SecretType,
		envoy.Runtime:  resource_v2.RuntimeType,
	}
}
