package envoy

import (
	envoy_resources "github.com/3scale/marin3r/pkg/envoy/resources"
	resource_v3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

// Mappings return a map associating "github.com/3scale/marin3r/pkg/envoy/resources".Type to
// the v3 envoy API type URLs for each resource type
func Mappings() map[envoy_resources.Type]string {
	return map[envoy_resources.Type]string{
		envoy_resources.Listener: resource_v3.ListenerType,
		envoy_resources.Route:    resource_v3.RouteType,
		envoy_resources.Cluster:  resource_v3.ClusterType,
		envoy_resources.Endpoint: resource_v3.EndpointType,
		envoy_resources.Secret:   resource_v3.SecretType,
		envoy_resources.Runtime:  resource_v3.RuntimeType,
	}
}
