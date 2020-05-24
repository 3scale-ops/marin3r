package configmap

import (
	"context"

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
	configMapAnnotation = "marin3r.3scale.net/node-id"
	configMapKey        = "config.yaml"
	secretAnnotation    = "cert-manager.io/common-name"
	secretCertificate   = "tls.crt"
	secretPrivateKey    = "tls.key"
)

var log = logf.Log.WithName("controller_configmap")

// Add creates a new ConfigMap Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileConfigMap{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("configmap-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	filter := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// ConfigMap has marin3r annotation
			_, ok := e.Meta.GetAnnotations()[configMapAnnotation]
			return ok
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// ConfigMap has marin3r annotation
			if _, ok := e.MetaNew.GetAnnotations()[configMapAnnotation]; ok {
				// Ignore updates to CR status in which case metadata.Generation does not change
				// return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
				return ok
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// ConfigMap has marin3r annotation
			_, ok := e.Meta.GetAnnotations()[configMapAnnotation]
			return ok
		},
	}

	// Watch for changes to primary resource ConfigMap
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{}, filter)
	if err != nil {
		return err
	}

	// // Watch for changes to secondary resource Pods and requeue the owner ConfigMap
	// err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &corev1.ConfigMap{},
	// })
	// if err != nil {
	// 	return err
	// }

	return nil
}

// blank assignment to verify that ReconcileConfigMap implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileConfigMap{}

// ReconcileConfigMap reconciles a ConfigMap object
type ReconcileConfigMap struct {
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ConfigMap object and makes changes based on the state read
// and what is in the ConfigMap.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileConfigMap) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.TODO()
	// Fetch the ConfigMap instance
	cm := &corev1.ConfigMap{}
	err := r.client.Get(ctx, request.NamespacedName, cm)
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
	nodeID := cm.GetAnnotations()[configMapAnnotation]
	reqLogger := log.WithValues(
		"Namespace", request.Namespace,
		"Name", request.Name,
		"NodeID", nodeID)

	reqLogger.Info("Reconciling from ConfigMap")

	return reconcile.Result{}, nil
}
