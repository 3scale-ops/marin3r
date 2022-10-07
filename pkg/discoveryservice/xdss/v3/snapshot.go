package discoveryservice

import (
	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_resources_v3 "github.com/3scale-ops/marin3r/pkg/envoy/resources/v3"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	"github.com/3scale-ops/marin3r/pkg/util"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_service_runtime_v3 "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

// Snapshot implements "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss".Snapshot for envoy API v3.
type Snapshot struct {
	v3 *cache_v3.Snapshot
}

// NewSnapshot returns a Snapshot object.
func NewSnapshot(v3 *cache_v3.Snapshot) Snapshot {
	return Snapshot{v3: v3}
}

// Consistent check verifies that the dependent resources are exactly listed in the
// snapshot:
// - all EDS resources are listed by name in CDS resources
// - all RDS resources are listed by name in LDS resources
//
// Note that clusters and listeners are requested without name references, so
// Envoy will accept the snapshot list of clusters as-is even if it does not match
// all references found in xDS.
func (s Snapshot) Consistent() error {
	return s.v3.Consistent()
}

// SetResource writes the given v3 resource in the Snapshot object.
func (s Snapshot) SetResource(name string, res envoy.Resource) {
	var rType envoy.Type

	switch o := res.(type) {

	case *envoy_config_endpoint_v3.ClusterLoadAssignment:
		rType = envoy.Endpoint
		s.v3.Resources[v3CacheResources(rType)].Items[name] = cache_types.ResourceWithTTL{Resource: o}

	case *envoy_config_cluster_v3.Cluster:
		rType = envoy.Cluster
		s.v3.Resources[v3CacheResources(rType)].Items[name] = cache_types.ResourceWithTTL{Resource: o}

	case *envoy_config_route_v3.RouteConfiguration:
		rType = envoy.Route
		s.v3.Resources[v3CacheResources(rType)].Items[name] = cache_types.ResourceWithTTL{Resource: o}

	case *envoy_config_route_v3.ScopedRouteConfiguration:
		rType = envoy.ScopedRoute
		s.v3.Resources[v3CacheResources(rType)].Items[name] = cache_types.ResourceWithTTL{Resource: o}

	case *envoy_config_listener_v3.Listener:
		rType = envoy.Listener
		s.v3.Resources[v3CacheResources(rType)].Items[name] = cache_types.ResourceWithTTL{Resource: o}

	case *envoy_extensions_transport_sockets_tls_v3.Secret:
		rType = envoy.Secret
		s.v3.Resources[v3CacheResources(rType)].Items[name] = cache_types.ResourceWithTTL{Resource: o}

	case *envoy_service_runtime_v3.Runtime:
		rType = envoy.Runtime
		s.v3.Resources[v3CacheResources(rType)].Items[name] = cache_types.ResourceWithTTL{Resource: o}
	}

	s.SetVersion(rType, s.recalculateVersion(rType))

}

// GetResources selects snapshot resources by type.
func (s Snapshot) GetResources(rType envoy.Type) map[string]envoy.Resource {

	typeURLs := envoy_resources_v3.Mappings()
	resources := map[string]envoy.Resource{}
	for k, v := range s.v3.GetResources(typeURLs[rType]) {
		resources[k] = v.(envoy.Resource)
	}
	return resources
}

// GetVersion returns the version for a resource type.
func (s Snapshot) GetVersion(rType envoy.Type) string {
	typeURLs := envoy_resources_v3.Mappings()
	return s.v3.GetVersion(typeURLs[rType])
}

// SetVersion sets the version for a resource type.
func (s Snapshot) SetVersion(rType envoy.Type, version string) {
	s.v3.Resources[v3CacheResources(rType)].Version = version
}

func (s Snapshot) recalculateVersion(rType envoy.Type) string {
	resources := map[string]string{}
	encoder := envoy_serializer.NewResourceMarshaller(envoy_serializer.JSON, envoy.APIv3)
	for n, r := range s.v3.Resources[v3CacheResources(rType)].Items {
		j, _ := encoder.Marshal(r.Resource)
		resources[n] = string(j)
	}
	if len(resources) > 0 {
		return util.Hash(resources)
	}
	return ""
}

func v3CacheResources(rType envoy.Type) int {
	types := map[envoy.Type]int{
		envoy.Endpoint:        0,
		envoy.Cluster:         1,
		envoy.Route:           2,
		envoy.ScopedRoute:     3,
		envoy.VirtualHost:     4,
		envoy.Listener:        5,
		envoy.Secret:          6,
		envoy.Runtime:         7,
		envoy.ExtensionConfig: 8,
	}

	return types[rType]
}
