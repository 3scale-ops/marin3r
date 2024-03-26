package envoy

import (
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
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

	case envoy.ScopedRoute:
		return &envoy_config_route_v3.ScopedRouteConfiguration{}

	case envoy.Listener:
		return &envoy_config_listener_v3.Listener{}

	case envoy.Runtime:
		return &envoy_service_runtime_v3.Runtime{}

	case envoy.Secret:
		return &envoy_extensions_transport_sockets_tls_v3.Secret{}

	case envoy.ExtensionConfig:
		return &envoy_config_core_v3.TypedExtensionConfig{}

	}

	return nil

}

// NewTlsCertificateSecret generates a new envoy secret given the certificate and key.
func (g Generator) NewTlsCertificateSecret(name, privateKey, certificateChain string) envoy.Resource {

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

// NewValidationContextSecret generates a new envoy validation context given the certificate and key.
func (g Generator) NewValidationContextSecret(name, certificateChain string) envoy.Resource {

	return &envoy_extensions_transport_sockets_tls_v3.Secret{
		Name: name,
		Type: &envoy_extensions_transport_sockets_tls_v3.Secret_ValidationContext{
			ValidationContext: &envoy_extensions_transport_sockets_tls_v3.CertificateValidationContext{
				TrustedCa: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte(certificateChain)},
				},
			},
		},
	}
}

func (g Generator) NewGenericSecret(name string, value string) envoy.Resource {
	return &envoy_extensions_transport_sockets_tls_v3.Secret{
		Name: name,
		Type: &envoy_extensions_transport_sockets_tls_v3.Secret_GenericSecret{
			GenericSecret: &envoy_extensions_transport_sockets_tls_v3.GenericSecret{
				Secret: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte(value)}},
			},
		},
	}
}

// NewSecretFromPath returns an envoy secret that uses path sds to get the certificate from
// a path and reload it whenever the certificate files change
func (g Generator) NewTlsSecretFromPath(name, certificateChainPath, privateKeyPath string) envoy.Resource {

	return &envoy_extensions_transport_sockets_tls_v3.Secret{
		Name: name,
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

func (g Generator) NewClusterLoadAssignment(clusterName string, hosts ...envoy.UpstreamHost) envoy.Resource {

	return &envoy_config_endpoint_v3.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*envoy_config_endpoint_v3.LocalityLbEndpoints{
			{
				LbEndpoints: func() []*envoy_config_endpoint_v3.LbEndpoint {
					endpoints := make([]*envoy_config_endpoint_v3.LbEndpoint, len(hosts))
					for idx, host := range hosts {
						endpoints[idx] = LbEndpoint(host).(*envoy_config_endpoint_v3.LbEndpoint)
					}
					return endpoints
				}(),
			},
		},
	}
}

func LbEndpoint(host envoy.UpstreamHost) envoy.Resource {
	return &envoy_config_endpoint_v3.LbEndpoint{
		HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
			Endpoint: &envoy_config_endpoint_v3.Endpoint{
				Address: &envoy_config_core_v3.Address{
					Address: &envoy_config_core_v3.Address_SocketAddress{
						SocketAddress: &envoy_config_core_v3.SocketAddress{
							Address: host.IP.String(),
							PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
								PortValue: host.Port,
							},
						},
					},
				},
			},
		},
		HealthStatus: envoy_config_core_v3.HealthStatus(host.Health),
	}
}
