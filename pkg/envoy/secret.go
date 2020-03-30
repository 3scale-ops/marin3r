package envoy

import (
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

func NewSecret(name, privateKey, certificateChain string) *auth.Secret {

	return &auth.Secret{
		Name: name,
		Type: &auth.Secret_TlsCertificate{
			TlsCertificate: &auth.TlsCertificate{
				PrivateKey: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte(privateKey)},
				},
				CertificateChain: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte(certificateChain)},
				},
			},
		},
	}
}
