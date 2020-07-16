package discoveryservicecertificatewatcher

import (
	"context"
	"fmt"
	"math"
	"time"

	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/util/pki"
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	renewalWindow int64 = 300
)

var log = logf.Log.WithName("controller_discoveryservicecertificatewatcher")

// Add creates a new DiscoveryServiceCertificateWatcher Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDiscoveryServiceCertificateWatcher{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("discoveryservicecertificatewatcher-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for primary resource DiscoveryServiceCertificateWatcher
	err = c.Watch(&source.Kind{Type: &operatorv1alpha1.DiscoveryServiceCertificate{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileDiscoveryServiceCertificateWatcher implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDiscoveryServiceCertificateWatcher{}

// ReconcileDiscoveryServiceCertificateWatcher reconciles a DiscoveryServiceCertificateWatcher object
type ReconcileDiscoveryServiceCertificateWatcher struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a DiscoveryServiceCertificateWatcher object and makes changes based on the state read
// and what is in the DiscoveryServiceCertificateWatcher.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDiscoveryServiceCertificateWatcher) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(1).Info("Watching certificate validity")

	// Fetch the DiscoveryServiceCertificateWatcher instance
	dsc := &operatorv1alpha1.DiscoveryServiceCertificate{}
	err := r.client.Get(ctx, request.NamespacedName, dsc)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Only the self-signed and ca-signed certificates have its renewal managed
	// by marin3r. cert-manager already does this for the cert-manager issued ones
	if dsc.Spec.Signer.SelfSigned != nil || dsc.Spec.Signer.CASigned != nil {
		secret := &corev1.Secret{}
		err := r.client.Get(ctx,
			types.NamespacedName{
				Name:      dsc.Spec.SecretRef.Name,
				Namespace: dsc.Spec.SecretRef.Namespace,
			},
			secret)

		if err != nil {
			return reconcile.Result{}, err
		}

		cert, err := pki.LoadX509Certificate(secret.Data["tls.crt"])
		if err != nil {
			return reconcile.Result{}, err
		}

		// renewalWindow is the 20% of the certificate validity window
		renewalWindow := float64(dsc.Spec.ValidFor) * 0.20
		reqLogger.V(1).Info("Debug", "RenewalWindow", renewalWindow)
		reqLogger.V(1).Info("Debug", "TimeToExpiracy", cert.NotAfter.Sub(time.Now()).Seconds())

		if cert.NotAfter.Sub(time.Now()).Seconds() < renewalWindow {
			if !dsc.Status.Conditions.IsTrueFor(operatorv1alpha1.CertificateNeedsRenewalCondition) {
				reqLogger.Info("Certificate needs renewal")

				// add condition
				patch := client.MergeFrom(dsc.DeepCopy())
				dsc.Status.Conditions.SetCondition(status.Condition{
					Type:    operatorv1alpha1.CertificateNeedsRenewalCondition,
					Status:  corev1.ConditionTrue,
					Reason:  status.ConditionReason("CertificateAboutToExpire"),
					Message: fmt.Sprintf("Certificate wil expire in less than %v seconds", renewalWindow),
				})
				if err := r.client.Status().Patch(ctx, dsc, patch); err != nil {
					return reconcile.Result{}, err
				}
			}
		}

		poll, _ := time.ParseDuration(fmt.Sprintf("%ds", int64(math.Floor(float64(dsc.Spec.ValidFor)*0.10))))

		return reconcile.Result{RequeueAfter: poll}, nil
	}

	return reconcile.Result{}, nil
}
