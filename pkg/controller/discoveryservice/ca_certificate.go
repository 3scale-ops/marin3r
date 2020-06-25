package discoveryservice

import (
	"context"
	"fmt"

	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	// cert-manager
)

// reconcileCA is responsible of keeping the root CA available at all times
func (r *ReconcileDiscoveryService) reconcileCA(ctx context.Context) (reconcile.Result, error) {

	r.logger.V(1).Info("Reconciling CA certificate")
	ca := &operatorv1alpha1.DiscoveryServiceCertificate{}
	err := r.client.Get(ctx, types.NamespacedName{Name: r.getCACertName(), Namespace: r.getNamespace()}, ca)

	if err != nil {
		if errors.IsNotFound(err) {
			ca = r.genCACertObject()
			if err := controllerutil.SetControllerReference(r.ds, ca, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.client.Create(ctx, ca); err != nil {
				return reconcile.Result{}, err
			}
			r.logger.Info("Created CA certificate")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// CA is not currently reconciled after initial creation, so do nothing
	// TODO: filter out updates?

	return reconcile.Result{}, nil
}

func (r *ReconcileDiscoveryService) getCACertName() string {
	return fmt.Sprintf("%s-%s", caCertSecretNamePrefix, r.ds.GetName())
}

func (r *ReconcileDiscoveryService) getCACertCommonName() string {
	return fmt.Sprintf("%s-%s", caCommonName, r.ds.GetName())
}

func (r *ReconcileDiscoveryService) getCACertNamespace() string {
	return r.getNamespace()
}

func (r *ReconcileDiscoveryService) genCACertObject() *operatorv1alpha1.DiscoveryServiceCertificate {

	return &operatorv1alpha1.DiscoveryServiceCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getCACertName(),
			Namespace: r.getCACertNamespace(),
		},
		Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
			CommonName: r.getCACertCommonName(),
			IsCA:       true,
			ValidFor:   caValidFor,
			Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
				SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
			},
			SecretRef: corev1.SecretReference{
				Name:      r.getCACertName(),
				Namespace: r.getCACertNamespace(),
			},
		},
	}
}
