package nodeconfigcache

import (
	"context"
	"fmt"

	marin3rv1alpha1 "github.com/3scale/marin3r/pkg/apis/marin3r/v1alpha1"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/operator-framework/operator-sdk/pkg/status"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
					e.ObjectOld.(*marin3rv1alpha1.NodeConfigCache).Status.Conditions,
					e.ObjectNew.(*marin3rv1alpha1.NodeConfigCache).Status.Conditions,
				) {
					return true
				}
				return false
			}
			return true
		},
	}
	// Watch for changes to primary resource NodeConfigCache
	err = c.Watch(&source.Kind{Type: &marin3rv1alpha1.NodeConfigCache{}}, &handler.EnqueueRequestForObject{}, filter)
	if err != nil {
		return err
	}

	// Watch for owned resources NodeConfigRevision
	err = c.Watch(&source.Kind{Type: &marin3rv1alpha1.NodeConfigRevision{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &marin3rv1alpha1.NodeConfigCache{},
	})

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
	ncc := &marin3rv1alpha1.NodeConfigCache{}
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
		if contains(ncc.GetFinalizers(), marin3rv1alpha1.NodeConfigCacheFinalizer) {
			r.finalizeNodeConfigCache(ncc.Spec.NodeID)
			reqLogger.V(1).Info("Successfully cleared ads server cache")
			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(ncc, marin3rv1alpha1.NodeConfigCacheFinalizer)
			err := r.client.Update(ctx, ncc)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// TODO: add the label with the nodeID if it is missing

	// Add finalizer for this CR
	if !contains(ncc.GetFinalizers(), marin3rv1alpha1.NodeConfigCacheFinalizer) {
		reqLogger.Info("Adding Finalizer for the NodeConfigCache")
		if err := r.addFinalizer(ctx, ncc); err != nil {
			reqLogger.Error(err, "Failed adding finalizer for nodecacheconfig")
			return reconcile.Result{}, err
		}
	}

	// desiredVersion is the version that matches the resources described in the spec
	desiredVersion := calculateRevisionHash(ncc.Spec.Resources)

	// ensure that the desiredVersion has a matching revision object
	if err := r.ensureNodeConfigRevision(ctx, ncc, desiredVersion); err != nil {
		return reconcile.Result{}, err
	}

	// Update the ConfigRevisions list in the status
	if err := r.consolidateRevisionList(ctx, ncc, desiredVersion); err != nil {
		return reconcile.Result{}, err
	}

	// determine the version that should be published
	version, err := r.getVersionToPublish(ctx, ncc)
	if err != nil {
		if err.(cacheError).ErrorType == AllRevisionsTaintedError {
			if err := r.setRollbackFailed(ctx, ncc); err != nil {
				return reconcile.Result{}, err
			}
			// This is an unrecoverable error because there are no
			// revisions to try and the controller cannot reconcile fix
			// this by . Set the RollbackFailedCOndition and exit without requeuing
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Mark the "version" as teh published revision
	if err := r.markRevisionPublished(ctx, ncc.Spec.NodeID, version, "VersionPublished", fmt.Sprintf("Version '%s' has been published", version)); err != nil {
		return reconcile.Result{}, err
	}

	// Update the status
	if err := r.updateStatus(ctx, ncc, desiredVersion, version); err != nil {
		return reconcile.Result{}, err
	}

	// Cleanup unreferenced NodeConfigRevision objects
	if err := r.deleteUnreferencedRevisions(ctx, ncc); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileNodeConfigCache) getVersionToPublish(ctx context.Context, ncc *marin3rv1alpha1.NodeConfigCache) (string, error) {
	// Get the list of revisions for this nodeID
	ncrList := &marin3rv1alpha1.NodeConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: ncc.Spec.NodeID},
	})
	if err != nil {
		return "", newCacheError(UnknownError, "getVersionToPublish", err.Error())
	}
	err = r.client.List(ctx, ncrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return "", newCacheError(UnknownError, "getVersionToPublish", err.Error())
	}

	// Starting from the highest index in the ConfigRevision list and going
	// down, return the first version found that is not tainted
	for i := len(ncc.Status.ConfigRevisions) - 1; i >= 0; i-- {
		for _, ncr := range ncrList.Items {
			if ncc.Status.ConfigRevisions[i].Version == ncr.Spec.Version && !ncr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionTaintedCondition) {
				return ncc.Status.ConfigRevisions[i].Version, nil
			}
		}
	}

	// If we get here it means that there is not untainted revision. Return a specific
	// error to the controller loop so it gets handled appropriately
	return "", newCacheError(AllRevisionsTaintedError, "getVersionToPublish", "All available revisions are tainted")
}

