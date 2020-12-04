package controllers

import (
	"crypto/x509"
	"testing"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/util/pki"
	v1 "k8s.io/api/core/v1"
)

func testIssuerCertificate() *x509.Certificate {

	s := `
-----BEGIN CERTIFICATE-----
MIICHzCCAYGgAwIBAgIRAKXy2t/M5W24DvEZdsOhcl0wCgYIKoZIzj0EAwQwOzEb
MBkGA1UEChMSbWFyaW4zci4zc2NhbGUubmV0MRwwGgYDVQQDExNtYXJpbjNyLWNh
LWluc3RhbmNlMB4XDTIwMDcxMjEwMzIwN1oXDTIzMDcxMjExMDUyN1owOzEbMBkG
A1UEChMSbWFyaW4zci4zc2NhbGUubmV0MRwwGgYDVQQDExNtYXJpbjNyLWNhLWlu
c3RhbmNlMIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQAJ6DsosdBysFh+URxre84
WfZAyYUsGvzK5nGXO/tSUY9V59xkOOAJ4Wu+Ep1lwFdxd9PwSlkZL+UDjMJlxutW
u6EBnQd3ZOB5x6dnrzjvlFgEPXnUDSO50dM0f46mpVT+PaGghYHzCGxivBF52kSn
Z4lEB075cJ5ApeWU5IwqPKKQmhSjIzAhMA4GA1UdDwEB/wQEAwICBDAPBgNVHRMB
Af8EBTADAQH/MAoGCCqGSM49BAMEA4GLADCBhwJCAOeRa7SgEDOlzEO2l0RPz0Tp
0AqfXVZOKBHSG6F9KXz4nmiP+9mWh6G/gYa2t+MoooT4xW/EWOdWcAlGnS5Z9Nex
AkEVtLQCSnCDb03gj9v4CLRDcF4TqJiRw8Vt2w7PAVa5QA89MiFhb6w1bY9ANM8x
CeKs2l0JkInwUB+SwpmKdQEGcQ==
-----END CERTIFICATE-----
`

	cert, _ := pki.LoadX509Certificate([]byte(s))
	return cert
}

func testIssuerKey() interface{} {
	s := `
-----BEGIN PRIVATE KEY-----
MIHuAgEAMBAGByqGSM49AgEGBSuBBAAjBIHWMIHTAgEBBEIAiCfLMXyPHO4ZWXZ3
jbQUvfxfm9vmktnDE+yZvnNs1p/LCy2mdiS0dMC5S8QWABVOQudPJtkotL6ABXFm
AaDTUDOhgYkDgYYABAAnoOyix0HKwWH5RHGt7zhZ9kDJhSwa/MrmcZc7+1JRj1Xn
3GQ44Anha74SnWXAV3F30/BKWRkv5QOMwmXG61a7oQGdB3dk4HnHp2evOO+UWAQ9
edQNI7nR0zR/jqalVP49oaCFgfMIbGK8EXnaRKdniUQHTvlwnkCl5ZTkjCo8opCa
FA==
-----END PRIVATE KEY-----
`
	key, _ := pki.DecodePrivateKeyBytes([]byte(s))
	return key
}

func Test_genCASignedCertificateObject(t *testing.T) {
	type args struct {
		cfg        operatorv1alpha1.DiscoveryServiceCertificateSpec
		issuerCert *x509.Certificate
		issuerKey  interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Generates a new certificate signed by the given parent certificate",
			args: args{
				cfg: operatorv1alpha1.DiscoveryServiceCertificateSpec{
					CommonName: "test",
					ValidFor:   3600,
					Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
						CASigned: &operatorv1alpha1.CASignedConfig{
							SecretRef: v1.SecretReference{Name: "ca", Namespace: "default"},
						},
					},
					SecretRef: v1.SecretReference{Name: "test", Namespace: "default"},
				},
				issuerCert: testIssuerCertificate(),
				issuerKey:  testIssuerKey(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := genCASignedCertificateObject(tt.args.cfg, tt.args.issuerCert, tt.args.issuerKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("genCASignedCertificateObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.Data["tls.crt"] == nil || got.Data["tls.key"] == nil {
				t.Errorf("genCASignedCertificateObject() malformed secret: \n%v", got)
				return
			}

		})
	}
}
