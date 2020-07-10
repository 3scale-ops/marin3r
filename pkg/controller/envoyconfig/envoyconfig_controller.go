package envoyconfig

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

var log = logf.Log.WithName("controller_envoyconfig")

// Add creates a new EnvoyConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, cache *xds_cache.SnapshotCache) error {
	return add(mgr, newReconciler(mgr, cache))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c *xds_cache.SnapshotCache) reconcile.Reconciler {
	return &ReconcileEnvoyConfig{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		adsCache: c,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("envoyconfig-controller", mgr, controller.Options{Reconciler: r})
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
					e.ObjectOld.(*marin3rv1alpha1.EnvoyConfig).Status.Conditions,
					e.ObjectNew.(*marin3rv1alpha1.EnvoyConfig).Status.Conditions,
				) {
					return true
				}
				return false
			}
			return true
		},
	}
	// Watch for changes to primary resource EnvoyConfig
	err = c.Watch(&source.Kind{Type: &marin3rv1alpha1.EnvoyConfig{}}, &handler.EnqueueRequestForObject{}, filter)
	if err != nil {
		return err
	}

	// Watch for owned resources EnvoyConfigRevision
	err = c.Watch(&source.Kind{Type: &marin3rv1alpha1.EnvoyConfigRevision{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &marin3rv1alpha1.EnvoyConfig{},
	})

	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileEnvoyConfig implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileEnvoyConfig{}

// ReconcileEnvoyConfig reconciles a EnvoyConfig object
type ReconcileEnvoyConfig struct {
	client   client.Client
	scheme   *runtime.Scheme
	adsCache *xds_cache.SnapshotCache
}

// Reconcile reads that state of the cluster for a EnvoyConfig object and makes changes based on the state read
// and what is in the EnvoyConfig.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileEnvoyConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling EnvoyConfig")

	ctx := context.TODO()

	// Fetch the EnvoyConfig instance
	ec := &marin3rv1alpha1.EnvoyConfig{}
	err := r.client.Get(ctx, request.NamespacedName, ec)
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

	// Check if the EnvoyConfig instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if ec.GetDeletionTimestamp() != nil {
		if contains(ec.GetFinalizers(), marin3rv1alpha1.EnvoyConfigFinalizer) {
			r.finalizeEnvoyConfig(ec.Spec.NodeID)
			reqLogger.V(1).Info("Successfully cleared ads server cache")
			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(ec, marin3rv1alpha1.EnvoyConfigFinalizer)
			err := r.client.Update(ctx, ec)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// TODO: add the label with the nodeID if it is missing

	// Add finalizer for this CR
	if !contains(ec.GetFinalizers(), marin3rv1alpha1.EnvoyConfigFinalizer) {
		reqLogger.Info("Adding Finalizer for the EnvoyConfig")
		if err := r.addFinalizer(ctx, ec); err != nil {
			reqLogger.Error(err, "Failed adding finalizer for envoyconfig")
			return reconcile.Result{}, err
		}
	}

	// desiredVersion is the version that matches the resources described in the spec
	desiredVersion := calculateRevisionHash(ec.Spec.EnvoyResources)

	// ensure that the desiredVersion has a matching revision object
	if err := r.ensureEnvoyConfigRevision(ctx, ec, desiredVersion); err != nil {
		return reconcile.Result{}, err
	}

	// Update the ConfigRevisions list in the status
	if err := r.consolidateRevisionList(ctx, ec, desiredVersion); err != nil {
		return reconcile.Result{}, err
	}

	// determine the version that should be published
	version, err := r.getVersionToPublish(ctx, ec)
	if err != nil {
		if err.(cacheError).ErrorType == AllRevisionsTaintedError {
			if err := r.setRollbackFailed(ctx, ec); err != nil {
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
	if err := r.markRevisionPublished(ctx, ec.Spec.NodeID, version, "VersionPublished", fmt.Sprintf("Version '%s' has been published", version)); err != nil {
		return reconcile.Result{}, err
	}

	// Update the status
	if err := r.updateStatus(ctx, ec, desiredVersion, version); err != nil {
		return reconcile.Result{}, err
	}

	// Cleanup unreferenced EnvoyConfigRevision objects
	if err := r.deleteUnreferencedRevisions(ctx, ec); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileEnvoyConfig) getVersionToPublish(ctx context.Context, ec *marin3rv1alpha1.EnvoyConfig) (string, error) {
	// Get the list of revisions for this nodeID
	ecrList := &marin3rv1alpha1.EnvoyConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID},
	})
	if err != nil {
		return "", newCacheError(UnknownError, "getVersionToPublish", err.Error())
	}
	err = r.client.List(ctx, ecrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return "", newCacheError(UnknownError, "getVersionToPublish", err.Error())
	}

	// Starting from the highest index in the ConfigRevision list and going
	// down, return the first version found that is not tainted
	for i := len(ec.Status.ConfigRevisions) - 1; i >= 0; i-- {
		for _, ecr := range ecrList.Items {
			if ec.Status.ConfigRevisions[i].Version == ecr.Spec.Version && !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionTaintedCondition) {
				return ec.Status.ConfigRevisions[i].Version, nil
			}
		}
	}

	// If we get here it means that there is not untainted revision. Return a specific
	// error to the controller loop so it gets handled appropriately
	return "", newCacheError(AllRevisionsTaintedError, "getVersionToPublish", "All available revisions are tainted")
}

