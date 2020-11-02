package envoy

import (
	envoy_resources "github.com/3scale/marin3r/pkg/envoy/resources"
)

func Mappings() map[envoy_resources.Type]string {
	return map[envoy_resources.Type]string{
		envoy_resources.Listener: "type.googleapis.com/envoy.api.v2.Listener",
		envoy_resources.Route:    "type.googleapis.com/envoy.api.v2.RouteConfiguration",
		envoy_resources.Cluster:  "type.googleapis.com/envoy.api.v2.Cluster",
		envoy_resources.Endpoint: "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment",
		envoy_resources.Secret:   "type.googleapis.com/envoy.api.v2.auth.Secret",
		envoy_resources.Runtime:  "type.googleapis.com/envoy.service.discovery.v2.Runtime",
	}
}
