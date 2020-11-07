// Copyright 2020 rvazquez@redhat.com
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package envoy

import (
	"testing"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/golang/protobuf/proto"
)

func TestSecretGenerator_New(t *testing.T) {
	type args struct {
		name             string
		privateKey       string
		certificateChain string
	}
	tests := []struct {
		name string
		g    Generator
		args args
		want *envoy_extensions_transport_sockets_tls_v3.Secret
	}{
		{
			name: "Return v2 secret",
			g:    Generator{},
			args: args{
				name:             "cert1",
				privateKey:       "xxxx",
				certificateChain: "yyyy",
			},
			want: &envoy_extensions_transport_sockets_tls_v3.Secret{
				Name: "cert1",
				Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
					TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
						PrivateKey: &envoy_config_core_v3.DataSource{
							Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("xxxx")},
						},
						CertificateChain: &envoy_config_core_v3.DataSource{
							Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("yyyy")},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Generator{}
			if got := g.NewSecret(tt.args.name, tt.args.privateKey, tt.args.certificateChain); !proto.Equal(got, tt.want) {
				t.Errorf("SecretGenerator.New() = %v, want %v", got, tt.want)
			}
		})
	}
}
