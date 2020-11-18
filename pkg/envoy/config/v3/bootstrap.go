package envoy

import (
	"bytes"
	"fmt"
	"time"

	"github.com/3scale/marin3r/pkg/webhooks/podv1mutator"

	envoy_config_bootstrap_v3 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	resource_v3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

// GenerateBootstrapConfig returns the json serialized representation of an envoy
// bootstrap object that can be passed as the configuration file to an envoy proxy
// so it can connect to the discovery service.
func GenerateBootstrapConfig(host string, port uint32) (string, error) {

	tlsContext := &envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext{
		CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{
			TlsCertificateSdsSecretConfigs: []*envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig{
				{
					SdsConfig: &envoy_config_core_v3.ConfigSource{
						ConfigSourceSpecifier: &envoy_config_core_v3.ConfigSource_Path{
							Path: fmt.Sprintf("%s/%s", podv1mutator.DefaultEnvoyConfigBasePath, podv1mutator.TlsCertificateSdsSecretFileName),
						},
					},
				},
			},
		},
	}

	serializedTLSContext, err := proto.Marshal(tlsContext)
	if err != nil {
		return "", err
	}

	cfg := &envoy_config_bootstrap_v3.Bootstrap{
		Admin: &envoy_config_bootstrap_v3.Admin{
			AccessLogPath: "/dev/stdout",
			Address: &envoy_config_core_v3.Address{
				Address: &envoy_config_core_v3.Address_SocketAddress{
					SocketAddress: &envoy_config_core_v3.SocketAddress{
						Address: "0.0.0.0",
						PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
							PortValue: 9901,
						},
					},
				},
			},
		},
		DynamicResources: &envoy_config_bootstrap_v3.Bootstrap_DynamicResources{
			AdsConfig: &envoy_config_core_v3.ApiConfigSource{
				ApiType:             envoy_config_core_v3.ApiConfigSource_GRPC,
				TransportApiVersion: envoy_config_core_v3.ApiVersion_V2,
				GrpcServices: []*envoy_config_core_v3.GrpcService{
					{
						TargetSpecifier: &envoy_config_core_v3.GrpcService_EnvoyGrpc_{
							EnvoyGrpc: &envoy_config_core_v3.GrpcService_EnvoyGrpc{
								ClusterName: "xds_cluster",
							},
						},
					},
				},
			},
			CdsConfig: &envoy_config_core_v3.ConfigSource{
				ConfigSourceSpecifier: &envoy_config_core_v3.ConfigSource_Ads{
					Ads: &envoy_config_core_v3.AggregatedConfigSource{},
				},
			},
			LdsConfig: &envoy_config_core_v3.ConfigSource{
				ConfigSourceSpecifier: &envoy_config_core_v3.ConfigSource_Ads{
					Ads: &envoy_config_core_v3.AggregatedConfigSource{},
				},
			},
		},
		StaticResources: &envoy_config_bootstrap_v3.Bootstrap_StaticResources{
			Clusters: []*envoy_config_cluster_v3.Cluster{
				{
					Name:           "xds_cluster",
					ConnectTimeout: ptypes.DurationProto(1 * time.Second),
					ClusterDiscoveryType: &envoy_config_cluster_v3.Cluster_Type{
						Type: envoy_config_cluster_v3.Cluster_STRICT_DNS,
					},
					Http2ProtocolOptions: &envoy_config_core_v3.Http2ProtocolOptions{},
					LoadAssignment: &envoy_config_endpoint_v3.ClusterLoadAssignment{
						ClusterName: "xds_cluster",
						Endpoints: []*envoy_config_endpoint_v3.LocalityLbEndpoints{
							{
								LbEndpoints: []*envoy_config_endpoint_v3.LbEndpoint{
									{
										HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
											Endpoint: &envoy_config_endpoint_v3.Endpoint{
												Address: &envoy_config_core_v3.Address{
													Address: &envoy_config_core_v3.Address_SocketAddress{
														SocketAddress: &envoy_config_core_v3.SocketAddress{
															Address: host,
															PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
																PortValue: port,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					TransportSocket: &envoy_config_core_v3.TransportSocket{
						Name: wellknown.TransportSocketTls,
						ConfigType: &envoy_config_core_v3.TransportSocket_TypedConfig{
							TypedConfig: &any.Any{
								TypeUrl: "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
								Value:   serializedTLSContext,
							},
						},
					},
				},
			},
		},
	}

	m := jsonpb.Marshaler{}

	json := bytes.NewBuffer([]byte{})
	err = m.Marshal(json, cfg)
	if err != nil {
		return "", err
	}

	return string(json.Bytes()), nil
}

// GenerateTLSCertificateSdsConfig generates the envoy static config required for
// filesystem discovery of certificates.
func GenerateTLSCertificateSdsConfig(host string, port uint32) (string, error) {

	tlsCertificate := &envoy_extensions_transport_sockets_tls_v3.Secret{
		Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
			TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
				CertificateChain: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_Filename{
						Filename: fmt.Sprintf("%s/%s", podv1mutator.DefaultEnvoyTLSBasePath, "tls.crt"),
					},
				},
				PrivateKey: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_Filename{
						Filename: fmt.Sprintf("%s/%s", podv1mutator.DefaultEnvoyTLSBasePath, "tls.key"),
					}},
			},
		},
	}

	serializedTLSCertificate, err := proto.Marshal(tlsCertificate)
	if err != nil {
		return "", err
	}

	cfg := &envoy_service_discovery_v3.DiscoveryResponse{
		Resources: []*any.Any{{
			TypeUrl: resource_v3.SecretType,
			Value:   serializedTLSCertificate,
		}},
	}

	m := jsonpb.Marshaler{}

	json := bytes.NewBuffer([]byte{})
	err = m.Marshal(json, cfg)
	if err != nil {
		return "", err
	}

	return string(json.Bytes()), nil
}
