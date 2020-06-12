package discoveryservice

import (
	"context"
	"fmt"
	"time"

	controlplanev1alpha1 "github.com/3scale/marin3r/pkg/apis/controlplane/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_dicoveryservice")

// Add creates a new DiscoveryService Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDiscoveryService{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("discoveryservice-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource DiscoveryService
	err = c.Watch(&source.Kind{Type: &controlplanev1alpha1.DiscoveryService{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for events in Namespaces to ensure the marin3r label is always properly set
	// in the marin3r enabled namespaces
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for Secret resources owned by the DiscoveryService resource
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &controlplanev1alpha1.DiscoveryService{},
	})
	if err != nil {
		return err
	}

	// Watch for ConfigMap resources owned by the DiscoveryService resource
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &controlplanev1alpha1.DiscoveryService{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileDiscoveryService implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDiscoveryService{}

// ReconcileDiscoveryService reconciles a DiscoveryService object
type ReconcileDiscoveryService struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a DiscoveryService object and makes changes based on the state read
// and what is in the DiscoveryService.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDiscoveryService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling DiscoveryService")
	ctx := context.Background()

	// Fetch the DiscoveryService instance
	cpList := &controlplanev1alpha1.DiscoveryServiceList{}
	err := r.client.List(ctx, cpList)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if len(cpList.Items) > 1 {
		err := fmt.Errorf("More than one DiscoveryService object in the cluster, refusing to reconcile")
		reqLogger.Error(err, "Only one marin3r installation per cluster is supported")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, err
	}

	// CertManagerSigner object??
	//        ensureCA: create it if it does not exist
	//        ensure cert-manager CA issuer

	// ensureServerCert
	//   if signerType == CertManagerSigner {
	//		create a cert-manager issued certificate using the CA issuer
	//      or better create a ServiceDiscoveryCertificate object with signer = CertManagerSigner??
	//   }

	// ensureDeployment -> not necessary if we achieve single deployment setup with envoy proxy for mTLS

	// ensureService

	// ensureWebhookConfig - use CA and service to create the service

	// create a SidecarConfig onject per namespace in the ServiceDiscovery objgit ect

	return reconcile.Result{}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *controlplanev1alpha1.DiscoveryService) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
