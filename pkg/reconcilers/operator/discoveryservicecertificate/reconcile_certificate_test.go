package reconcilers

import (
	"context"
	"reflect"
	"testing"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	testutil "github.com/3scale/marin3r/pkg/util/test"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
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

const (
	tlsCertificateKey = "tls.crt"
	tlsPrivateKeyKey  = "tls.key"
)

func TestNewCertificateReconciler(t *testing.T) {
	type args struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		s      *runtime.Scheme
		dsc    *operatorv1alpha1.DiscoveryServiceCertificate
	}
	tests := []struct {
		name string
		args args
		want CertificateReconciler
	}{
		{
			name: "Returns a CertificateReconciler",
			args: args{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s),
				s:      s,
				dsc:    &operatorv1alpha1.DiscoveryServiceCertificate{},
			},
			want: CertificateReconciler{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s),
				scheme: s,
				dsc:    &operatorv1alpha1.DiscoveryServiceCertificate{},
				ready:  false,
				hash:   "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCertificateReconciler(tt.args.ctx, tt.args.logger, tt.args.client, tt.args.s, tt.args.dsc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCertificateReconciler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCertificateReconciler_IsReady(t *testing.T) {
	tests := []struct {
		name string
		r    *CertificateReconciler
		want bool
	}{
		{
			name: "Returns r.ready",
			r: &CertificateReconciler{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s),
				scheme: s,
				dsc:    &operatorv1alpha1.DiscoveryServiceCertificate{},
				ready:  true,
			},
			want: true,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.IsReady(); got != tt.want {
				t.Errorf("CertificateReconciler.IsReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCertificateReconciler_GetCertificateHash(t *testing.T) {
	tests := []struct {
		name string
		r    *CertificateReconciler
		want string
	}{
		{
			name: "Returns r.hash",
			r: &CertificateReconciler{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s),
				scheme: s,
				dsc:    &operatorv1alpha1.DiscoveryServiceCertificate{},
				hash:   "xxxx",
			},
			want: "xxxx",
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.GetCertificateHash(); got != tt.want {
				t.Errorf("CertificateReconciler.GetCertificateHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCertificateReconciler_Reconcile(t *testing.T) {
	tests := []struct {
		name        string
		r           *CertificateReconciler
		want        ctrl.Result
		wantErr     bool
		wantIsReady bool
		wantHashSet bool
	}{
		{
			name: "Creates a new self-signed certificate",
			r: &CertificateReconciler{
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
					}},
			},
			want:        ctrl.Result{Requeue: true},
			wantErr:     false,
			wantIsReady: false,
			wantHashSet: false,
		},
		{
			name: "Verifies a self-sifned certificate",
			r: &CertificateReconciler{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "test"},
						Data: map[string][]byte{
							tlsCertificateKey: testutil.TestValidCertificate(),
							tlsPrivateKeyKey:  []byte("xxxx"),
						}},
				),
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
					}},
			},
			want:        ctrl.Result{},
			wantErr:     false,
			wantIsReady: true,
			wantHashSet: true,
		},
		{
			name: "Returns not ready on verify error, no renewal enabled",
			r: &CertificateReconciler{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "test"},
						Data: map[string][]byte{
							tlsCertificateKey: testutil.TestExpiredCertificate(),
							tlsPrivateKeyKey:  []byte("xxxx"),
						}},
				),
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
						SecretRef:                corev1.SecretReference{Name: "secret"},
						CertificateRenewalConfig: &operatorv1alpha1.CertificateRenewalConfig{Enabled: false},
					}},
			},
			want:        ctrl.Result{},
			wantErr:     false,
			wantIsReady: false,
			wantHashSet: true,
		},
		{
			name: "Returns requeue on verify error, renewal enabled (default)",
			r: &CertificateReconciler{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "test"},
						Data: map[string][]byte{
							tlsCertificateKey: testutil.TestExpiredCertificate(),
							tlsPrivateKeyKey:  []byte("xxxx"),
						}},
				),
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
					}},
			},
			want:        ctrl.Result{Requeue: true},
			wantErr:     false,
			wantIsReady: false,
			wantHashSet: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.Reconcile()
			if (err != nil) != tt.wantErr {
				t.Errorf("CertificateReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CertificateReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
			if tt.wantIsReady != tt.r.IsReady() {
				t.Errorf("CertificateReconciler.Reconcile() IsReady = %v, want %v", tt.r.IsReady(), tt.wantIsReady)
			}
			if tt.wantHashSet && tt.r.GetCertificateHash() == "" {
				t.Errorf("CertificateReconciler.Reconcile() Hash = %v, want %v", tt.r.GetCertificateHash(), tt.r.hash)
			} else if !tt.wantHashSet && tt.r.GetCertificateHash() != "" {
				t.Errorf("CertificateReconciler.Reconcile() Hash = %v, want %v", tt.r.GetCertificateHash(), tt.r.hash)
			}
		})
	}
}
