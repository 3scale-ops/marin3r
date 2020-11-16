package envoy

import (
	"github.com/3scale/marin3r/pkg/envoy"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
)

// Generator returns a strcut that implements the envoy_resources.Generator
// interface for v2 resources
type Generator struct{}

// New returns an empty resource of the given type
func (g Generator) New(rType envoy.Type) envoy.Resource {

	switch rType {
	case envoy.Endpoint:
		return &envoy_api_v2.ClusterLoadAssignment{}

	case envoy.Cluster:
		return &envoy_api_v2.Cluster{}

	case envoy.Route:
		return &envoy_api_v2.RouteConfiguration{}

	case envoy.Listener:
		return &envoy_api_v2.Listener{}

	case envoy.Runtime:
		return &envoy_service_discovery_v2.Runtime{}

	case envoy.Secret:
		return &envoy_api_v2_auth.Secret{}

	}

	return nil

}

// NewSecret generates return a new envoy seret given the certificate and key.
func (g Generator) NewSecret(name, privateKey, certificateChain string) envoy.Resource {

	return &envoy_api_v2_auth.Secret{
		Name: name,
		Type: &envoy_api_v2_auth.Secret_TlsCertificate{
			TlsCertificate: &envoy_api_v2_auth.TlsCertificate{
				PrivateKey: &envoy_api_v2_core.DataSource{
					Specifier: &envoy_api_v2_core.DataSource_InlineBytes{InlineBytes: []byte(privateKey)},
				},
				CertificateChain: &envoy_api_v2_core.DataSource{
					Specifier: &envoy_api_v2_core.DataSource_InlineBytes{InlineBytes: []byte(certificateChain)},
				},
			},
		},
	}
}

// NewSecretFromPath returns an envoy secret that uses path sds to get the certificate from
// a path and reload it whenever the certificate files change
func (g Generator) NewSecretFromPath(name, privateKeyPath, certificateChainPath string) envoy.Resource {

	return &envoy_api_v2_auth.Secret{
		Type: &envoy_api_v2_auth.Secret_TlsCertificate{
			TlsCertificate: &envoy_api_v2_auth.TlsCertificate{
				CertificateChain: &envoy_api_v2_core.DataSource{
					Specifier: &envoy_api_v2_core.DataSource_Filename{
						Filename: certificateChainPath,
					},
				},
				PrivateKey: &envoy_api_v2_core.DataSource{
					Specifier: &envoy_api_v2_core.DataSource_Filename{
						Filename: privateKeyPath,
					}},
			},
		},
	}
}
