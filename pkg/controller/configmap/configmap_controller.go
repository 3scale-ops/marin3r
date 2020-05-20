package configmap

import (
	"context"

	"github.com/3scale/marin3r/pkg/cache"
	"github.com/3scale/marin3r/pkg/envoy"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
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
func Add(mgr manager.Manager, cache *xds_cache.SnapshotCache) error {
	return add(mgr, newReconciler(mgr, cache))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c *xds_cache.SnapshotCache) reconcile.Reconciler {
	return &ReconcileConfigMap{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		cache:    cache.NewCache(),
		adsCache: c,
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
				return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
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
	client   client.Client
	scheme   *runtime.Scheme
	cache    cache.Cache
	adsCache *xds_cache.SnapshotCache
}

// Reconcile reads that state of the cluster for a ConfigMap object and makes changes based on the state read
// and what is in the ConfigMap.Spec
/// Note:
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

	// Check if it's the first time we see this
	// nodeID, in which case we need to bootstrap
	// its cache
	if _, ok := r.cache[nodeID]; !ok {
		reqLogger.Info("New node-id, boostraping in-memory cache")
		r.cache.NewNodeCache(nodeID)
		// We need to trigger a reconcile for the secrets
		// so this new cache gets populated with them
		secrets, err := getNamespaceSecrets(ctx, r.client, request.Namespace, nodeID, reqLogger)
		for _, s := range secrets {
			r.cache.SetResource(nodeID, s.Name, cache.Secret, s)
		}
		if err != nil {
			reqLogger.Error(err, "Error populating secrets cache")
			// Delete the node cache so in the next reconcile it will try to rebuild the
			// secrets cache again
			r.cache.DeleteNodeCache(nodeID)
			// Reenqueue
			return reconcile.Result{}, err
		}
	}

	// Clear current cached clusters and listeners, we don't care about
	// previous values because the yaml in the ConfigMap provider is
	// expected to be complete
	r.cache.ClearResources(nodeID, cache.Cluster)
	r.cache.ClearResources(nodeID, cache.Listener)
	// TODO: function that acutally validates that the resources in the ConfigMap have changed
	// to avoid unnecessary updates of the ads server cache

	// Get envoy resources
	resources, err := envoy.YAMLtoResources([]byte(cm.Data[configMapKey]), reqLogger)

	for _, cluster := range resources.Clusters {
		r.cache.SetResource(nodeID, cluster.Name, cache.Cluster, cluster)
	}

	for _, lis := range resources.Listeners {
		r.cache.SetResource(nodeID, lis.Name, cache.Listener, lis)
	}

	// Publish resources to the ads server cache
	r.cache.BumpCacheVersion(nodeID)
	r.cache.SetSnapshot(nodeID, *r.adsCache)

	return reconcile.Result{}, nil
}

// SyncNodeSecrets synchronously builds/rebuilds the whole secrets cache
func getNamespaceSecrets(ctx context.Context, cli client.Client, namespace, nodeID string, logger logr.Logger) ([]*auth.Secret, error) {

	list := &corev1.SecretList{}
	err := cli.List(ctx, list, []client.ListOption{client.InNamespace(namespace)}...)
	if err != nil {
		return nil, err
	}

	secrets := []*auth.Secret{}
	for _, s := range list.Items {
		if cn, ok := s.GetAnnotations()[secretAnnotation]; ok {
			logger.V(1).Info("Discovered secret containing certificate", "Secret", s.ObjectMeta.Name, "CN", cn)
			secrets = append(secrets, envoy.NewSecret(cn, string(s.Data[secretPrivateKey]), string(s.Data[secretCertificate])))
		}
	}
	return secrets, nil
}