func (r *ReconcileNodeConfigCache) updateStatus(ctx context.Context, ncc *marin3rv1alpha1.NodeConfigCache, desired, published string) error {

	changed := false
	patch := client.MergeFrom(ncc.DeepCopy())

	if ncc.Status.PublishedVersion != published {
		ncc.Status.PublishedVersion = published
		changed = true
	}

	if ncc.Status.DesiredVersion != desired {
		ncc.Status.DesiredVersion = desired
		changed = true
	}

	// Set the cacheStatus field
	if desired != published && ncc.Status.CacheState != marin3rv1alpha1.RollbackState {
		ncc.Status.CacheState = marin3rv1alpha1.RollbackState
		changed = true
	}
	if desired == published && ncc.Status.CacheState != marin3rv1alpha1.InSyncState {
		ncc.Status.CacheState = marin3rv1alpha1.InSyncState
		changed = true
	}

	// Set the CacheOutOfSyncCondition
	if desired != published && !ncc.Status.Conditions.IsTrueFor(marin3rv1alpha1.CacheOutOfSyncCondition) {
		ncc.Status.Conditions.SetCondition(status.Condition{
			Type:    marin3rv1alpha1.CacheOutOfSyncCondition,
			Status:  corev1.ConditionTrue,
			Reason:  "CantPublishDesiredVersion",
			Message: "Desired resources spec cannot be applied",
		})
		changed = true
	} else if desired == published && !ncc.Status.Conditions.IsFalseFor(marin3rv1alpha1.CacheOutOfSyncCondition) {
		ncc.Status.Conditions.SetCondition(status.Condition{
			Type:    marin3rv1alpha1.CacheOutOfSyncCondition,
			Status:  corev1.ConditionFalse,
			Reason:  "DesiredVersionPublished",
			Message: "Desired version successfully published",
		})
		changed = true
	}

	// Clear the RollbackFailedCondition (if we have reached this code it means that
	// at least one untainted revision exists)
	if ncc.Status.Conditions.IsTrueFor(marin3rv1alpha1.RollbackFailedCondition) {
		ncc.Status.Conditions.SetCondition(status.Condition{
			Type:   marin3rv1alpha1.RollbackFailedCondition,
			Reason: "Recovered",
			Status: corev1.ConditionFalse,
		})
		changed = true
	}

	// Only write if something needs changing to reduce API calls
	if changed {
		if err := r.client.Status().Patch(ctx, ncc, patch); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileNodeConfigCache) finalizeNodeConfigCache(nodeID string) {
	(*r.adsCache).ClearSnapshot(nodeID)
}

func (r *ReconcileNodeConfigCache) addFinalizer(ctx context.Context, ncc *marin3rv1alpha1.NodeConfigCache) error {
	controllerutil.AddFinalizer(ncc, marin3rv1alpha1.NodeConfigCacheFinalizer)

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

func (r *ReconcileNodeConfigCache) setRollbackFailed(ctx context.Context, ncc *marin3rv1alpha1.NodeConfigCache) error {
	if !ncc.Status.Conditions.IsTrueFor(marin3rv1alpha1.RollbackFailedCondition) {
		patch := client.MergeFrom(ncc.DeepCopy())
		ncc.Status.Conditions.SetCondition(status.Condition{
			Type:    marin3rv1alpha1.RollbackFailedCondition,
			Status:  corev1.ConditionTrue,
			Reason:  "AllRevisionsTainted",
			Message: "All revisions are tainted, rollback failed",
		})
		ncc.Status.Conditions.SetCondition(status.Condition{
			Type:    marin3rv1alpha1.CacheOutOfSyncCondition,
			Status:  corev1.ConditionTrue,
			Reason:  "AllRevisionsTainted",
			Message: "All revisions are tainted, rollback failed",
		})
		ncc.Status.CacheState = marin3rv1alpha1.RollbackFailedState

		if err := r.client.Status().Patch(ctx, ncc, patch); err != nil {
			return err
		}
	}
	return nil
}
