package discoveryservice

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	// cert-manager
	"github.com/3scale/marin3r/pkg/apis/external"
	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
)

// reconcileSigner is responsible of keeping the signer (certificates emission backend) configuration
// and related resources so they match the config in the ServiceDiscoverySpec
func (r *ReconcileDiscoveryService) reconcileSigner(ctx context.Context) (reconcile.Result, error) {

	if r.ds.Spec.Signer.CertManager != nil {
		result, err := r.reconcileCertManagerSigner(ctx)
		if result.Requeue || err != nil {
			return result, err
		}

	} else {
		return reconcile.Result{}, fmt.Errorf("Unsupported signer for DiscoveryService object")
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileDiscoveryService) reconcileCertManagerSigner(ctx context.Context) (reconcile.Result, error) {

	r.logger.V(1).Info("Reconciling CertManager ClusterIssuer")

	// Validate the cert-manager apis are available
	exists, err := external.HasCertManagerClusterIssuer(r.discoveryClient)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !exists {
		err := fmt.Errorf("certmanagerv1alpha2.ClusterIssuer unavailabe")
		return reconcile.Result{}, err
	}

	signer := &certmanagerv1alpha2.ClusterIssuer{}
	err = r.client.Get(ctx, types.NamespacedName{Name: r.getClusterIssuerName()}, signer)

	if err != nil {

		if errors.IsNotFound(err) {
			signer := r.getClusterIssuerObject()
			if err := controllerutil.SetControllerReference(r.ds, signer, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.client.Create(ctx, signer); err != nil {
				return reconcile.Result{}, err
			}
			r.logger.Info("Created CertManager ClusterIssuer")

		} else {
			return reconcile.Result{}, err
		}
	}

	if err := r.syncCASecret(ctx); err != nil {
		return reconcile.Result{}, err
	}

	// TODO: the ClusterIssuer is currently created but no reconciled afterwards

	return reconcile.Result{}, nil
}

func (r *ReconcileDiscoveryService) getClusterIssuerName() string {
	return fmt.Sprintf("%s-%s", "marin3r", r.ds.GetName())
}

func (r *ReconcileDiscoveryService) getClusterIssuerObject() *certmanagerv1alpha2.ClusterIssuer {
	return &certmanagerv1alpha2.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getClusterIssuerName(),
		},
		Spec: certmanagerv1alpha2.IssuerSpec{
			IssuerConfig: certmanagerv1alpha2.IssuerConfig{
				CA: &certmanagerv1alpha2.CAIssuer{
					SecretName: getCACertName(r.ds),
				},
			},
		},
	}
}

func (r *ReconcileDiscoveryService) syncCASecret(ctx context.Context) error {

	r.logger.V(1).Info("Syncing CA secret")

	// Get the CA secret
	secret := &corev1.Secret{}
	err := r.client.Get(ctx, types.NamespacedName{
		Name:      getCACertName(r.ds),
		Namespace: OwnedObjectNamespace(r.ds),
	}, secret)
	if err != nil {
		return err
	}

	// Get the synced secret
	syncedSecret := &corev1.Secret{}
	err = r.client.Get(ctx, types.NamespacedName{
		Name:      getCACertName(r.ds),
		Namespace: r.ds.Spec.Signer.CertManager.Namespace,
	}, syncedSecret)

	if err != nil {

		if errors.IsNotFound(err) {
			syncedSecret.SetName(getCACertName(r.ds))
			syncedSecret.SetNamespace(r.ds.Spec.Signer.CertManager.Namespace)
			syncedSecret.Data = secret.Data
			if err := controllerutil.SetControllerReference(r.ds, syncedSecret, r.scheme); err != nil {
				return err
			}
			if err := r.client.Create(ctx, syncedSecret); err != nil {
				return err
			}
			r.logger.Info("Syecronized CA secret")

		} else {
			return err
		}
	}

	// Resync if secret data differs
	if !equality.Semantic.DeepEqual(secret.Data, syncedSecret.Data) {
		// TODO: resync
		patch := client.MergeFrom(syncedSecret.DeepCopy())
		syncedSecret.Data = secret.Data
		if err := r.client.Patch(ctx, syncedSecret, patch); err != nil {
			return err
		}
		r.logger.Info("Syecronized CA secret")
	}

	return nil
}
