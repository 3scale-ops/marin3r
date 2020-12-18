package internal

import (
	"context"
	"crypto/x509"
	"reflect"
	"testing"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/util/pki"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var s *runtime.Scheme = scheme.Scheme

func init() {
	s.AddKnownTypes(operatorv1alpha1.GroupVersion,
		&operatorv1alpha1.DiscoveryServiceCertificate{},
	)
}

func testIssuerCertificate() []byte {

	return []byte(`
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
`)
}

func testIssuerKey() []byte {
	return []byte(`
-----BEGIN PRIVATE KEY-----
MIHuAgEAMBAGByqGSM49AgEGBSuBBAAjBIHWMIHTAgEBBEIAiCfLMXyPHO4ZWXZ3
jbQUvfxfm9vmktnDE+yZvnNs1p/LCy2mdiS0dMC5S8QWABVOQudPJtkotL6ABXFm
AaDTUDOhgYkDgYYABAAnoOyix0HKwWH5RHGt7zhZ9kDJhSwa/MrmcZc7+1JRj1Xn
3GQ44Anha74SnWXAV3F30/BKWRkv5QOMwmXG61a7oQGdB3dk4HnHp2evOO+UWAQ9
edQNI7nR0zR/jqalVP49oaCFgfMIbGK8EXnaRKdniUQHTvlwnkCl5ZTkjCo8opCa
FA==
-----END PRIVATE KEY-----
`)
}

func testValidCertificate() []byte {
	return []byte(`
-----BEGIN CERTIFICATE-----
MIICwjCCAiSgAwIBAgIQGdGZ18q7NFDRpp9L9S40bzAKBggqhkjOPQQDBDA7MRsw
GQYDVQQKExJtYXJpbjNyLjNzY2FsZS5uZXQxHDAaBgNVBAMTE21hcmluM3ItY2Et
aW5zdGFuY2UwHhcNMjAxMjE4MTIzNjI0WhcNMzAxMjE4MjI0ODI0WjAsMRswGQYD
VQQKExJtYXJpbjNyLjNzY2FsZS5uZXQxDTALBgNVBAMTBHRlc3QwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQCvHkAScfh7EQ3qTDq252uZLeaXTcsf7hfN
dQwcAnCUJhQONA27j91vOWEiKOjp2/ZPbShS5xxXt1Grrc+DJ2DJfL8odJ/XdN5Q
/W84l7UxnpcxlPUofoXIjccssWAEQDGGFwdxJ0WbGw+8KGsZkE9oYdLKp3gXmKPk
1knxdonEUqJy1bRAmJLaI5BR50bWrTQbfgtASRjxx59X7StLJLPBtG6YQCDY+qdh
JnDqNRX3C0O9pH8ujb1J2A7T2sGMZ2H7kt4C15vcIwzUwZ3ASKGQah1YIxC8P+1P
xRh6wsptXLpOHafkkBIecY4xdljfgJG8ujbsOV4efyL1fVqBsejLAgMBAAGjTjBM
MA4GA1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAMBgNVHRMBAf8E
AjAAMBcGA1UdEQQQMA6CDGV4YW1wbGUudGVzdDAKBggqhkjOPQQDBAOBiwAwgYcC
QgFVnME3LBIOwOCU1Pz0XRMdSMjtaIHxtSzCqlLixIJs64m5hNxmelcjWFlUayaC
hFNB+Cmsg4iE/1ttdggIh0AD7wJBSYUrlaIDPzq82PeQLvRkAPlAJlm21MCqV/kI
sgFV8+UDXCmXKZCOucPi7yD4qOsoqUHENDJ3nqOiQoIVOeC0nc8=
-----END CERTIFICATE-----
`)
}

func testExpiredCertificate() []byte {
	return []byte(`
-----BEGIN CERTIFICATE-----
MIICwjCCAiSgAwIBAgIQIOyQL+7YxHIda+FqkHjYqTAKBggqhkjOPQQDBDA7MRsw
GQYDVQQKExJtYXJpbjNyLjNzY2FsZS5uZXQxHDAaBgNVBAMTE21hcmluM3ItY2Et
aW5zdGFuY2UwHhcNMjAxMjE4MTMwMTQ2WhcNMjAxMjE4MTMwMTQ3WjAsMRswGQYD
VQQKExJtYXJpbjNyLjNzY2FsZS5uZXQxDTALBgNVBAMTBHRlc3QwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQC6Lq1+onmVDTc9kbuXS5BgTy98aadxDly8
wpmcssRV0OBvIpUYhh17vcYd5t/a52XJQ3P1vT25hMv62Bhw0gokMjNAM76mBILG
BN7uK443+ob0E7Z5ZVxdzqlyUBpHMdb5XCiIbcf4U0XP5iZLF/njvZZjgDa6halh
c5whh0g7ekkhGtyQZYOI+pfdsQCVs7c0HjilIMi4CZfIwkefs1R/PQ6KZnABGUBs
k817R4aNLw54pCinYTQ4FvlxLR2A2cstR/f+CBHhcjDSUDdE25omHmRRaiIAigfl
UiKPoFYKA+kU4i69I2w4mjWTWZ7Y1qoURCXHDd34vIJyUXzx5iFLAgMBAAGjTjBM
MA4GA1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAMBgNVHRMBAf8E
AjAAMBcGA1UdEQQQMA6CDGV4YW1wbGUudGVzdDAKBggqhkjOPQQDBAOBiwAwgYcC
QXvi1o7Tk/Co7aLyQfyjxWTTmFIRbso5E+vp/v70U+RzQzJjMU9LHjLHPpaJT6hv
wC4IQVoqTFV9o0LMfmrY9tu0AkIBkSPte09kukRVH682WPLoSgMAfou9DxiwcilT
osel1VN6TUilTRCHM7UU5UItRDaEhFwxnGr9ErAmvkQIVTGSURM=
-----END CERTIFICATE-----
`)
}

func TestNewCertificateProvider(t *testing.T) {
	type args struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		scheme *runtime.Scheme
		dsc    *operatorv1alpha1.DiscoveryServiceCertificate
	}
	tests := []struct {
		name string
		args args
		want *CertificateProvider
	}{
		{
			name: "Returns a CertificateProvider",
			args: args{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s),
				scheme: s,
				dsc:    &operatorv1alpha1.DiscoveryServiceCertificate{},
			},
			want: &CertificateProvider{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s),
				scheme: s,
				dsc:    &operatorv1alpha1.DiscoveryServiceCertificate{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCertificateProvider(tt.args.ctx, tt.args.logger, tt.args.client, tt.args.scheme, tt.args.dsc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCertificateProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCertificateProvider_CreateCertificate(t *testing.T) {
	type fields struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		scheme *runtime.Scheme
		dsc    *operatorv1alpha1.DiscoveryServiceCertificate
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Creates a certificate",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s),
				scheme: s,
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					ObjectMeta: metav1.ObjectMeta{Name: "dsc", Namespace: "test"},
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						CommonName: "test",
						ValidFor:   3600,
						Hosts:      []string{"example.test"},
						Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
							SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
						},
						SecretRef: corev1.SecretReference{Name: "secret"},
					}}},
			wantErr: false,
		},
		{
			name: "Fails because Secret already exists",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s, &corev1.
					Secret{ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "test"}}),
				scheme: s,
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					ObjectMeta: metav1.ObjectMeta{Name: "dsc", Namespace: "test"},
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						CommonName: "test",
						ValidFor:   3600,
						Hosts:      []string{"example.test"},
						Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
							SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
						},
						SecretRef: corev1.SecretReference{Name: "secret"},
					}}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &CertificateProvider{
				ctx:    tt.fields.ctx,
				logger: tt.fields.logger,
				client: tt.fields.client,
				scheme: tt.fields.scheme,
				dsc:    tt.fields.dsc,
			}
			if _, _, err := p.CreateCertificate(); (err != nil) != tt.wantErr {
				t.Errorf("Provider.CreateCertificate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCertificateProvider_GetCertificate(t *testing.T) {
	type fields struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		scheme *runtime.Scheme
		dsc    *operatorv1alpha1.DiscoveryServiceCertificate
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		want1   []byte
		wantErr bool
	}{
		{
			name: "Gets a certificate",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s, &corev1.
					Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "test"},
					Data: map[string][]byte{
						tlsCertificateKey: []byte("xxxx"),
						tlsPrivateKeyKey:  []byte("xxxx"),
					}}),
				scheme: s,
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					ObjectMeta: metav1.ObjectMeta{Name: "dsc", Namespace: "test"},
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						CommonName: "test",
						ValidFor:   3600,
						Hosts:      []string{"example.test"},
						Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
							SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
						},
						SecretRef: corev1.SecretReference{Name: "secret"},
					}}},
			want:    []byte("xxxx"),
			want1:   []byte("xxxx"),
			wantErr: false,
		},
		{
			name: "Returns an error, Secret not found",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s),
				scheme: s,
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					ObjectMeta: metav1.ObjectMeta{Name: "dsc", Namespace: "test"},
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						CommonName: "test",
						ValidFor:   3600,
						Hosts:      []string{"example.test"},
						Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
							SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
						},
						SecretRef: corev1.SecretReference{Name: "secret"},
					}}},
			want:    nil,
			want1:   nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &CertificateProvider{
				ctx:    tt.fields.ctx,
				logger: tt.fields.logger,
				client: tt.fields.client,
				scheme: tt.fields.scheme,
				dsc:    tt.fields.dsc,
			}
			got, got1, err := p.GetCertificate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.GetCertificate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Provider.GetCertificate() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Provider.GetCertificate() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestCertificateProvider_UpdateCertificate(t *testing.T) {
	type fields struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		scheme *runtime.Scheme
		dsc    *operatorv1alpha1.DiscoveryServiceCertificate
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Updates a certificate",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s, &corev1.
					Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "test"},
					Data: map[string][]byte{
						tlsCertificateKey: []byte("xxxx"),
						tlsPrivateKeyKey:  []byte("xxxx"),
					}}),
				scheme: s,
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					ObjectMeta: metav1.ObjectMeta{Name: "dsc", Namespace: "test"},
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						CommonName: "test",
						ValidFor:   3600,
						Hosts:      []string{"example.test"},
						Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
							SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
						},
						SecretRef: corev1.SecretReference{Name: "secret"},
					}}},
			wantErr: false,
		},
		{
			name: "Fails because Secret does not exists",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s),
				scheme: s,
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					ObjectMeta: metav1.ObjectMeta{Name: "dsc", Namespace: "test"},
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						CommonName: "test",
						ValidFor:   3600,
						Hosts:      []string{"example.test"},
						Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
							SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
						},
						SecretRef: corev1.SecretReference{Name: "secret"},
					}}},
			wantErr: true,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &CertificateProvider{
				ctx:    tt.fields.ctx,
				logger: tt.fields.logger,
				client: tt.fields.client,
				scheme: tt.fields.scheme,
				dsc:    tt.fields.dsc,
			}
			_, _, err := p.UpdateCertificate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.UpdateCertificate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestCertificateProvider_VerifyCertificate(t *testing.T) {
	type fields struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		scheme *runtime.Scheme
		dsc    *operatorv1alpha1.DiscoveryServiceCertificate
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Verify returns error=nil",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "issuer", Namespace: "test"},
						Data: map[string][]byte{
							tlsCertificateKey: testIssuerCertificate(),
							tlsPrivateKeyKey:  testIssuerKey(),
						}},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "test"},
						Data: map[string][]byte{
							tlsCertificateKey: testValidCertificate(),
							tlsPrivateKeyKey:  []byte("xxxx"),
						}},
				),
				scheme: s,
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					ObjectMeta: metav1.ObjectMeta{Name: "dsc", Namespace: "test"},
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
							CASigned: &operatorv1alpha1.CASignedConfig{
								SecretRef: corev1.SecretReference{Name: "issuer", Namespace: "test"},
							},
						},
						SecretRef: corev1.SecretReference{Name: "secret"},
					}}},
			wantErr: false,
		},
		{
			name: "Verify returns an error (expired certificate)",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "issuer", Namespace: "test"},
						Data: map[string][]byte{
							tlsCertificateKey: testIssuerCertificate(),
							tlsPrivateKeyKey:  testIssuerKey(),
						}},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "test"},
						Data: map[string][]byte{
							tlsCertificateKey: testExpiredCertificate(),
							tlsPrivateKeyKey:  []byte("xxxx"),
						}},
				),
				scheme: s,
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					ObjectMeta: metav1.ObjectMeta{Name: "dsc", Namespace: "test"},
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
							CASigned: &operatorv1alpha1.CASignedConfig{
								SecretRef: corev1.SecretReference{Name: "issuer", Namespace: "test"},
							},
						},
						SecretRef: corev1.SecretReference{Name: "secret"},
					}}},
			wantErr: true,
		},
		{
			name: "Verify returns an error (secret not found)",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "issuer", Namespace: "test"},
						Data: map[string][]byte{
							tlsCertificateKey: testIssuerCertificate(),
							tlsPrivateKeyKey:  testIssuerKey(),
						}},
				),
				scheme: s,
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					ObjectMeta: metav1.ObjectMeta{Name: "dsc", Namespace: "test"},
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
							CASigned: &operatorv1alpha1.CASignedConfig{
								SecretRef: corev1.SecretReference{Name: "issuer", Namespace: "test"},
							},
						},
						SecretRef: corev1.SecretReference{Name: "secret"},
					}}},
			wantErr: true,
		},
		{
			name: "Verify returns an error (issuer not found)",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "test"},
						Data: map[string][]byte{
							tlsCertificateKey: testExpiredCertificate(),
							tlsPrivateKeyKey:  []byte("xxxx"),
						}},
				),
				scheme: s,
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					ObjectMeta: metav1.ObjectMeta{Name: "dsc", Namespace: "test"},
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
							CASigned: &operatorv1alpha1.CASignedConfig{
								SecretRef: corev1.SecretReference{Name: "issuer", Namespace: "test"},
							},
						},
						SecretRef: corev1.SecretReference{Name: "secret"},
					}}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := &CertificateProvider{
				ctx:    tt.fields.ctx,
				logger: tt.fields.logger,
				client: tt.fields.client,
				scheme: tt.fields.scheme,
				dsc:    tt.fields.dsc,
			}
			if err := cp.VerifyCertificate(); (err != nil) != tt.wantErr {
				t.Errorf("CertificateProvider.VerifyCertificate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCertificateProvider_getIssuerCertificate(t *testing.T) {
	type fields struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		scheme *runtime.Scheme
		dsc    *operatorv1alpha1.DiscoveryServiceCertificate
	}
	tests := []struct {
		name    string
		fields  fields
		want    *x509.Certificate
		want1   interface{}
		wantErr bool
	}{
		{
			name: "Returns the issuer certificate and key",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s, &corev1.
					Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "issuer", Namespace: "test"},
					Type:       corev1.SecretTypeTLS,
					Data: map[string][]byte{
						tlsCertificateKey: testIssuerCertificate(),
						tlsPrivateKeyKey:  testIssuerKey(),
					},
				}),
				scheme: s,
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					ObjectMeta: metav1.ObjectMeta{Name: "dsc", Namespace: "test"},
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						CommonName: "test",
						ValidFor:   3600,
						Hosts:      []string{"example.test"},
						Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
							CASigned: &operatorv1alpha1.CASignedConfig{
								SecretRef: corev1.SecretReference{Name: "issuer", Namespace: "test"},
							},
						},
						SecretRef: corev1.SecretReference{Name: "secret"},
					}}},
			want: func() *x509.Certificate {
				cert, _ := pki.LoadX509Certificate(testIssuerCertificate())
				return cert
			}(),
			want1: func() interface{} {
				signer, _ := pki.DecodePrivateKeyBytes(testIssuerKey())
				return signer
			}(),
			wantErr: false,
		},
		{
			name: "Returns an error if secret holding the issuer cannot be retrieved",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s),
				scheme: s,
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					ObjectMeta: metav1.ObjectMeta{Name: "dsc", Namespace: "test"},
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						CommonName: "test",
						ValidFor:   3600,
						Hosts:      []string{"example.test"},
						Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
							CASigned: &operatorv1alpha1.CASignedConfig{
								SecretRef: corev1.SecretReference{Name: "issuer", Namespace: "test"},
							},
						},
						SecretRef: corev1.SecretReference{Name: "secret"},
					}}},
			want:    nil,
			want1:   nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &CertificateProvider{
				ctx:    tt.fields.ctx,
				logger: tt.fields.logger,
				client: tt.fields.client,
				scheme: tt.fields.scheme,
				dsc:    tt.fields.dsc,
			}
			got, got1, err := p.getIssuerCertificate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.getIssuerCertificate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Provider.getIssuerCertificate() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Provider.getIssuerCertificate() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestCertificateProvider_genSecret(t *testing.T) {
	type fields struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		scheme *runtime.Scheme
		dsc    *operatorv1alpha1.DiscoveryServiceCertificate
	}
	type args struct {
		issuerCert *x509.Certificate
		issuerKey  interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Generates a Secret with a valid 'kubernetes.io/tls' Secret with a valid certificate",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s),
				scheme: s,
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						CommonName: "test",
						ValidFor:   3600,
						Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
							SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
						},
						SecretRef: v1.SecretReference{Name: "secret"},
					},
				},
			},
			args:    args{issuerCert: nil, issuerKey: nil},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &CertificateProvider{
				ctx:    tt.fields.ctx,
				logger: tt.fields.logger,
				client: tt.fields.client,
				scheme: tt.fields.scheme,
				dsc:    tt.fields.dsc,
			}
			secret, err := p.genSecret(tt.args.issuerCert, tt.args.issuerKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.genSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if secret.GetName() != tt.fields.dsc.Spec.SecretRef.Name ||
				secret.GetNamespace() != tt.fields.dsc.GetNamespace() ||
				secret.Type != corev1.SecretTypeTLS {
				t.Errorf("Provider.genSecret() generated secret is not correct = %v", secret)
				return
			}

			cert, err := pki.LoadX509Certificate(secret.Data[tlsCertificateKey])
			if err != nil {
				t.Errorf("Provider.genSecret() error loading generated certificate = %v", err)
				return
			}
			if err := pki.Verify(cert, cert); err != nil {
				t.Errorf("Provider.genSecret() certificate is not valid = %v", err)
				return
			}
		})
	}
}
