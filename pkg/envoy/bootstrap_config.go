package envoy

import (
	"bytes"
	"fmt"
	"time"

	"github.com/3scale/marin3r/pkg/webhook"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	envoy_config_bootstrap_v2 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	resource_v2 "github.com/envoyproxy/go-control-plane/pkg/resource/v2"

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

	tlsContext := &envoy_api_v2_auth.UpstreamTlsContext{
		CommonTlsContext: &envoy_api_v2_auth.CommonTlsContext{
			TlsCertificateSdsSecretConfigs: []*envoy_api_v2_auth.SdsSecretConfig{
				{
					SdsConfig: &envoy_api_v2_core.ConfigSource{
						ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Path{
							Path: fmt.Sprintf("%s/%s", webhook.DefaultEnvoyConfigBasePath, webhook.TlsCertificateSdsSecretFileName),
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

	cfg := &envoy_config_bootstrap_v2.Bootstrap{
		Admin: &envoy_config_bootstrap_v2.Admin{
			AccessLogPath: "/dev/stdout",
			Address: &envoy_api_v2_core.Address{
				Address: &envoy_api_v2_core.Address_SocketAddress{
					SocketAddress: &envoy_api_v2_core.SocketAddress{
						Address: "0.0.0.0",
						PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
							PortValue: 9901,
						},
					},
				},
			},
		},
		DynamicResources: &envoy_config_bootstrap_v2.Bootstrap_DynamicResources{
			AdsConfig: &envoy_api_v2_core.ApiConfigSource{
				ApiType:             envoy_api_v2_core.ApiConfigSource_GRPC,
				TransportApiVersion: envoy_api_v2_core.ApiVersion_V2,
				GrpcServices: []*envoy_api_v2_core.GrpcService{
					{
						TargetSpecifier: &envoy_api_v2_core.GrpcService_EnvoyGrpc_{
							EnvoyGrpc: &envoy_api_v2_core.GrpcService_EnvoyGrpc{
								ClusterName: "ads_cluster",
							},
						},
					},
				},
			},
			CdsConfig: &envoy_api_v2_core.ConfigSource{
				ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
					Ads: &envoy_api_v2_core.AggregatedConfigSource{},
				},
			},
			LdsConfig: &envoy_api_v2_core.ConfigSource{
				ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
					Ads: &envoy_api_v2_core.AggregatedConfigSource{},
				},
			},
		},
		StaticResources: &envoy_config_bootstrap_v2.Bootstrap_StaticResources{
			Clusters: []*envoy_api_v2.Cluster{
				{
					Name:           "ads_cluster",
					ConnectTimeout: ptypes.DurationProto(1 * time.Second),
					ClusterDiscoveryType: &envoy_api_v2.Cluster_Type{
						Type: envoy_api_v2.Cluster_STRICT_DNS,
					},
					Http2ProtocolOptions: &envoy_api_v2_core.Http2ProtocolOptions{},
					LoadAssignment: &envoy_api_v2.ClusterLoadAssignment{
						ClusterName: "ads_cluster",
						Endpoints: []*envoy_api_v2_endpoint.LocalityLbEndpoints{
							{
								LbEndpoints: []*envoy_api_v2_endpoint.LbEndpoint{
									{
										HostIdentifier: &envoy_api_v2_endpoint.LbEndpoint_Endpoint{
											Endpoint: &envoy_api_v2_endpoint.Endpoint{
												Address: &envoy_api_v2_core.Address{
													Address: &envoy_api_v2_core.Address_SocketAddress{
														SocketAddress: &envoy_api_v2_core.SocketAddress{
															Address: host,
															PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
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
					TransportSocket: &envoy_api_v2_core.TransportSocket{
						Name: wellknown.TransportSocketTls,
						ConfigType: &envoy_api_v2_core.TransportSocket_TypedConfig{
							TypedConfig: &any.Any{
								TypeUrl: "type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext",
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

func GenerateTlsCertificateSdsConfig(host string, port uint32) (string, error) {

	tlsCertificate := &envoy_api_v2_auth.Secret{
		Type: &envoy_api_v2_auth.Secret_TlsCertificate{
			TlsCertificate: &envoy_api_v2_auth.TlsCertificate{
				CertificateChain: &envoy_api_v2_core.DataSource{
					Specifier: &envoy_api_v2_core.DataSource_Filename{
						Filename: fmt.Sprintf("%s/%s", webhook.DefaultEnvoyTLSBasePath, "tls.crt"),
					},
				},
				PrivateKey: &envoy_api_v2_core.DataSource{
					Specifier: &envoy_api_v2_core.DataSource_Filename{
						Filename: fmt.Sprintf("%s/%s", webhook.DefaultEnvoyTLSBasePath, "tls.key"),
					}},
			},
		},
	}

	serializedTLSCertificate, err := proto.Marshal(tlsCertificate)
	if err != nil {
		return "", err
	}

	cfg := &envoy_api_v2.DiscoveryResponse{
		Resources: []*any.Any{{
			TypeUrl: resource_v2.SecretType,
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
