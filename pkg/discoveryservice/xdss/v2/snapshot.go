package discoveryservice

import (
	envoy "github.com/3scale/marin3r/pkg/envoy"
	envoy_resources_v2 "github.com/3scale/marin3r/pkg/envoy/resources/v2"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

// Snapshot implements "github.com/3scale/marin3r/pkg/discoveryservice/xdss".Snapshot for envoy API v2.
type Snapshot struct {
	v2 *cache_v2.Snapshot
}

// NewSnapshot returns a Snapshot object.
func NewSnapshot(v2 *cache_v2.Snapshot) Snapshot {
	return Snapshot{v2: v2}
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
	return s.v2.Consistent()
}

// SetResource writes the given v2 resource in the Snapshot object.
func (s Snapshot) SetResource(name string, res envoy.Resource) {

	switch o := res.(type) {

	case *envoy_api_v2.ClusterLoadAssignment:
		s.v2.Resources[v2CacheResources(envoy.Endpoint)].Items[name] = o

	case *envoy_api_v2.Cluster:
		s.v2.Resources[v2CacheResources(envoy.Cluster)].Items[name] = o

	case *envoy_api_v2_route.Route:
		s.v2.Resources[v2CacheResources(envoy.Route)].Items[name] = o

	case *envoy_api_v2.Listener:
		s.v2.Resources[v2CacheResources(envoy.Listener)].Items[name] = o

	case *envoy_api_v2_auth.Secret:
		s.v2.Resources[v2CacheResources(envoy.Secret)].Items[name] = o

	case *envoy_service_discovery_v2.Runtime:
		s.v2.Resources[v2CacheResources(envoy.Runtime)].Items[name] = o

	}
}

// GetResources selects snapshot resources by type.
func (s Snapshot) GetResources(rType envoy.Type) map[string]envoy.Resource {

	typeURLs := envoy_resources_v2.Mappings()
	resources := map[string]envoy.Resource{}
	for k, v := range s.v2.GetResources(typeURLs[rType]) {
		resources[k] = v.(envoy.Resource)
	}
	return resources
}

// GetVersion returns the version for a resource type.
func (s Snapshot) GetVersion(rType envoy.Type) string {
	typeURLs := envoy_resources_v2.Mappings()
	return s.v2.GetVersion(typeURLs[rType])
}

// SetVersion sets the version for a resource type.
func (s Snapshot) SetVersion(rType envoy.Type, version string) {
	s.v2.Resources[v2CacheResources(rType)].Version = version
}

func v2CacheResources(rType envoy.Type) int {
	types := map[envoy.Type]int{
		envoy.Endpoint: 0,
		envoy.Cluster:  1,
		envoy.Route:    2,
		envoy.Listener: 3,
		envoy.Secret:   4,
		envoy.Runtime:  5,
	}

	return types[rType]
}
