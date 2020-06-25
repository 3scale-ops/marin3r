package discoveryservicecertificate

import (
	"reflect"
	"testing"

	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var s *runtime.Scheme = scheme.Scheme

func init() {
	s.AddKnownTypes(operatorv1alpha1.SchemeGroupVersion, &operatorv1alpha1.DiscoveryServiceCertificate{})
}

func TestReconcileDiscoveryServiceCertificate_Reconcile(t *testing.T) {

	t.Run("New DiscoveryServiceCertificate creates SelfSigned certificate", func(t *testing.T) {
		sdcert := &operatorv1alpha1.DiscoveryServiceCertificate{
			ObjectMeta: metav1.ObjectMeta{Name: "sdcert", Namespace: "default"},
			Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
				CommonName: "www.example.com",
				ValidFor:   86400,
				Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
					SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
				},
				SecretRef: corev1.SecretReference{Name: "sdcert", Namespace: "default"},
			},
		}
		r := &ReconcileDiscoveryServiceCertificate{
			client: fake.NewFakeClient(sdcert),
			scheme: s,
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "sdcert",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("ReconcileNodeConfigRevision.Reconcile() error = %v", gotErr)
			return
		}

	})
}

func TestAdd(t *testing.T) {
	type args struct {
		mgr manager.Manager
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Add(tt.args.mgr); (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_newReconciler(t *testing.T) {
	type args struct {
		mgr manager.Manager
	}
	tests := []struct {
		name string
		args args
		want reconcile.Reconciler
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newReconciler(tt.args.mgr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newReconciler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_add(t *testing.T) {
	type args struct {
		mgr manager.Manager
		r   reconcile.Reconciler
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := add(tt.args.mgr, tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
