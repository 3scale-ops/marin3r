package reconcilers

import (
	"context"
	"math"
	"time"

	reconcilerutil "github.com/3scale-ops/basereconciler/util"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/operator/discoveryservicecertificate/providers"
	internal_provider "github.com/3scale-ops/marin3r/pkg/reconcilers/operator/discoveryservicecertificate/providers/marin3r"
	"github.com/3scale-ops/marin3r/pkg/util/clock"
	"github.com/3scale-ops/marin3r/pkg/util/pki"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CertificateReconciler is a struct with methods to reconcile DiscoveryServiceCertificates
type CertificateReconciler struct {
	ctx      context.Context
	logger   logr.Logger
	client   client.Client
	scheme   *runtime.Scheme
	dsc      *operatorv1alpha1.DiscoveryServiceCertificate
	provider providers.CertificateProvider
	clock    clock.Clock

	// Calculated fields
	ready     bool
	hash      string
	notBefore *time.Time
	notAfter  *time.Time
	schedule  *time.Duration
}

// Ensure the provider implements the CertificateProvider interface
var _ providers.CertificateProvider = &internal_provider.CertificateProvider{}

// NewCertificateReconciler returns a new RevisionReconciler
func NewCertificateReconciler(ctx context.Context, logger logr.Logger, client client.Client,
	s *runtime.Scheme, dsc *operatorv1alpha1.DiscoveryServiceCertificate, provider providers.CertificateProvider) CertificateReconciler {

	return CertificateReconciler{ctx, logger, client, s, dsc, provider, clock.Real{}, false, "", nil, nil, nil}
}

// IsReady returns true if the certificate is ready after the
// reconcile. Should be invoked only after running Reconcile()
func (r *CertificateReconciler) IsReady() bool {
	return r.ready
}

// GetCertificateHash returns true if the hash of the certificate.
// Should be invoked only after running Reconcile()
func (r *CertificateReconciler) GetCertificateHash() string {
	return r.hash
}

// NotBefore returns the NotBefore property of the reconciled certificate.
// Should be invoked only after running Reconcile()
func (r *CertificateReconciler) NotBefore() time.Time {

	return *r.notBefore
}

// NotAfter returns the NotAfter property of the reconciled certificate.
// Should be invoked only after running Reconcile()
func (r *CertificateReconciler) NotAfter() time.Time {
	return *r.notAfter
}

// GetSchedule returns a time.Duration value that indicates
// when the reconcile needs to be triggered to renew the
// certificate
func (r *CertificateReconciler) GetSchedule() *time.Duration {
	return r.schedule
}

// Reconcile progresses DiscoveryServiceCertificates revisions to match the desired state.
// It does so by creating/updating/deleting EnvoyConfigRevision API resources.
func (r *CertificateReconciler) Reconcile() (ctrl.Result, error) {

	var err error
	var certBytes []byte

	// Get the certificate
	certBytes, _, err = r.provider.GetCertificate()
	if err != nil {
		if errors.IsNotFound(err) {
			_, _, err = r.provider.CreateCertificate()
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	// Verify the certificate is valid
	err = r.provider.VerifyCertificate()
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

	cert, err := pki.LoadX509Certificate(certBytes)
	if err != nil {
		return ctrl.Result{}, err
	}

	// time to certificate expiration is used to calculate next reconcile schedule
	timeToExpire := cert.NotAfter.Sub(r.clock.Now())
	// total duration of the certificate, used to calculate when to start trying renewal
	duration := cert.NotAfter.Sub(cert.NotBefore)

	if r.dsc.GetCertificateRenewalConfig().Enabled {
		// renew the certificate when 20% or less of certificate's duration has passed
		renewBefore := time.Duration(int64(math.Floor(float64(duration) * 0.20)))

		// If certificate is not valid or is within the renewal window, reissue it
		if r.ready == false || timeToExpire < renewBefore {
			certBytes, _, err = r.provider.UpdateCertificate()
			if err != nil {
				return ctrl.Result{}, err
			}
			r.logger.Info("reissued certificate")
			return ctrl.Result{Requeue: true}, nil
		}

		// schedule next reconcile
		schedule := timeToExpire - renewBefore
		r.schedule = &schedule
		r.logger.Info("scheduled certificate renewal", "time", r.clock.Now().Add(schedule).String())

	} else {
		// schedule nextReconcile when certificate expires to update Ready = false in the status
		if !r.clock.Now().After(cert.NotAfter) {
			schedule := timeToExpire + time.Second
			r.schedule = &schedule
			r.logger.Info("scheduled certificate reconcile", "time", r.clock.Now().Add(schedule).String())
		} else {
			r.schedule = nil
		}
	}

	// store the certificate hash for status reconciliation
	r.hash = reconcilerutil.Hash(certBytes)

	//store certificate validity times for status reconciliation
	r.notBefore = &cert.NotBefore
	r.notAfter = &cert.NotAfter

	return ctrl.Result{}, nil
}
