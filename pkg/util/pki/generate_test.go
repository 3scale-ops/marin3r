package pki

import (
	"crypto/x509"
	"reflect"
	"testing"
	"time"

	"github.com/3scale-ops/marin3r/pkg/util/test"
)

func testIssuerCertificate() *x509.Certificate {
	cert, _ := LoadX509Certificate(test.TestIssuerCertificate())
	return cert
}

func testIssuerKey() interface{} {
	key, _ := DecodePrivateKeyBytes(test.TestIssuerKey())
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
