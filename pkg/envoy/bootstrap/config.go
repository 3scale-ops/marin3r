package envoy

import (
	"github.com/3scale/marin3r/pkg/envoy"
	envoy_bootstrap_options "github.com/3scale/marin3r/pkg/envoy/bootstrap/options"
	envoy_bootstrap_v2 "github.com/3scale/marin3r/pkg/envoy/bootstrap/v2"
	envoy_bootstrap_v3 "github.com/3scale/marin3r/pkg/envoy/bootstrap/v3"
)

// Config in an interface with methods to generate
// envoy bootstrap configs
type Config interface {
	GenerateStatic() (string, error)
	GenerateSdsResources() (map[string]string, error)
}

// NewConfig returns a Comfig struct for the given API version
func NewConfig(version envoy.APIVersion, opts envoy_bootstrap_options.ConfigOptions) Config {
	switch version {
	case envoy.APIv3:
		return &envoy_bootstrap_v3.Config{
			Options: opts,
		}
	default:
		return &envoy_bootstrap_v2.Config{
			Options: opts,
		}
	}
}
