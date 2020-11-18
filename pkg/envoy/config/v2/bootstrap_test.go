package envoy

import "testing"

func TestGenerateBootstrapConfig(t *testing.T) {
	type args struct {
		host string
		port uint32
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "Generates the boostrap config",
			args:    args{host: "localhost", port: 1000},
			want:    `{"staticResources":{"clusters":[{"name":"xds_cluster","type":"STRICT_DNS","connectTimeout":"1s","loadAssignment":{"clusterName":"xds_cluster","endpoints":[{"lbEndpoints":[{"endpoint":{"address":{"socketAddress":{"address":"localhost","portValue":1000}}}}]}]},"http2ProtocolOptions":{},"transportSocket":{"name":"envoy.transport_sockets.tls","typedConfig":{"@type":"type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext","commonTlsContext":{"tlsCertificateSdsSecretConfigs":[{"sdsConfig":{"path":"/etc/envoy/bootstrap/tls_certificate_sds_secret.yaml"}}]}}}}]},"dynamicResources":{"ldsConfig":{"ads":{}},"cdsConfig":{"ads":{}},"adsConfig":{"apiType":"GRPC","transportApiVersion":"V2","grpcServices":[{"envoyGrpc":{"clusterName":"xds_cluster"}}]}},"admin":{"accessLogPath":"/dev/stdout","address":{"socketAddress":{"address":"0.0.0.0","portValue":9901}}}}`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateBootstrapConfig(tt.args.host, tt.args.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateBootstrapConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GenerateBootstrapConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateTLSCertificateSdsConfig(t *testing.T) {
	type args struct {
		host string
		port uint32
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "Generates the filesystem discovery for client certificate",
			args:    args{},
			want:    `{"resources":[{"@type":"type.googleapis.com/envoy.api.v2.auth.Secret","tlsCertificate":{"certificateChain":{"filename":"/etc/envoy/tls/client/tls.crt"},"privateKey":{"filename":"/etc/envoy/tls/client/tls.key"}}}]}`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateTLSCertificateSdsConfig(tt.args.host, tt.args.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateTLSCertificateSdsConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GenerateTLSCertificateSdsConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
