package envoy

import (
	"bytes"
	"time"

	"github.com/3scale/marin3r/pkg/envoy"
	envoy_bootstrap_options "github.com/3scale/marin3r/pkg/envoy/bootstrap/options"
	envoy_resources "github.com/3scale/marin3r/pkg/envoy/resources"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	envoy_config_bootstrap_v2 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

// Config is a struct with options and methods to generate an envoy bootstrap config
type Config struct {
	Options envoy_bootstrap_options.ConfigOptions
}

func (c *Config) getAdminAddress() string { return stringOrDefault(c.Options.AdminAddress, "0.0.0.0") }
func (c *Config) getAdminPort() uint32    { return intOrDefault(c.Options.AdminPort, 9001) }
func (c *Config) getAdminAccessLogPath() string {
	return stringOrDefault(c.Options.AdminAccessLogPath, "/dev/null")
}

// GenerateStatic returns the json serialized representation of an envoy
// bootstrap object that can be passed as the configuration file to an envoy proxy
// so it can connect to the discovery service.
func (c *Config) GenerateStatic() (string, error) {

	tlsContext := &envoy_api_v2_auth.UpstreamTlsContext{
		CommonTlsContext: &envoy_api_v2_auth.CommonTlsContext{
			TlsCertificateSdsSecretConfigs: []*envoy_api_v2_auth.SdsSecretConfig{
				{
					SdsConfig: &envoy_api_v2_core.ConfigSource{
						ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Path{
							Path: c.Options.SdsConfigSourcePath,
						},
					},
				},
			},
		},
	}

	serializedTLSContext, err := ptypes.MarshalAny(tlsContext)
	if err != nil {
		return "", err
	}

	cfg := &envoy_config_bootstrap_v2.Bootstrap{
		Admin: &envoy_config_bootstrap_v2.Admin{
			AccessLogPath: c.getAdminAccessLogPath(),
			Address: &envoy_api_v2_core.Address{
				Address: &envoy_api_v2_core.Address_SocketAddress{
					SocketAddress: &envoy_api_v2_core.SocketAddress{
						Address: c.getAdminAddress(),
						PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
							PortValue: c.getAdminPort(),
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
								ClusterName: envoy_bootstrap_options.XdsClusterName,
							},
						},
					},
				},
			},
			CdsConfig: &envoy_api_v2_core.ConfigSource{
				ResourceApiVersion: envoy_api_v2_core.ApiVersion_V2,
				ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
					Ads: &envoy_api_v2_core.AggregatedConfigSource{},
				},
			},
			LdsConfig: &envoy_api_v2_core.ConfigSource{
				ResourceApiVersion: envoy_api_v2_core.ApiVersion_V2,
				ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
					Ads: &envoy_api_v2_core.AggregatedConfigSource{},
				},
			},
		},
		StaticResources: &envoy_config_bootstrap_v2.Bootstrap_StaticResources{
			Clusters: []*envoy_api_v2.Cluster{
				{
					Name:           envoy_bootstrap_options.XdsClusterName,
					ConnectTimeout: ptypes.DurationProto(1 * time.Second),
					ClusterDiscoveryType: &envoy_api_v2.Cluster_Type{
						Type: envoy_api_v2.Cluster_STRICT_DNS,
					},
					Http2ProtocolOptions: &envoy_api_v2_core.Http2ProtocolOptions{},
					LoadAssignment: &envoy_api_v2.ClusterLoadAssignment{
						ClusterName: envoy_bootstrap_options.XdsClusterName,
						Endpoints: []*envoy_api_v2_endpoint.LocalityLbEndpoints{
							{
								LbEndpoints: []*envoy_api_v2_endpoint.LbEndpoint{
									{
										HostIdentifier: &envoy_api_v2_endpoint.LbEndpoint_Endpoint{
											Endpoint: &envoy_api_v2_endpoint.Endpoint{
												Address: &envoy_api_v2_core.Address{
													Address: &envoy_api_v2_core.Address_SocketAddress{
														SocketAddress: &envoy_api_v2_core.SocketAddress{
															Address: c.Options.XdsHost,
															PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
																PortValue: c.Options.XdsPort,
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
							TypedConfig: serializedTLSContext,
						},
					},
				},
			},
		},
		LayeredRuntime: &envoy_config_bootstrap_v2.LayeredRuntime{
			Layers: []*envoy_config_bootstrap_v2.RuntimeLayer{{
				Name: c.Options.RtdsLayerResourceName,
				LayerSpecifier: &envoy_config_bootstrap_v2.RuntimeLayer_RtdsLayer_{
					RtdsLayer: &envoy_config_bootstrap_v2.RuntimeLayer_RtdsLayer{
						Name: c.Options.RtdsLayerResourceName,
						RtdsConfig: &envoy_api_v2_core.ConfigSource{
							ResourceApiVersion: envoy_api_v2_core.ApiVersion_V2,
							ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
								Ads: &envoy_api_v2_core.AggregatedConfigSource{},
							},
						},
					},
				},
			}},
		},
	}

	m := jsonpb.Marshaler{OrigName: true}
	json := bytes.NewBuffer([]byte{})
	err = m.Marshal(json, cfg)
	if err != nil {
		return "", err
	}

	return string(json.Bytes()), nil
}

// GenerateSdsResources generates the envoy static config required for
// filesystem discovery of certificates.
func (c *Config) GenerateSdsResources() (map[string]string, error) {

	generator := envoy_resources.NewGenerator(envoy.APIv2)
	secret := generator.NewSecretFromPath("xds_client_certificate", c.Options.XdsClientCertificatePath, c.Options.XdsClientCertificateKeyPath)

	a, err := ptypes.MarshalAny(secret)
	if err != nil {
		return nil, err
	}
	cfg := &envoy_api_v2.DiscoveryResponse{
		Resources: []*any.Any{a},
	}

	m := jsonpb.Marshaler{OrigName: true}
	json := bytes.NewBuffer([]byte{})
	err = m.Marshal(json, cfg)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		envoy_bootstrap_options.TlsCertificateSdsSecretFileName: string(json.Bytes()),
	}, nil
}

func stringOrDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func intOrDefault(i, def uint32) uint32 {
	if i == 0 {
		return def
	}
	return i
}
