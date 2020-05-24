package secret

import (
	"context"

	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
	"github.com/3scale/marin3r/pkg/envoy"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	secretCertificate = "tls.crt"
	secretPrivateKey  = "tls.key"
)

var log = logf.Log.WithName("controller_secret")

// Add creates a new Secret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSecret{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("secret-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	filter := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			switch o := e.Object.(type) {
			case *corev1.Secret:
				if o.Type == "kubernetes.io/tls" {
					return true
				}
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			switch o := e.ObjectNew.(type) {
			case *corev1.Secret:
				if o.Type == "kubernetes.io/tls" {
					// Ignore updates to resource status in which case metadata.Generation does not change
					return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
				}
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			switch o := e.Object.(type) {
			case *corev1.Secret:
				if o.Type == "kubernetes.io/tls" {
					return true
				}
			}
			return false
		},
	}

	// Watch for changes to primary resource Secret
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{}, filter)
	if err != nil {
		return err
	}

	// // Watch for changes to secondary resource Pods and requeue the owner Secret
	// err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &corev1.Secret{},
	// })
	// if err != nil {
	// 	return err
	// }

	return nil
}

// blank assignment to verify that ReconcileSecret implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSecret{}

// ReconcileSecret reconciles a Secret object
type ReconcileSecret struct {
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Secret object and makes changes based on the state read
// and what is in the Secret.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSecret) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.TODO()
	// Fetch the Secret instance
	secret := &corev1.Secret{}
	err := r.client.Get(ctx, request.NamespacedName, secret)
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

	logger := log.WithValues("Namespace", request.Namespace, "Name", request.Name)
	logger.Info("Reconciling from 'kubernetes.io/tls' Secret")

	// Get the list of NoceConfigCaches and check which of them
	// contain refs to this secret
	list := &cachesv1alpha1.NodeConfigCacheList{}
	err = r.client.List(ctx, list)

	for _, ncc := range list.Items {
		// TODO: Might need to look inside specific revision instead,
		// when revisions are implemented
		for _, secret := range ncc.Spec.Resources.Secrets {
			if secret.Ref.Name == request.Name && secret.Ref.Namespace == request.Namespace {
				logger.Info("Triggered NodeConfigCache reconcile",
					"NodeConfigCache_Name", ncc.ObjectMeta.Name, "NodeConfigCache_Namespace", ncc.ObjectMeta.Namespace)
				version, err := envoy.BumpVersion(ncc.Spec.Version)
				if err != nil {
					return reconcile.Result{}, err
				}
				// patch operation to update Spec.Version in the cache
				patch := client.MergeFrom(ncc.DeepCopy())
				ncc.Spec.Version = version
				r.client.Patch(ctx, &ncc, patch)
			}
		}
	}

	return reconcile.Result{}, nil
}
