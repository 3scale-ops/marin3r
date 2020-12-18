package reconcilers

import (
	"context"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/common"
	"github.com/3scale/marin3r/pkg/reconcilers/operator/discoveryservicecertificate/providers"
	internal_provider "github.com/3scale/marin3r/pkg/reconcilers/operator/discoveryservicecertificate/providers/marin3r"
	"github.com/3scale/marin3r/pkg/util/pki"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CertificateReconciler is a struct with methods to reconcile DiscoveryServiceCertificates
type CertificateReconciler struct {
	ctx    context.Context
	logger logr.Logger
	client client.Client
	scheme *runtime.Scheme
	dsc    *operatorv1alpha1.DiscoveryServiceCertificate
	ready  bool
	hash   string
}

// Ensure the provider implements the CertificateProvider interface
var _ providers.CertificateProvider = &internal_provider.CertificateProvider{}

// NewCertificateReconciler returns a new RevisionReconciler
func NewCertificateReconciler(ctx context.Context, logger logr.Logger, client client.Client,
	s *runtime.Scheme, dsc *operatorv1alpha1.DiscoveryServiceCertificate) CertificateReconciler {

	return CertificateReconciler{ctx, logger, client, s, dsc, false, ""}
}

// IsReady returns true if the certificate is ready after the
// reconcile. Should be invoked only after runnung Reconcile()
func (r *CertificateReconciler) IsReady() bool {
	return r.ready
}

// GetCertificateHash returns true if the hash of the certificate.
// Should be invoked only after runnung Reconcile()
func (r *CertificateReconciler) GetCertificateHash() string {
	return r.hash
}

// Reconcile progresses DiscoveryServiceCertificates revisions to match the desired state.
// It does so by creating/updating/deleting EnvoyConfigRevision API resources.
func (r *CertificateReconciler) Reconcile() (ctrl.Result, error) {

	provider := internal_provider.NewCertificateProvider(r.ctx, r.logger, r.client, r.scheme, r.dsc)

	var err error
	var certBytes []byte

	// Get the certificate
	certBytes, _, err = provider.GetCertificate()
	if err != nil {
		if errors.IsNotFound(err) {
			_, _, err = provider.CreateCertificate()
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	// Verify the certificate is valid
	err = provider.VerifyCertificate()
	if err != nil {
		if pki.IsVerifyError(err) {
			// The certificate is invalid
			r.logger.Info("certificate failed validation", "reason", err.Error())
			r.ready = false
		} else {
			// Some other failure occurred during the verify process
			return ctrl.Result{}, err
		}
	} else {
		r.ready = true
	}

	if r.dsc.GetCertificateRenewalConfig().Enabled {
		// If certificate is not valid or has been marked for renewal, reissue it
		if r.ready == false || r.dsc.Status.Conditions.IsTrueFor(operatorv1alpha1.CertificateNeedsRenewalCondition) {
			certBytes, _, err = provider.UpdateCertificate()
			if err != nil {
				return ctrl.Result{}, err
			}
			r.logger.Info("reissued certificate")
			return ctrl.Result{Requeue: true}, nil
		}
	}

	r.hash = common.Hash(certBytes)

	return ctrl.Result{}, nil
}
