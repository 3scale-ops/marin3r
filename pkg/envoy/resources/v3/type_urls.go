package envoy

import (
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	resource_v3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

// Mappings return a map associating "github.com/3scale-ops/marin3r/pkg/envoy/resources".Type to
// the v3 envoy API type URLs for each resource type
func Mappings() map[envoy.Type]string {
	return map[envoy.Type]string{
		envoy.Listener:        resource_v3.ListenerType,
		envoy.Route:           resource_v3.RouteType,
		envoy.ScopedRoute:     resource_v3.ScopedRouteType,
		envoy.VirtualHost:     resource_v3.VirtualHostType,
		envoy.Cluster:         resource_v3.ClusterType,
		envoy.Endpoint:        resource_v3.EndpointType,
		envoy.Secret:          resource_v3.SecretType,
		envoy.Runtime:         resource_v3.RuntimeType,
		envoy.ExtensionConfig: resource_v3.ExtensionConfigType,
	}
}
