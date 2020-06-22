package discoveryservice

import (
	"context"
	"fmt"

	controlplanev1alpha1 "github.com/3scale/marin3r/pkg/apis/controlplane/v1alpha1"
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

	if r.ds.Spec.Signer.CertManager == nil {
		return reconcile.Result{}, fmt.Errorf("Unsupported signer for DiscoveryService object")
	}

	r.logger.V(1).Info("Reconciling server certificate")

	cert := &controlplanev1alpha1.DiscoveryServiceCertificate{}
	err := r.client.Get(ctx, types.NamespacedName{Name: r.getServerCertName(), Namespace: r.getNamespace()}, cert)

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

func (r *ReconcileDiscoveryService) getServerCertName() string {
	return fmt.Sprintf("%s-%s", serverCertSecretNamePrefix, r.ds.GetName())
}

// TODO: CN is wrong
func (r *ReconcileDiscoveryService) getServerCertCommonName() string {
	return fmt.Sprintf("%s-%s", caCommonName, r.ds.GetName())
}

func (r *ReconcileDiscoveryService) getServerCertObject() *controlplanev1alpha1.DiscoveryServiceCertificate {
	return &controlplanev1alpha1.DiscoveryServiceCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getServerCertName(),
			Namespace: r.getNamespace(),
		},
		Spec: controlplanev1alpha1.DiscoveryServiceCertificateSpec{
			CommonName:          r.getServerCertCommonName(),
			IsServerCertificate: true,
			ValidFor:            serverValidFor,
			Signer: controlplanev1alpha1.DiscoveryServiceCertificateSigner{
				CertManager: &controlplanev1alpha1.CertManagerConfig{
					ClusterIssuer: r.getClusterIssuerName(),
				},
			},
			Hosts: []string{r.getDiscoveryServiceHost()},
			SecretRef: corev1.SecretReference{
				Name:      r.getServerCertName(),
				Namespace: r.getNamespace(),
			},
		},
	}
}
