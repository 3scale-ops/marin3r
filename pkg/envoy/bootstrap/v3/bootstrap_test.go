package envoy

import (
	"reflect"
	"testing"

	envoy_bootstrap_options "github.com/3scale-ops/marin3r/pkg/envoy/bootstrap/options"
)

func TestConfig_GenerateStatic(t *testing.T) {
	tests := []struct {
		name    string
		c       *Config
		want    string
		wantErr bool
	}{
		{
			name: "Returns a string with the envoy booststrap static configuration",
			c: &Config{
				Options: envoy_bootstrap_options.ConfigOptions{
					NodeID:                      "some-id",
					Cluster:                     "some-cluster",
					XdsHost:                     "localhost",
					XdsPort:                     10000,
					XdsClientCertificatePath:    "/tls.crt",
					XdsClientCertificateKeyPath: "/tls.key",
					SdsConfigSourcePath:         "/sds-config-source.json",
					RtdsLayerResourceName:       "runtime",
					Metadata:                    map[string]string{"key1": "value1", "key2": "value2"},
				},
			},
			want:    `{"node":{"id":"some-id","cluster":"some-cluster","metadata":{"key1":"value1","key2":"value2"}},"static_resources":{"clusters":[{"name":"xds_cluster","type":"STRICT_DNS","connect_timeout":"1s","load_assignment":{"cluster_name":"xds_cluster","endpoints":[{"lb_endpoints":[{"endpoint":{"address":{"socket_address":{"address":"localhost","port_value":10000}}}}]}]},"typed_extension_protocol_options":{"envoy.extensions.upstreams.http.v3.HttpProtocolOptions":{"@type":"type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions","explicit_http_config":{"http2_protocol_options":{}}}},"transport_socket":{"name":"envoy.transport_sockets.tls","typed_config":{"@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext","common_tls_context":{"tls_certificate_sds_secret_configs":[{"name":"xds_client_certificate","sds_config":{"path_config_source":{"path":"/sds-config-source.json"},"resource_api_version":"V3"}}]}}}}]},"dynamic_resources":{"lds_config":{"ads":{},"resource_api_version":"V3"},"cds_config":{"ads":{},"resource_api_version":"V3"},"ads_config":{"api_type":"GRPC","transport_api_version":"V3","grpc_services":[{"envoy_grpc":{"cluster_name":"xds_cluster"}}]}},"layered_runtime":{"layers":[{"name":"runtime","rtds_layer":{"name":"runtime","rtds_config":{"ads":{},"resource_api_version":"V3"}}}]},"admin":{"access_log":[{"name":"envoy.access_loggers.file","typed_config":{"@type":"type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog","path":"/dev/null"}}],"address":{"socket_address":{"address":"0.0.0.0","port_value":9001}}}}`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.GenerateStatic()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.GenerateStatic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Config.GenerateStatic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GenerateSdsResources(t *testing.T) {
	tests := []struct {
		name    string
		c       *Config
		want    map[string]string
		wantErr bool
	}{
		{
			name: "Returns a string with the envoy booststrap static configuration",
			c: &Config{
				Options: envoy_bootstrap_options.ConfigOptions{
					XdsHost:                     "localhost",
					XdsPort:                     10000,
					XdsClientCertificatePath:    "/tls.crt",
					XdsClientCertificateKeyPath: "/tls.key",
					SdsConfigSourcePath:         "/sds-config-source.json",
				},
			},
			want: map[string]string{
				"tls_certificate_sds_secret.json": `{"resources":[{"@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret","name":"xds_client_certificate","tls_certificate":{"certificate_chain":{"filename":"/tls.crt"},"private_key":{"filename":"/tls.key"}}}]}`,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.GenerateSdsResources()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.GeneratSds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Config.GeneratSds() = %v, want %v", got, tt.want)
			}
		})
	}
}