func (r *ReconcileEnvoyConfig) updateStatus(ctx context.Context, ec *marin3rv1alpha1.EnvoyConfig, desired, published string) error {

	changed := false
	patch := client.MergeFrom(ec.DeepCopy())

	if ec.Status.PublishedVersion != published {
		ec.Status.PublishedVersion = published
		changed = true
	}

	if ec.Status.DesiredVersion != desired {
		ec.Status.DesiredVersion = desired
		changed = true
	}

	// Set the cacheStatus field
	if desired != published && ec.Status.CacheState != marin3rv1alpha1.RollbackState {
		ec.Status.CacheState = marin3rv1alpha1.RollbackState
		changed = true
	}
	if desired == published && ec.Status.CacheState != marin3rv1alpha1.InSyncState {
		ec.Status.CacheState = marin3rv1alpha1.InSyncState
		changed = true
	}

	// Set the CacheOutOfSyncCondition
	if desired != published && !ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.CacheOutOfSyncCondition) {
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:    marin3rv1alpha1.CacheOutOfSyncCondition,
			Status:  corev1.ConditionTrue,
			Reason:  "CantPublishDesiredVersion",
			Message: "Desired resources spec cannot be applied",
		})
		changed = true
	} else if desired == published && !ec.Status.Conditions.IsFalseFor(marin3rv1alpha1.CacheOutOfSyncCondition) {
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:    marin3rv1alpha1.CacheOutOfSyncCondition,
			Status:  corev1.ConditionFalse,
			Reason:  "DesiredVersionPublished",
			Message: "Desired version successfully published",
		})
		changed = true
	}

	// Clear the RollbackFailedCondition (if we have reached this code it means that
	// at least one untainted revision exists)
	if ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.RollbackFailedCondition) {
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:   marin3rv1alpha1.RollbackFailedCondition,
			Reason: "Recovered",
			Status: corev1.ConditionFalse,
		})
		changed = true
	}

	// Only write if something needs changing to reduce API calls
	if changed {
		if err := r.client.Status().Patch(ctx, ec, patch); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileEnvoyConfig) finalizeEnvoyConfig(nodeID string) {
	(*r.adsCache).ClearSnapshot(nodeID)
}

func (r *ReconcileEnvoyConfig) addFinalizer(ctx context.Context, ec *marin3rv1alpha1.EnvoyConfig) error {
	controllerutil.AddFinalizer(ec, marin3rv1alpha1.EnvoyConfigFinalizer)

	// Update CR
	err := r.client.Update(ctx, ec)
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

func (r *ReconcileEnvoyConfig) setRollbackFailed(ctx context.Context, ec *marin3rv1alpha1.EnvoyConfig) error {
	if !ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.RollbackFailedCondition) {
		patch := client.MergeFrom(ec.DeepCopy())
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:    marin3rv1alpha1.RollbackFailedCondition,
			Status:  corev1.ConditionTrue,
			Reason:  "AllRevisionsTainted",
			Message: "All revisions are tainted, rollback failed",
		})
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:    marin3rv1alpha1.CacheOutOfSyncCondition,
			Status:  corev1.ConditionTrue,
			Reason:  "AllRevisionsTainted",
			Message: "All revisions are tainted, rollback failed",
		})
		ec.Status.CacheState = marin3rv1alpha1.RollbackFailedState

		if err := r.client.Status().Patch(ctx, ec, patch); err != nil {
			return err
		}
	}
	return nil
}
