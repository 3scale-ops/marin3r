package reconcilers

import (
	"context"
	"reflect"
	"testing"
	"time"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewClientCertificateReconciler(t *testing.T) {
	type args struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		scheme *runtime.Scheme
		eb     *marin3rv1alpha1.EnvoyBootstrap
	}
	tests := []struct {
		name string
		args args
		want ClientCertificateReconciler
	}{
		{
			name: "Returns a reconciler",
			args: args{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClient(),
				scheme: &runtime.Scheme{},
				eb:     &marin3rv1alpha1.EnvoyBootstrap{},
			},
			want: ClientCertificateReconciler{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClient(),
				scheme: &runtime.Scheme{},
				eb:     &marin3rv1alpha1.EnvoyBootstrap{},
			},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClientCertificateReconciler(tt.args.ctx, tt.args.logger, tt.args.client, tt.args.scheme, tt.args.eb); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClientCertificateReconciler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientCertificateReconciler_Reconcile(t *testing.T) {
	var s *runtime.Scheme = scheme.Scheme
	s.AddKnownTypes(marin3rv1alpha1.GroupVersion, &marin3rv1alpha1.EnvoyBootstrap{})
	s.AddKnownTypes(operatorv1alpha1.GroupVersion, &operatorv1alpha1.DiscoveryService{}, &operatorv1alpha1.DiscoveryServiceCertificate{})

	tests := []struct {
		name    string
		r       *ClientCertificateReconciler
		want    ctrl.Result
		wantErr bool
		wantDSC *operatorv1alpha1.DiscoveryServiceCertificate
	}{
		{
			name: "Creates a DiscoveryServiceCertificatetenem",
			r: &ClientCertificateReconciler{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(
					s,
					&operatorv1alpha1.DiscoveryService{
						ObjectMeta: v1.ObjectMeta{Name: "ds", Namespace: "default"},
						Spec: operatorv1alpha1.DiscoveryServiceSpec{
							Image: pointer.StringPtr("xxx"),
							Debug: pointer.BoolPtr(false),
						},
					},
				),
				scheme: s,
				eb: &marin3rv1alpha1.EnvoyBootstrap{
					ObjectMeta: v1.ObjectMeta{Name: "eb", Namespace: "default"},
					Spec: marin3rv1alpha1.EnvoyBootstrapSpec{
						DiscoveryService: "ds",
						ClientCertificate: &marin3rv1alpha1.ClientCertificate{
							Directory:  "/tls",
							SecretName: "client-certificate",
							Duration: metav1.Duration{
								Duration: func() time.Duration { d, _ := time.ParseDuration("1h"); return d }(),
							},
						},
						EnvoyStaticConfig: &marin3rv1alpha1.EnvoyStaticConfig{
							ConfigMapNameV2:       "cm-v2",
							ConfigMapNameV3:       "cm-v3",
							ConfigFile:            "config.json",
							ResourcesDir:          "/resdir",
							RtdsLayerResourceName: "runtime",
							AdminBindAddress:      "127.0.0.1:1000",
							AdminAccessLogPath:    "/dev/null",
						},
					},
				},
			},
			want:    ctrl.Result{},
			wantErr: false,
			wantDSC: &operatorv1alpha1.DiscoveryServiceCertificate{
				ObjectMeta: v1.ObjectMeta{Name: "client-certificate", Namespace: "default"},
				Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
					CommonName: "client-certificate",
					ValidFor:   3600,
					Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
						CASigned: &operatorv1alpha1.CASignedConfig{
							SecretRef: corev1.SecretReference{
								Name:      "marin3r-ca-cert-ds",
								Namespace: "default",
							}},
					},
					SecretRef: corev1.SecretReference{
						Name: "client-certificate",
					},
				},
			},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.Reconcile()
			if (err != nil) != tt.wantErr {
				t.Errorf("ClientCertificateReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientCertificateReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
			gotDSC := &operatorv1alpha1.DiscoveryServiceCertificate{}
			_ = tt.r.client.Get(tt.r.ctx, types.NamespacedName{Name: tt.wantDSC.GetName(), Namespace: tt.wantDSC.GetNamespace()}, gotDSC)
			if !tt.wantErr && !equality.Semantic.DeepEqual(tt.wantDSC.Spec, gotDSC.Spec) {
				t.Errorf("BootstrapConfigReconciler.Reconcile() DiscoveryServiceCertificate.Spec = %v, want %v", gotDSC.Spec, tt.wantDSC.Spec)
				return
			}
		})
	}
}
