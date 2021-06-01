package envoy

import (
	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_resources_v2 "github.com/3scale-ops/marin3r/pkg/envoy/resources/v2"
	envoy_resources_v3 "github.com/3scale-ops/marin3r/pkg/envoy/resources/v3"
)

func TypeURL(rType envoy.Type, version envoy.APIVersion) string {
	if version == envoy.APIv2 {
		return envoy_resources_v2.Mappings()[rType]
	}
	return envoy_resources_v3.Mappings()[rType]
}
