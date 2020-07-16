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

// reconcileServerCertificate is in charge of keeping the DiscoveryService server certificate available as a secret
func (r *ReconcileDiscoveryService) reconcileServerCertificate(ctx context.Context) (reconcile.Result, error) {

	r.logger.V(1).Info("Reconciling server certificate")

	cert := &operatorv1alpha1.DiscoveryServiceCertificate{}
	err := r.client.Get(ctx, types.NamespacedName{Name: getServerCertName(r.ds), Namespace: OwnedObjectNamespace(r.ds)}, cert)

	if err != nil {
		if errors.IsNotFound(err) {
			cert = r.getServerCertObject()
			if err := controllerutil.SetControllerReference(r.ds, cert, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.client.Create(ctx, cert); err != nil {
				return reconcile.Result{}, err
			}
			r.logger.Info("Created server certificate")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Server certificate is not currently reconciled after initial creation, so do nothing
	// TODO: validate if status Ready/NotReady (return requeue on NotReady so we wont progress the
	// deployment of other resources until we have a valid certificate)

	return reconcile.Result{}, nil
}

func getServerCertName(ds *operatorv1alpha1.DiscoveryService) string {
	return fmt.Sprintf("%s-%s", serverCertSecretNamePrefix, ds.GetName())
}

func getServerCertCommonName(ds *operatorv1alpha1.DiscoveryService) string {
	return fmt.Sprintf("%s-%s", serverCommonName, ds.GetName())
}

func (r *ReconcileDiscoveryService) getServerCertObject() *operatorv1alpha1.DiscoveryServiceCertificate {
	return &operatorv1alpha1.DiscoveryServiceCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getServerCertName(r.ds),
			Namespace: OwnedObjectNamespace(r.ds),
			Labels:    Labels(r.ds),
		},
		Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
			CommonName:          getServerCertCommonName(r.ds),
			IsServerCertificate: true,
			ValidFor:            serverCertValidFor,
			Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
				CASigned: &operatorv1alpha1.CASignedConfig{
					SecretRef: corev1.SecretReference{
						Name:      getCACertName(r.ds),
						Namespace: OwnedObjectNamespace(r.ds),
					}},
			},
			Hosts: []string{getDiscoveryServiceHost(r.ds)},
			SecretRef: corev1.SecretReference{
				Name:      getServerCertName(r.ds),
				Namespace: OwnedObjectNamespace(r.ds),
			},
			CertificateRenewalConfig: &operatorv1alpha1.CertificateRenewalConfig{
				Enabled: true,
				Notify: &corev1.ObjectReference{
					Kind:      operatorv1alpha1.DiscoveryServiceKind,
					Name:      r.ds.GetName(),
					Namespace: r.ds.GetNamespace(),
				},
			},
		},
	}
}
