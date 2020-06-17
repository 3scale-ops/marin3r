package discoveryservicecertificate

import (
	"context"
	"time"

	controlplanev1alpha1 "github.com/3scale/marin3r/pkg/apis/controlplane/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	// cert-manager
	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
)

const pollingPeriod time.Duration = 10

var log = logf.Log.WithName("controller_discoveryservicecertificate")

// Add creates a new DiscoveryServiceCertificate Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	dc, _ := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	return &ReconcileDiscoveryServiceCertificate{
		client:          mgr.GetClient(),
		scheme:          mgr.GetScheme(),
		discoveryClient: dc,
		// WARNING: this variable is not thread safe, change this
		// if you need support for more than one concurrent worker
		certificateWatch: false,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("DiscoveryServiceCertificate-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource DiscoveryServiceCertificate
	err = c.Watch(&source.Kind{Type: &controlplanev1alpha1.DiscoveryServiceCertificate{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Set up a goroutine to autodetect if required apis are available
	go func() {

		discoverFn := func() {
			rec := r.(*ReconcileDiscoveryServiceCertificate)
			resourceExists, _ := k8sutil.ResourceExists(
				rec.discoveryClient,
				certmanagerv1alpha2.SchemeGroupVersion.String(),
				certmanagerv1alpha2.CertificateKind,
			)
			if resourceExists && !rec.certificateWatch {

				err := c.Watch(&source.Kind{Type: &certmanagerv1alpha2.Certificate{}}, &handler.EnqueueRequestForOwner{
					IsController: true,
					OwnerType:    &controlplanev1alpha1.DiscoveryService{},
				})

				if err != nil {
					log.Error(err, "Failed setting a watch on certmanagerv1alpha2.Certificate type")
				} else {
					// Mark the watch was correctly set
					log.Info("Discovered certmanagerv1alpha2 api, watching type 'Certificate'")
					// WARNING: this is not thread safe
					rec.certificateWatch = true
				}
			}
		}

		ticker := time.NewTicker(pollingPeriod * time.Second)

		discoverFn()
		for range ticker.C {
			discoverFn()
		}
	}()

	return nil
}

// blank assignment to verify that ReconcileDiscoveryServiceCertificate implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDiscoveryServiceCertificate{}

// ReconcileDiscoveryServiceCertificate reconciles a DiscoveryServiceCertificate object
type ReconcileDiscoveryServiceCertificate struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client           client.Client
	scheme           *runtime.Scheme
	discoveryClient  discovery.DiscoveryInterface
	certificateWatch bool
}

// Reconcile reads that state of the cluster for a DiscoveryServiceCertificate object and makes changes based on the state read
// and what is in the DiscoveryServiceCertificate.Spec
func (r *ReconcileDiscoveryServiceCertificate) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling DiscoveryServiceCertificate")
	ctx := context.Background()

	// Fetch the DiscoveryServiceCertificate instance
	dsc := &controlplanev1alpha1.DiscoveryServiceCertificate{}
	err := r.client.Get(context.TODO(), request.NamespacedName, dsc)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if dsc.Spec.Signer.CertManager != nil {
		reqLogger.Info("Reconciling cert-manager certificate")
		if err := r.reconcileCertManagerCertificate(ctx, dsc); err != nil {
			return reconcile.Result{}, err
		}
	} else {
		reqLogger.Info("Reconciling self-signed certificate")
		if err := r.reconcileSelfSignedCertificate(ctx, dsc); err != nil {
			return reconcile.Result{}, err
		}
	}

	// TODO: set status Ready/NotReady

	return reconcile.Result{}, nil
}
