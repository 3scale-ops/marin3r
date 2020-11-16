package reconcilers

import (
	"context"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ClientCertificateReconciler has methods to reconcile discovery service
// client certificates
type ClientCertificateReconciler struct {
	ctx    context.Context
	logger logr.Logger
	client client.Client
	scheme *runtime.Scheme
	eb     *envoyv1alpha1.EnvoyBootstrap
}

// NewClientCertificateReconciler returns a ClientCertificateReconciler struct
func NewClientCertificateReconciler(ctx context.Context, logger logr.Logger, client client.Client, scheme *runtime.Scheme,
	eb *envoyv1alpha1.EnvoyBootstrap) ClientCertificateReconciler {

	return ClientCertificateReconciler{ctx, logger, client, scheme, eb}
}

// Reconcile keeps a discovery service client certificates in sync with the desired state
func (r *ClientCertificateReconciler) Reconcile() (ctrl.Result, error) {

	// Get the DiscoveryService instance this client want to connect to
	ds := &operatorv1alpha1.DiscoveryService{}
	if err := r.client.Get(r.ctx, types.NamespacedName{Name: r.eb.Spec.DiscoveryService}, ds); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Error(err, "DiscoveryService does not exist", "DiscoveryService", r.eb.Spec.DiscoveryService)
		}
		return ctrl.Result{}, err
	}

	dscName := r.eb.GetName()
	dscNamespace := r.eb.GetNamespace()

	// Get this client's DiscoveryServiceCertificate
	dsc := &operatorv1alpha1.DiscoveryServiceCertificate{}
	if err := r.client.Get(r.ctx, types.NamespacedName{Name: dscName, Namespace: dscNamespace}, dsc); err != nil {
		if errors.IsNotFound(err) {
			dsc = r.genClientCertResource(
				types.NamespacedName{
					Name:      dscName,
					Namespace: dscNamespace,
				},
				types.NamespacedName{
					Name:      ds.GetRootCertificateAuthorityOptions().SecretName,
					Namespace: ds.Spec.DiscoveryServiceNamespace,
				},
			)
			if err := controllerutil.SetControllerReference(r.eb, dsc, r.scheme); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.client.Create(r.ctx, dsc); err != nil {
				return ctrl.Result{}, err
			}
			r.logger.Info("Created discovery service client certificate",
				"Name", dscName, "Namespace", dscNamespace)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// To reconcile certificates if spec.clientCertificate changes we just delete
	// the old DiscoveryServiceCertificate and let the controller create a new one
	// in the next reconcile loop
	if int64(r.eb.Spec.ClientCertificate.Duration.Seconds()) != dsc.Spec.ValidFor ||
		r.eb.Spec.ClientCertificate.SecretName != dsc.Spec.SecretRef.Name {
		// Delete the current DiscoveryServiceCertificate and let it be recreated
		// in the next loop
		if err := r.client.Delete(r.ctx, dsc); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ClientCertificateReconciler) genClientCertResource(certificateKey, signingCertificateKey types.NamespacedName) *operatorv1alpha1.DiscoveryServiceCertificate {
	return &operatorv1alpha1.DiscoveryServiceCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certificateKey.Name,
			Namespace: certificateKey.Namespace,
			// Labels:    Labels(r.ds),
		},
		Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
			CommonName: certificateKey.Name,
			ValidFor:   int64(r.eb.Spec.ClientCertificate.Duration.Seconds()),
			Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
				CASigned: &operatorv1alpha1.CASignedConfig{
					SecretRef: corev1.SecretReference{
						Name:      signingCertificateKey.Name,
						Namespace: signingCertificateKey.Namespace,
					}},
			},
			SecretRef: corev1.SecretReference{
				Name:      r.eb.Spec.ClientCertificate.SecretName,
				Namespace: certificateKey.Namespace,
			},
		},
	}
}
