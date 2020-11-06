package envoy

import (
	envoy_resources "github.com/3scale/marin3r/pkg/envoy/resources"
	resource_v2 "github.com/envoyproxy/go-control-plane/pkg/resource/v2"
)

// Mappings return a map associating "github.com/3scale/marin3r/pkg/envoy/resources".Type to
// the v2 envoy API type URLs for each resource type
func Mappings() map[envoy_resources.Type]string {
	return map[envoy_resources.Type]string{
		envoy_resources.Listener: resource_v2.ListenerType,
		envoy_resources.Route:    resource_v2.RouteType,
		envoy_resources.Cluster:  resource_v2.ClusterType,
		envoy_resources.Endpoint: resource_v2.EndpointType,
		envoy_resources.Secret:   resource_v2.SecretType,
		envoy_resources.Runtime:  resource_v2.RuntimeType,
	}
}
