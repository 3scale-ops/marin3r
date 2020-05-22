package secret

import (
	"context"
	"fmt"

	"github.com/3scale/marin3r/pkg/cache"
	"github.com/3scale/marin3r/pkg/envoy"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
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
	secretAnnotation  = "cert-manager.io/common-name"
	secretCertificate = "tls.crt"
	secretPrivateKey  = "tls.key"
)

var log = logf.Log.WithName("controller_secret")

// Add creates a new Secret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, cache *xds_cache.SnapshotCache) error {
	return add(mgr, newReconciler(mgr, cache))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c *xds_cache.SnapshotCache) reconcile.Reconciler {
	return &ReconcileSecret{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		cache:    cache.NewCache(),
		adsCache: c,
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
			// Secret has certificate annotation
			_, ok := e.Meta.GetAnnotations()[secretAnnotation]
			return ok
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Secret has certificate annotation
			if _, ok := e.MetaNew.GetAnnotations()[secretAnnotation]; ok {
				// Ignore updates to CR status in which case metadata.Generation does not change
				// return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
				return ok
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Secret has certificate annotation
			_, ok := e.Meta.GetAnnotations()[secretAnnotation]
			return ok
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
	client   client.Client
	scheme   *runtime.Scheme
	cache    cache.Cache
	adsCache *xds_cache.SnapshotCache
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

	cn := secret.GetAnnotations()[secretAnnotation]
	logger := log.WithValues(
		"Namespace", request.Namespace,
		"Name", request.Name,
		"CN", cn)

	logger.Info("Reconciling from Secret")

	// We don't have nodeID information because the secrets holding
	// the certificates in k8s/ocp can be created by other tools (eg cert-manager)
	// We need inspect the node-ids registered in the cache and publish the secrets to
	// all of them
	// TODO: improve this and publish secrets only to those node-ids actually interested
	// in them

	nodeIDs := make([]string, len(r.cache))
	i := 0
	for k := range r.cache {
		nodeIDs[i] = k
		i++
	}

	logger.Info("Pushing secret for each NodeID", "NodeIDs", nodeIDs)

	// validate that the secret is well formed
	if _, ok := secret.Data[secretCertificate]; !ok {
		logger.Error(fmt.Errorf("Secret '%s' is missing required key '%s'", secret.ObjectMeta.Name, secretCertificate), "Malformed 'kubernetes.io/tls' secret")
	}
	if _, ok := secret.Data[secretPrivateKey]; !ok {
		logger.Error(fmt.Errorf("Secret '%s' is missing required key '%s'", secret.ObjectMeta.Name, secretPrivateKey), "Malformed 'kubernetes.io/tls' secret")
	}

	// Copy the secret to all existent node caches
	for _, nodeID := range nodeIDs {
		r.cache.SetResource(nodeID, cn, cache.Secret, envoy.NewSecret(
			cn,
			string(secret.Data[secretPrivateKey]),
			string(secret.Data[secretCertificate]),
		))
		// Publish resources to the ads server cache
		r.cache.BumpCacheVersion(nodeID)
		r.cache.SetSnapshot(nodeID, *r.adsCache)
		logger.V(1).Info("Secret added to cache", "nodeID", nodeID)
	}

	return reconcile.Result{}, nil
}
