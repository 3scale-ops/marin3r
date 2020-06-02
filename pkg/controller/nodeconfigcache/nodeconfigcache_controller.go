package nodeconfigcache

import (
	"context"
	"fmt"
	"time"

	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoyapi_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoyapi_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoyapi_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/protobuf/proto"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"

	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
	"github.com/3scale/marin3r/pkg/envoy"
	xds_cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	secretAnnotation         = "cert-manager.io/common-name"
	secretCertificate        = "tls.crt"
	secretPrivateKey         = "tls.key"
	nodeconfigcacheFinalizer = "finalizer.caches.3scale.net"
)

var log = logf.Log.WithName("controller_nodeconfigcache")

// Add creates a new NodeConfigCache Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, cache *xds_cache.SnapshotCache) error {
	return add(mgr, newReconciler(mgr, cache))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c *xds_cache.SnapshotCache) reconcile.Reconciler {
	return &ReconcileNodeConfigCache{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		adsCache: c,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nodeconfigcache-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	filter := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Filter out all changes to status except for the ConfigFailed condition
			// The addition of a ConfigFailed condition triggers a rollback
			if e.MetaOld.GetGeneration() == e.MetaNew.GetGeneration() {
				nccOld := e.ObjectOld.(*cachesv1alpha1.NodeConfigCache)
				nccNew := e.ObjectNew.(*cachesv1alpha1.NodeConfigCache)
				if !nccOld.Status.Conditions.IsTrueFor("ConfigFailed") && nccNew.Status.Conditions.IsTrueFor("ConfigFailed") {
					return true
				}
				return false
			}
			return true
		},
	}
	// Watch for changes to primary resource NodeConfigCache
	err = c.Watch(&source.Kind{Type: &cachesv1alpha1.NodeConfigCache{}}, &handler.EnqueueRequestForObject{}, filter)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileNodeConfigCache implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNodeConfigCache{}

// ReconcileNodeConfigCache reconciles a NodeConfigCache object
type ReconcileNodeConfigCache struct {
	client   client.Client
	scheme   *runtime.Scheme
	adsCache *xds_cache.SnapshotCache
}

