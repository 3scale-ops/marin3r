package pki

import (
	"crypto/x509"
	"reflect"
	"testing"
	"time"
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

	cert, _ := LoadX509Certificate([]byte(s))
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
	key, _ := DecodePrivateKeyBytes([]byte(s))
	return key
}

func TestGenerateCertificate(t *testing.T) {
	type args struct {
		issuerCert *x509.Certificate
		signerKey  interface{}
		commonName string
		validFor   time.Duration
		isServer   bool
		isCA       bool
		host       []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Generates a self-signed certificate",
			args: args{
				issuerCert: nil,
				signerKey:  nil,
				commonName: "test",
				validFor:   300 * time.Second,
				isServer:   false,
				isCA:       false,
				host:       []string{"example.test"},
			},
			wantErr: false,
		},
		{
			name: "Generates a ca-signed server certificate",
			args: args{
				issuerCert: testIssuerCertificate(),
				signerKey:  testIssuerKey(),
				commonName: "test",
				validFor:   300 * time.Second,
				isServer:   true,
				isCA:       false,
				host:       []string{"example.test"},
			},
			wantErr: false,
		},
		{
			name: "Generates a self-signed CA certificate",
			args: args{
				issuerCert: nil,
				signerKey:  nil,
				commonName: "test",
				validFor:   300 * time.Second,
				isServer:   false,
				isCA:       true,
				host:       []string{"example.test"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := GenerateCertificate(tt.args.issuerCert, tt.args.signerKey, tt.args.commonName, tt.args.validFor, tt.args.isServer, tt.args.isCA, tt.args.host...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateCertificate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			cert, err := LoadX509Certificate(got)
			if err != nil {
				t.Errorf("GenerateCertificate() error trying to load certificate = %v", err)
				return
			}

			if tt.args.issuerCert != nil {
				if err := Verify(cert, tt.args.issuerCert); err != nil {
					t.Errorf("GenerateCertificate() error validating certificate = %v\n\n%s", err, string(got))
					return
				}
			} else {
				if err := Verify(cert, cert); err != nil {
					t.Errorf("GenerateCertificate() error validating certificate = %v\n\n%s", err, string(got))
					return
				}
			}

			if cert.Subject.CommonName != tt.args.commonName {
				t.Errorf("GenerateCertificate() got CommonName = %s, want %s", cert.Subject.CommonName, tt.args.commonName)
				return
			}

			if tt.args.isServer && !reflect.DeepEqual(cert.ExtKeyUsage, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}) {
				t.Errorf("GenerateCertificate() got ExtKeyUsage = %v, want %v", cert.ExtKeyUsage, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth})
				return
			} else if !tt.args.isServer && cert.ExtKeyUsage != nil {
				t.Errorf("GenerateCertificate() got ExtKeyUsage = %v, want %v", cert.ExtKeyUsage, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth})
				return
			}

			if tt.args.isCA && !cert.IsCA {
				t.Errorf("GenerateCertificate() got IsCA = %v, want %v", cert.IsCA, tt.args.isCA)
			} else if !tt.args.isCA && cert.IsCA {
				t.Errorf("GenerateCertificate() got IsCA = %v, want %v", cert.IsCA, tt.args.isCA)
			}
			if !reflect.DeepEqual(cert.DNSNames, tt.args.host) {
				t.Errorf("GenerateCertificate() got Hosts = %v, want %v", cert.DNSNames, tt.args.host)
			}
		})
	}
}

func TestGeneratePrivateKey(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Generates a new private key",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GeneratePrivateKey()
			if (err != nil) != tt.wantErr {
				t.Errorf("GeneratePrivateKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
