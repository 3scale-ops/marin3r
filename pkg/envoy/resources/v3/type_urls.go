package envoy

import (
	envoy_resources "github.com/3scale/marin3r/pkg/envoy/resources"
)

func Mappings() map[envoy_resources.Type]string {
	return map[envoy_resources.Type]string{
		envoy_resources.Listener: "type.googleapis.com/envoy.config.listener.v3.Listener",
		envoy_resources.Route:    "type.googleapis.com/envoy.config.route.v3.RouteConfiguration",
		envoy_resources.Cluster:  "type.googleapis.com/envoy.config.cluster.v3.Cluster",
		envoy_resources.Endpoint: "type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment",
		envoy_resources.Secret:   "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret",
		envoy_resources.Runtime:  "type.googleapis.com/envoy.service.runtime.v3.Runtime",
	}
}