// Reconcile reads that state of the cluster for a NodeConfigCache object and makes changes based on the state read
// and what is in the NodeConfigCache.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNodeConfigCache) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling NodeConfigCache")

	ctx := context.TODO()

	// Fetch the NodeConfigCache instance
	ncc := &cachesv1alpha1.NodeConfigCache{}
	err := r.client.Get(ctx, request.NamespacedName, ncc)
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

	// Check if the NodeConfigCache instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if ncc.GetDeletionTimestamp() != nil {
		if contains(ncc.GetFinalizers(), nodeconfigcacheFinalizer) {
			r.finalizeNodeConfigCache(ncc.Spec.NodeID)
			reqLogger.V(1).Info("Successfully cleared ads server cache")
			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(ncc, nodeconfigcacheFinalizer)
			err := r.client.Update(ctx, ncc)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// TODO: add the label with the nodeID if it is missing

	// Add finalizer for this CR
	if !contains(ncc.GetFinalizers(), nodeconfigcacheFinalizer) {
		reqLogger.Info("Adding Finalizer for the NodeConfigCache")
		if err := r.addFinalizer(ctx, ncc); err != nil {
			reqLogger.Error(err, "Failed adding finalizer for nodecacheconfig")
			return reconcile.Result{}, err
		}
	}

	nodeID := ncc.Spec.NodeID
	version := ncc.Spec.Version
	snap := newNodeSnapshot(nodeID, version)

	// If the rollback condition is true, return the resources from the
	// immediately previous revision instead of the ones in the spec
	// in order to perform a rollback operation. Resources in the spec
	// will be ignored until the rollback condition is cleared
	if ncc.Status.Conditions.IsTrueFor("ConfigFailed") {
		if err := r.rollback(ctx, ncc, snap, reqLogger); err != nil {
			reqLogger.Error(err, "Rollback failed", "NodeID", nodeID)
			return reconcile.Result{}, err
		}
		// Rollback complete, do not requeue
		reqLogger.Info("Failing config detected, rollback performed", "NodeID", nodeID)
		return reconcile.Result{}, nil
	}

	// Deserialize envoy resources from the spec and create a new snapshot with them
	if err := r.loadResources(ctx, request.Name, request.Namespace,
		ncc.Spec.Serialization, ncc.Spec.Resources, field.NewPath("spec", "resources"), snap); err != nil {
		// Requeue with delay, as the envoy resources syntax is probably wrong
		// and that is not a transitory error (some other higher level resource
		// probaly needs fixing)
		reqLogger.Error(err, "Errors occured while loading resources from CR")
		return reconcile.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Update the status with the pusblished version
	if ncc.Status.PublishedVersion != ncc.Spec.Version {
		patch := client.MergeFrom(ncc.DeepCopy())
		ncc.Status.PublishedVersion = ncc.Spec.Version
		if err := r.client.Status().Patch(ctx, ncc, patch); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Create a NodeConfigRevision for this config
	if err := r.ensureNodeConfigRevision(ctx, ncc); err != nil {
		return reconcile.Result{}, err
	}

	// Update the ConfigRevisions list in the status
	if err := r.consolidateRevisionList(ctx, ncc); err != nil {
		return reconcile.Result{}, err
	}

	// Cleanup unreferenced NodeConfigRevision objects
	if err := r.deleteUnreferencedRevisions(ctx, ncc); err != nil {
		return reconcile.Result{}, err
	}

	// Push the snapshot to the xds server cache
	oldSnap, err := (*r.adsCache).GetSnapshot(nodeID)
	if !snapshotIsEqual(snap, &oldSnap) {
		// TODO: check snapshot consistency using "snap.Consistent()". Consider also validating
		// consistency between clusters and listeners, as this is not done by snap.Consistent()
		// because listeners and clusters are not requested by name by the envoy gateways

		// Publish the in-memory cache to the envoy control-plane
		reqLogger.Info("Publishing new snapshot for nodeID", "Version", version, "NodeID", nodeID)
		if err := (*r.adsCache).SetSnapshot(nodeID, *snap); err != nil {
			return reconcile.Result{}, err
		}
	} else {
		reqLogger.Info("Generated snapshot is equal to published one, avoiding push to xds server cache", "NodeID", nodeID)
	}

	// // Remove Rollback condition, if active
	// if ncc.Status.Conditions.IsTrueFor("Rollback") {
	// 	if ncc.Status.Conditions.GetCondition("Rollback").Reason == "RollbackComplete" {
	// 		if err := r.removeRollbackCondition(ctx, ncc); err != nil {
	// 			reqLogger.Error(err, "Failed to remove 'Rollback' condition", "NodeID", nodeID)
	// 			return reconcile.Result{}, err
	// 		}
	// 	}
	// }

	return reconcile.Result{}, nil
}

func resourceLoaderError(name, namespace, rtype, rvalue string, resPath *field.Path, idx int) error {
	return errors.NewInvalid(
		schema.GroupKind{Group: "caches", Kind: "NodeCacheConfig"},
		fmt.Sprintf("%s/%s", namespace, name),
		field.ErrorList{
			field.Invalid(
				resPath.Child(rtype).Index(idx).Child("Value"),
				rvalue,
				fmt.Sprint("Invalid envoy resource value"),
			),
		},
	)
}

func (r *ReconcileNodeConfigCache) loadResources(ctx context.Context, name, namespace, serialization string,
	resources *cachesv1alpha1.EnvoyResources, resPath *field.Path, snap *xds_cache.Snapshot) error {

	var ds envoy.ResourceUnmarshaller
	switch serialization {
	case "b64json":
		ds = envoy.B64JSON{}

	case "yaml":
		ds = envoy.YAML{}
	default:
		// "json" is the default
		ds = envoy.JSON{}
	}

	for idx, endpoint := range resources.Endpoints {
		res := &envoyapi.ClusterLoadAssignment{}
		if err := ds.Unmarshal(endpoint.Value, res); err != nil {
			return resourceLoaderError(name, namespace, "Endpoints", endpoint.Value, resPath, idx)
		}
		setResource(endpoint.Name, res, snap)
	}

	for idx, cluster := range resources.Clusters {
		res := &envoyapi.Cluster{}
		if err := ds.Unmarshal(cluster.Value, res); err != nil {
			return resourceLoaderError(name, namespace, "Clusters", cluster.Value, resPath, idx)
		}
		setResource(cluster.Name, res, snap)
	}

	for idx, route := range resources.Routes {
		res := &envoyapi_route.Route{}
		if err := ds.Unmarshal(route.Value, res); err != nil {
			return resourceLoaderError(name, namespace, "Routes", route.Value, resPath, idx)
		}
		setResource(route.Name, res, snap)
	}

	for idx, listener := range resources.Listeners {
		res := &envoyapi.Listener{}
		if err := ds.Unmarshal(listener.Value, res); err != nil {
			return resourceLoaderError(name, namespace, "Listeners", listener.Value, resPath, idx)
		}
		setResource(listener.Name, res, snap)
	}

	for idx, runtime := range resources.Runtimes {
		res := &envoyapi_discovery.Runtime{}
		if err := ds.Unmarshal(runtime.Value, res); err != nil {
			return resourceLoaderError(name, namespace, "Runtimes", runtime.Value, resPath, idx)
		}
		setResource(runtime.Name, res, snap)
	}

	for idx, secret := range resources.Secrets {
		s := &corev1.Secret{}
		key := types.NamespacedName{
			Name:      secret.Ref.Name,
			Namespace: secret.Ref.Namespace,
		}
		if err := r.client.Get(ctx, key, s); err != nil {
			if errors.IsNotFound(err) {
				return err
			}
			return err
		}

		// Validate secret holds a certificate
		if s.Type == "kubernetes.io/tls" {
			res := envoy.NewSecret(secret.Name, string(s.Data[secretPrivateKey]), string(s.Data[secretCertificate]))
			setResource(secret.Name, res, snap)
		} else {
			return errors.NewInvalid(
				schema.GroupKind{Group: "caches", Kind: "NodeCacheConfig"},
				fmt.Sprintf("%s/%s", namespace, name),
				field.ErrorList{
					field.Invalid(
						resPath.Child("Secrets").Index(idx).Child("Ref"),
						secret.Ref,
						fmt.Sprint("Only 'kubernetes.io/tls' type secrets allowed"),
					),
				},
			)
		}
	}

	return nil
}

func newNodeSnapshot(nodeID string, version string) *xds_cache.Snapshot {

	snap := xds_cache.Snapshot{Resources: [6]xds_cache.Resources{}}
	snap.Resources[xds_cache_types.Listener] = xds_cache.NewResources(version, []xds_cache_types.Resource{})
	snap.Resources[xds_cache_types.Endpoint] = xds_cache.NewResources(version, []xds_cache_types.Resource{})
	snap.Resources[xds_cache_types.Cluster] = xds_cache.NewResources(version, []xds_cache_types.Resource{})
	snap.Resources[xds_cache_types.Route] = xds_cache.NewResources(version, []xds_cache_types.Resource{})
	snap.Resources[xds_cache_types.Secret] = xds_cache.NewResources(version, []xds_cache_types.Resource{})
	snap.Resources[xds_cache_types.Runtime] = xds_cache.NewResources(version, []xds_cache_types.Resource{})

	return &snap
}

func setResource(name string, res xds_cache_types.Resource, snap *xds_cache.Snapshot) {

	switch o := res.(type) {

	case *envoyapi.ClusterLoadAssignment:
		snap.Resources[xds_cache_types.Endpoint].Items[name] = o

	case *envoyapi.Cluster:
		snap.Resources[xds_cache_types.Cluster].Items[name] = o

	case *envoyapi_route.Route:
		snap.Resources[xds_cache_types.Route].Items[name] = o

	case *envoyapi.Listener:
		snap.Resources[xds_cache_types.Listener].Items[name] = o

	case *envoyapi_auth.Secret:
		snap.Resources[xds_cache_types.Secret].Items[name] = o

	case *envoyapi_discovery.Runtime:
		snap.Resources[xds_cache_types.Runtime].Items[name] = o

	}
}

func snapshotIsEqual(newSnap, oldSnap *xds_cache.Snapshot) bool {

	// Check resources are equal for each resource type
	for rtype, newResources := range newSnap.Resources {
		oldResources := oldSnap.Resources[rtype]

		// If lenght is not equal, resources are not equal
		if len(oldResources.Items) != len(newResources.Items) {
			return false
		}

		for name, oldValue := range oldResources.Items {

			newValue, ok := newResources.Items[name]

			// If some key does not exist, resources are not equal
			if !ok {
				return false
			}

			// If value has changed, resources are not equal
			if !proto.Equal(oldValue, newValue) {
				return false
			}
		}
	}
	return true
}

func (r *ReconcileNodeConfigCache) finalizeNodeConfigCache(nodeID string) {
	(*r.adsCache).ClearSnapshot(nodeID)
}

func (r *ReconcileNodeConfigCache) addFinalizer(ctx context.Context, ncc *cachesv1alpha1.NodeConfigCache) error {
	controllerutil.AddFinalizer(ncc, nodeconfigcacheFinalizer)

	// Update CR
	err := r.client.Update(ctx, ncc)
	if err != nil {
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
