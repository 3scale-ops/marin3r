package controllers

import (
	"context"
	"fmt"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	// cert-manager
)

// reconcileCA is responsible of keeping the root CA available at all times
func (r *DiscoveryServiceReconciler) reconcileCA(ctx context.Context, log logr.Logger) (reconcile.Result, error) {

	ca := &operatorv1alpha1.DiscoveryServiceCertificate{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: getCACertName(r.ds), Namespace: OwnedObjectNamespace(r.ds)}, ca)

	if err != nil {
		if errors.IsNotFound(err) {
			ca = r.genCACertObject()
			if err := controllerutil.SetControllerReference(r.ds, ca, r.Scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.Client.Create(ctx, ca); err != nil {
				return reconcile.Result{}, err
			}
			log.Info("Created CA certificate")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// CA is not currently reconciled after initial creation, so do nothing
	// TODO: filter out updates?

	return reconcile.Result{}, nil
}

func getCACertName(ds *operatorv1alpha1.DiscoveryService) string {
	return fmt.Sprintf("%s-%s", caCertSecretNamePrefix, ds.GetName())
}

func getCACertCommonName(ds *operatorv1alpha1.DiscoveryService) string {
	return fmt.Sprintf("%s-%s", caCommonName, ds.GetName())
}

func (r *DiscoveryServiceReconciler) genCACertObject() *operatorv1alpha1.DiscoveryServiceCertificate {

	return &operatorv1alpha1.DiscoveryServiceCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getCACertName(r.ds),
			Namespace: OwnedObjectNamespace(r.ds),
			Labels:    Labels(r.ds),
		},
		Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
			CommonName: getCACertCommonName(r.ds),
			IsCA:       pointer.BoolPtr(true),
			ValidFor:   int64(r.ds.GetRootCertificateAuthorityOptions().Duration.Seconds()),
			Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
				SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
			},
			SecretRef: corev1.SecretReference{
				Name:      getCACertName(r.ds),
				Namespace: OwnedObjectNamespace(r.ds),
			},
			CertificateRenewalConfig: &operatorv1alpha1.CertificateRenewalConfig{
				Enabled: false,
			},
		},
	}
}
