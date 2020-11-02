package discoveryservice

import (
	"github.com/3scale/marin3r/pkg/envoy"
	envoy_resources "github.com/3scale/marin3r/pkg/envoy/resources"

	// v3 resources
	envoy_resources_v3 "github.com/3scale/marin3r/pkg/envoy/resources/v3"
	envoy_config_bootstrap_v3 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"

	// v2 resources

	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

type Snapshot struct {
	v3 *cache_v3.Snapshot
}

func NewSnapshot(v3 *cache_v3.Snapshot) Snapshot {
	return Snapshot{v3: v3}
}

func (s Snapshot) Consistent() error {
	return s.v3.Consistent()
}

func (s Snapshot) SetResource(name string, res envoy.Resource) {

	switch o := res.(type) {

	case *envoy_config_endpoint_v3.ClusterLoadAssignment:
		s.v3.Resources[v3CacheResources(envoy_resources.Endpoint)].Items[name] = o

	case *envoy_config_cluster_v3.Cluster:
		s.v3.Resources[v3CacheResources(envoy_resources.Cluster)].Items[name] = o

	case *envoy_config_route_v3.Route:
		s.v3.Resources[v3CacheResources(envoy_resources.Route)].Items[name] = o

	case *envoy_config_listener_v3.Listener:
		s.v3.Resources[v3CacheResources(envoy_resources.Listener)].Items[name] = o

	case *envoy_extensions_transport_sockets_tls_v3.Secret:
		s.v3.Resources[v3CacheResources(envoy_resources.Secret)].Items[name] = o

	case *envoy_config_bootstrap_v3.Runtime:
		s.v3.Resources[v3CacheResources(envoy_resources.Runtime)].Items[name] = o
	}
}

func (s Snapshot) GetResources(rType envoy_resources.Type) map[string]envoy.Resource {

	typeURLs := envoy_resources_v3.Mappings()
	resources := map[string]envoy.Resource{}
	for k, v := range s.v3.GetResources(typeURLs[rType]) {
		resources[k] = v.(envoy.Resource)
	}
	return resources
}

func (s Snapshot) GetVersion(rType envoy_resources.Type) string {
	typeURLs := envoy_resources_v3.Mappings()
	return s.v3.GetVersion(typeURLs[rType])
}

func (s Snapshot) SetVersion(rType envoy_resources.Type, version string) {
	s.v3.Resources[v3CacheResources(rType)].Version = version
}

func v3CacheResources(rType envoy_resources.Type) int {
	types := map[envoy_resources.Type]int{
		envoy_resources.Endpoint: 0,
		envoy_resources.Cluster:  1,
		envoy_resources.Route:    2,
		envoy_resources.Listener: 3,
		envoy_resources.Secret:   4,
		envoy_resources.Runtime:  5,
	}

	return types[rType]
}
