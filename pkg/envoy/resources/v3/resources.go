package envoy

import (
	envoy "github.com/3scale/marin3r/pkg/envoy"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_service_runtime_v3 "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
)

// Generator returns a strcut that implements the envoy_resources.Generator
// interface for v3 resources
type Generator struct{}

// New returns an empty resource of the given type
func (g Generator) New(rType envoy.Type) envoy.Resource {

	switch rType {
	case envoy.Endpoint:
		return &envoy_config_endpoint_v3.ClusterLoadAssignment{}

	case envoy.Cluster:
		return &envoy_config_cluster_v3.Cluster{}

	case envoy.Route:
		return &envoy_config_route_v3.RouteConfiguration{}

	case envoy.Listener:
		return &envoy_config_listener_v3.Listener{}

	case envoy.Runtime:
		return &envoy_service_runtime_v3.Runtime{}

	case envoy.Secret:
		return &envoy_extensions_transport_sockets_tls_v3.Secret{}

	}

	return nil

}

// NewSecret generates return a new envoy seret given the certificate and key.
func (g Generator) NewSecret(name, privateKey, certificateChain string) envoy.Resource {

	return &envoy_extensions_transport_sockets_tls_v3.Secret{
		Name: name,
		Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
			TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
				PrivateKey: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte(privateKey)},
				},
				CertificateChain: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte(certificateChain)},
				},
			},
		},
	}
}

// NewSecretFromPath returns an envoy secret that uses path sds to get the certificate from
// a path and reload it whenever the certificate files change
func (g Generator) NewSecretFromPath(name, privateKeyPath, certificateChainPath string) envoy.Resource {

	return &envoy_extensions_transport_sockets_tls_v3.Secret{
		Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
			TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
				CertificateChain: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_Filename{
						Filename: certificateChainPath,
					},
				},
				PrivateKey: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_Filename{
						Filename: privateKeyPath,
					}},
			},
		},
	}
}
