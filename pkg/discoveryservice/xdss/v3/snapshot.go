package discoveryservice

import (
	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_resources_v3 "github.com/3scale-ops/marin3r/pkg/envoy/resources/v3"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	"github.com/3scale-ops/marin3r/pkg/util"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resource_v3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

// Snapshot implements "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss".Snapshot for envoy API v3.
type Snapshot struct {
	v3 *cache_v3.Snapshot
}

// NewSnapshot returns a Snapshot object
func NewSnapshot() Snapshot {

	snap, _ := cache_v3.NewSnapshot("",
		map[resource_v3.Type][]cache_types.Resource{
			resource_v3.EndpointType:        {},
			resource_v3.ClusterType:         {},
			resource_v3.RouteType:           {},
			resource_v3.ScopedRouteType:     {},
			resource_v3.VirtualHostType:     {},
			resource_v3.ListenerType:        {},
			resource_v3.SecretType:          {},
			resource_v3.RuntimeType:         {},
			resource_v3.ExtensionConfigType: {},
		},
	)

	return Snapshot{v3: snap}
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

func (s Snapshot) SetResources(rType envoy.Type, resources []envoy.Resource) xdss.Snapshot {

	items := make([]cache_types.Resource, 0, len(resources))
	for _, r := range resources {
		items = append(items, cache_types.Resource(r))
	}

	cv3resources := cache_v3.NewResources("", items)
	s.v3.Resources[v3CacheResources(rType)] = cv3resources

	s.SetVersion(rType, s.recalculateVersion(rType))

	return s
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
		envoy.Endpoint:        int(cache_v3.GetResponseType(envoy_resources_v3.Mappings()[envoy.Endpoint])),
		envoy.Cluster:         int(cache_v3.GetResponseType(envoy_resources_v3.Mappings()[envoy.Cluster])),
		envoy.Route:           int(cache_v3.GetResponseType(envoy_resources_v3.Mappings()[envoy.Route])),
		envoy.ScopedRoute:     int(cache_v3.GetResponseType(envoy_resources_v3.Mappings()[envoy.ScopedRoute])),
		envoy.VirtualHost:     int(cache_v3.GetResponseType(envoy_resources_v3.Mappings()[envoy.VirtualHost])),
		envoy.Listener:        int(cache_v3.GetResponseType(envoy_resources_v3.Mappings()[envoy.Listener])),
		envoy.Secret:          int(cache_v3.GetResponseType(envoy_resources_v3.Mappings()[envoy.Secret])),
		envoy.Runtime:         int(cache_v3.GetResponseType(envoy_resources_v3.Mappings()[envoy.Runtime])),
		envoy.ExtensionConfig: int(cache_v3.GetResponseType(envoy_resources_v3.Mappings()[envoy.ExtensionConfig])),
	}

	return types[rType]
}
