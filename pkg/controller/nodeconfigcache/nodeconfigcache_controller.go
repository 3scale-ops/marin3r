package nodeconfigcache

import (
	"context"

	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/operator-framework/operator-sdk/pkg/status"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
	previousVersionPrefix string = "ReceivedPreviousVersion"
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
			// Ignore updates to CR status in which case metadata.Generation does not change
			if e.MetaOld.GetGeneration() == e.MetaNew.GetGeneration() {
				// But trigger reconciles on condition updates as this is the way
				// other controllers communicate with this one
				if !apiequality.Semantic.DeepEqual(
					e.ObjectOld.(*cachesv1alpha1.NodeConfigCache).Status.Conditions,
					e.ObjectNew.(*cachesv1alpha1.NodeConfigCache).Status.Conditions,
				) {
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
		if contains(ncc.GetFinalizers(), cachesv1alpha1.NodeConfigCacheFinalizer) {
			r.finalizeNodeConfigCache(ncc.Spec.NodeID)
			reqLogger.V(1).Info("Successfully cleared ads server cache")
			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(ncc, cachesv1alpha1.NodeConfigCacheFinalizer)
			err := r.client.Update(ctx, ncc)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// TODO: add the label with the nodeID if it is missing

	// Add finalizer for this CR
	if !contains(ncc.GetFinalizers(), cachesv1alpha1.NodeConfigCacheFinalizer) {
		reqLogger.Info("Adding Finalizer for the NodeConfigCache")
		if err := r.addFinalizer(ctx, ncc); err != nil {
			reqLogger.Error(err, "Failed adding finalizer for nodecacheconfig")
			return reconcile.Result{}, err
		}
	}

	// If the rollback condition is true, return the resources from the
	// immediately previous revision instead of the ones in the spec
	// in order to perform a rollback operation. Resources in the spec
	// will be ignored until the rollback condition is cleared

	// if ncc.Status.Conditions.IsTrueFor(ResourcesUpdateUnsuccessful) {
	// 	if err := r.rollback(ctx, ncc, snap, reqLogger); err != nil {
	// 		reqLogger.Error(err, "Rollback failed", "NodeID", nodeID)
	// 		return reconcile.Result{}, err
	// 	}
	// 	// Rollback complete, do not requeue
	// 	reqLogger.Info("Failing config detected, rollback performed", "NodeID", nodeID)
	// 	return reconcile.Result{}, nil
	// }

	version := calculateRevisionHash(ncc.Spec.Resources)

	// Update the status with the pusblished version, init conditions
	if ncc.Status.PublishedVersion != version {
		patch := client.MergeFrom(ncc.DeepCopy())
		// if len(ncc.Status.Conditions) == 0 {
		// 	ncc.Status.Conditions = status.NewConditions()
		// }
		ncc.Status.PublishedVersion = version
		if err := r.client.Status().Patch(ctx, ncc, patch); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Create a NodeConfigRevision for this config
	if err := r.ensureNodeConfigRevision(ctx, ncc, ncc.Status.PublishedVersion); err != nil {
		return reconcile.Result{}, err
	}

	// Update the ConfigRevisions list in the status
	if err := r.consolidateRevisionList(ctx, ncc, ncc.Status.PublishedVersion); err != nil {
		return reconcile.Result{}, err
	}

	// Cleanup unreferenced NodeConfigRevision objects
	if err := r.deleteUnreferencedRevisions(ctx, ncc); err != nil {
		return reconcile.Result{}, err
	}

	// Clear any ResourcesOutOfSyncCondition
	if err := r.clearResourcesOutOfSyncCondition(ctx, ncc); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileNodeConfigCache) finalizeNodeConfigCache(nodeID string) {
	(*r.adsCache).ClearSnapshot(nodeID)
}

func (r *ReconcileNodeConfigCache) addFinalizer(ctx context.Context, ncc *cachesv1alpha1.NodeConfigCache) error {
	controllerutil.AddFinalizer(ncc, cachesv1alpha1.NodeConfigCacheFinalizer)

	// Update CR
	err := r.client.Update(ctx, ncc)
	if err != nil {
		return err
	}
	return nil
}

func (r *ReconcileNodeConfigCache) clearResourcesOutOfSyncCondition(ctx context.Context, ncc *cachesv1alpha1.NodeConfigCache) error {

	if ncc.Status.Conditions.IsTrueFor(cachesv1alpha1.ResourcesOutOfSyncCondition) {
		patch := client.MergeFrom(ncc.DeepCopy())
		ncc.Status.Conditions.SetCondition(status.Condition{
			Type:    cachesv1alpha1.ResourcesOutOfSyncCondition,
			Reason:  "NodeConficCacheSynced",
			Status:  corev1.ConditionFalse,
			Message: "NodeConfigCache successfully synced",
		})
		if err := r.client.Patch(ctx, ncc, patch); err != nil {
			return err
		}
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
