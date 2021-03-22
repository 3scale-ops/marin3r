package providers

import (
	"context"
	"crypto/x509"
	"reflect"
	"testing"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/util/pki"
	"github.com/3scale/marin3r/pkg/util/test"
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
							tlsCertificateKey: test.TestIssuerCertificate(),
							tlsPrivateKeyKey:  test.TestIssuerKey(),
						}},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "test"},
						Data: map[string][]byte{
							tlsCertificateKey: test.TestValidCertificate(),
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
							tlsCertificateKey: test.TestIssuerCertificate(),
							tlsPrivateKeyKey:  test.TestIssuerKey(),
						}},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "test"},
						Data: map[string][]byte{
							tlsCertificateKey: test.TestExpiredCertificate(),
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
							tlsCertificateKey: test.TestIssuerCertificate(),
							tlsPrivateKeyKey:  test.TestIssuerKey(),
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
							tlsCertificateKey: test.TestExpiredCertificate(),
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
						tlsCertificateKey: test.TestIssuerCertificate(),
						tlsPrivateKeyKey:  test.TestIssuerKey(),
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
				cert, _ := pki.LoadX509Certificate(test.TestIssuerCertificate())
				return cert
			}(),
			want1: func() interface{} {
				signer, _ := pki.DecodePrivateKeyBytes(test.TestIssuerKey())
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
