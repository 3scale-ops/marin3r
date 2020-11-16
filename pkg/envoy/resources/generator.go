package envoy

import (
	"github.com/3scale/marin3r/pkg/envoy"
	envoy_resources_v2 "github.com/3scale/marin3r/pkg/envoy/resources/v2"
	envoy_resources_v3 "github.com/3scale/marin3r/pkg/envoy/resources/v3"
)

// Generator in an interface with methods to generate
// envoy resource structs
type Generator interface {
	New(rType envoy.Type) envoy.Resource
	NewSecret(string, string, string) envoy.Resource
	NewSecretFromPath(string, string, string) envoy.Resource
}

// NewGenerator returns a generator struct for the given API version
func NewGenerator(version envoy.APIVersion) Generator {
	switch version {
	case envoy.APIv3:
		return envoy_resources_v3.Generator{}
	default:
		return envoy_resources_v2.Generator{}
	}

}
