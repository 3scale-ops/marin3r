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
	"reflect"
	"testing"

	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

func TestNewSecret(t *testing.T) {
	type args struct {
		name             string
		privateKey       string
		certificateChain string
	}
	tests := []struct {
		name string
		args args
		want *auth.Secret
	}{
		{
			"Returns a valid Secret response struct",
			args{
				name:             "cert1",
				privateKey:       "xxxx",
				certificateChain: "yyyy",
			},
			&auth.Secret{
				Name: "cert1",
				Type: &auth.Secret_TlsCertificate{
					TlsCertificate: &auth.TlsCertificate{
						PrivateKey: &core.DataSource{
							Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte("xxxx")},
						},
						CertificateChain: &core.DataSource{
							Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte("yyyy")},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSecret(tt.args.name, tt.args.privateKey, tt.args.certificateChain); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}
