package controllers

import (
	"context"
	"testing"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var s *runtime.Scheme = scheme.Scheme

func init() {
	s.AddKnownTypes(operatorv1alpha1.GroupVersion,
		&operatorv1alpha1.DiscoveryService{},
		&operatorv1alpha1.DiscoveryServiceList{},
		&operatorv1alpha1.DiscoveryServiceCertificate{},
	)
}

func TestReconcileDiscoveryService_Reconcile(t *testing.T) {

	t.Run("Creates a new DiscoveryService", func(t *testing.T) {
		ds := &operatorv1alpha1.DiscoveryService{
			ObjectMeta: metav1.ObjectMeta{Name: "instance"},
			Spec: operatorv1alpha1.DiscoveryServiceSpec{
				Image:                     "image",
				DiscoveryServiceNamespace: "default",
				EnabledNamespaces:         []string{"default"},
			},
		}

		// Fake namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		}
		// Fake certificates that should be created by
		// the discoveryservicecertificate controller
		serverCert := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "marin3r-server-cert-instance",
				Namespace: "default",
			},
			Type: corev1.SecretTypeTLS,
			Data: map[string][]byte{
				"tls.key": []byte("xxxxx"),
				"tls.crt": []byte("xxxxx"),
			},
		}
		caCert := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "marin3r-ca-cert-instance",
				Namespace: "default",
			},
			Type: corev1.SecretTypeTLS,
			Data: map[string][]byte{
				"tls.key": []byte("xxxxx"),
				"tls.crt": []byte("xxxxx"),
			},
		}

		r := &DiscoveryServiceReconciler{
			Client: fake.NewFakeClient(ns, ds, serverCert, caCert),
			Scheme: s,
			Log:    ctrl.Log.WithName("test"),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "instance",
				Namespace: "",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("ReconcileDiscoveryService.Reconcile() error = %v", gotErr)
			return
		}

		// --------------------------------------------
		// Validate that all objects have been created
		// --------------------------------------------

		// DiscoveryServiceCertificates are created
		dscCA := &operatorv1alpha1.DiscoveryServiceCertificate{}
		if r.Client.Get(context.TODO(), types.NamespacedName{Name: getCACertName(ds), Namespace: "default"}, dscCA) != nil {
			t.Errorf("The DiscoveryServiceCertificate object for the CA certificate is missing: %v", getCACertName(ds))
		}
		dscServer := &operatorv1alpha1.DiscoveryServiceCertificate{}
		if r.Client.Get(context.TODO(), types.NamespacedName{Name: getServerCertName(ds), Namespace: "default"}, dscServer) != nil {
			t.Errorf("The DiscoveryServiceCertificate object for the server certificate is missing: %v", getServerCertName(ds))
		}

		// All discovery service Deployment related objects are created: ServiceAccount, ClusterRole,
		// ClusterRoleBinding and Deployment
		sa := &corev1.ServiceAccount{}
		if r.Client.Get(context.TODO(), types.NamespacedName{Name: OwnedObjectName(ds), Namespace: "default"}, sa) != nil {
			t.Errorf("The ServiceAccount object for the discovery service Deployment is missing: %v", OwnedObjectName(ds))
		}
		cr := &rbacv1.ClusterRole{}
		if r.Client.Get(context.TODO(), types.NamespacedName{Name: OwnedObjectName(ds)}, cr) != nil {
			t.Errorf("The ClusterRole object for the discovery service Deployment is missing: %v", OwnedObjectName(ds))
		}
		crb := &rbacv1.ClusterRoleBinding{}
		if r.Client.Get(context.TODO(), types.NamespacedName{Name: OwnedObjectName(ds)}, crb) != nil {
			t.Errorf("The ClusterRoleBinding object for the discovery service Deployment is missing: %v", OwnedObjectName(ds))
		}
		dep := &appsv1.Deployment{}
		if r.Client.Get(context.TODO(), types.NamespacedName{Name: OwnedObjectName(ds), Namespace: "default"}, dep) != nil {
			t.Errorf("The Deployment object for the discovery service is missing: %v", OwnedObjectName(ds))
		}

		mwc := &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
		if r.Client.Get(context.TODO(), types.NamespacedName{Name: OwnedObjectName(ds)}, mwc) != nil {
			t.Errorf("The MutatingWebhookConfiguration object is missing: %v", OwnedObjectName(ds))
		}

		// TODO: validate enabled namespaces
	})
}
