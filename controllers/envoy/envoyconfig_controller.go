/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EnvoyConfigReconciler reconciles a EnvoyConfig object
type EnvoyConfigReconciler struct {
	Client   client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	ADSCache *xds_cache.SnapshotCache
}

// +kubebuilder:rbac:groups=envoy.marin3r.3scale.net,resources=envoyconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=envoy.marin3r.3scale.net,resources=envoyconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=envoy.marin3r.3scale.net,resources=envoyconfigrevisions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=envoy.marin3r.3scale.net,resources=envoyconfigrevisions/status,verbs=get;update;patch

func (r *EnvoyConfigReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("envoyconfig", req.NamespacedName)

	// Fetch the EnvoyConfig instance
	ec := &envoyv1alpha1.EnvoyConfig{}
	err := r.Client.Get(ctx, req.NamespacedName, ec)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Add finalizer for this CR
	if !contains(ec.GetFinalizers(), envoyv1alpha1.EnvoyConfigFinalizer) {
		r.Log.Info("Adding Finalizer for the EnvoyConfig")
		if err := r.addFinalizer(ctx, ec); err != nil {
			r.Log.Error(err, "Failed adding finalizer for envoyconfig")
			return ctrl.Result{}, err
		}
	}

	// Check if the EnvoyConfig instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if ec.GetDeletionTimestamp() != nil {
		if contains(ec.GetFinalizers(), envoyv1alpha1.EnvoyConfigFinalizer) {
			r.finalizeEnvoyConfig(ec.Spec.NodeID)
			r.Log.V(1).Info("Successfully cleared ads server cache")
			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(ec, envoyv1alpha1.EnvoyConfigFinalizer)
			err := r.Client.Update(ctx, ec)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// TODO: add the label with the nodeID if it is missing

	// desiredVersion is the version that matches the resources described in the spec
	desiredVersion := calculateRevisionHash(ec.Spec.EnvoyResources)

	// ensure that the desiredVersion has a matching revision object
	if err := r.ensureEnvoyConfigRevision(ctx, ec, desiredVersion); err != nil {
		return ctrl.Result{}, err
	}

	// Update the ConfigRevisions list in the status
	if err := r.consolidateRevisionList(ctx, ec, desiredVersion); err != nil {
		return ctrl.Result{}, err
	}

	// determine the version that should be published
	version, err := r.getVersionToPublish(ctx, ec)
	if err != nil {
		if err.(cacheError).ErrorType == AllRevisionsTaintedError {
			if err := r.setRollbackFailed(ctx, ec); err != nil {
				return ctrl.Result{}, err
			}
			// This is an unrecoverable error because there are no
			// revisions to try and the controller cannot reconcile fix
			// this by . Set the RollbackFailedCOndition and exit without requeuing
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Mark the "version" as teh published revision
	if err := r.markRevisionPublished(ctx, ec.Spec.NodeID, version, "VersionPublished", fmt.Sprintf("Version '%s' has been published", version)); err != nil {
		return ctrl.Result{}, err
	}

	// Update the status
	if err := r.updateStatus(ctx, ec, desiredVersion, version); err != nil {
		return ctrl.Result{}, err
	}

	// Cleanup unreferenced EnvoyConfigRevision objects
	if err := r.deleteUnreferencedRevisions(ctx, ec); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *EnvoyConfigReconciler) getVersionToPublish(ctx context.Context, ec *envoyv1alpha1.EnvoyConfig) (string, error) {
	// Get the list of revisions for this nodeID
	ecrList := &envoyv1alpha1.EnvoyConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID},
	})
	if err != nil {
		return "", newCacheError(UnknownError, "getVersionToPublish", err.Error())
	}
	err = r.Client.List(ctx, ecrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return "", newCacheError(UnknownError, "getVersionToPublish", err.Error())
	}

	// Starting from the highest index in the ConfigRevision list and going
	// down, return the first version found that is not tainted
	for i := len(ec.Status.ConfigRevisions) - 1; i >= 0; i-- {
		for _, ecr := range ecrList.Items {
			if ec.Status.ConfigRevisions[i].Version == ecr.Spec.Version && !ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionTaintedCondition) {
				return ec.Status.ConfigRevisions[i].Version, nil
			}
		}
	}

	// If we get here it means that there is not untainted revision. Return a specific
	// error to the controller loop so it gets handled appropriately
	return "", newCacheError(AllRevisionsTaintedError, "getVersionToPublish", "All available revisions are tainted")
}

func (r *EnvoyConfigReconciler) updateStatus(ctx context.Context, ec *envoyv1alpha1.EnvoyConfig, desired, published string) error {

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
	if desired != published && ec.Status.CacheState != envoyv1alpha1.RollbackState {
		ec.Status.CacheState = envoyv1alpha1.RollbackState
		changed = true
	}
	if desired == published && ec.Status.CacheState != envoyv1alpha1.InSyncState {
		ec.Status.CacheState = envoyv1alpha1.InSyncState
		changed = true
	}

	// Set the CacheOutOfSyncCondition
	if desired != published && !ec.Status.Conditions.IsTrueFor(envoyv1alpha1.CacheOutOfSyncCondition) {
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:    envoyv1alpha1.CacheOutOfSyncCondition,
			Status:  corev1.ConditionTrue,
			Reason:  "CantPublishDesiredVersion",
			Message: "Desired resources spec cannot be applied",
		})
		changed = true
	} else if desired == published && !ec.Status.Conditions.IsFalseFor(envoyv1alpha1.CacheOutOfSyncCondition) {
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:    envoyv1alpha1.CacheOutOfSyncCondition,
			Status:  corev1.ConditionFalse,
			Reason:  "DesiredVersionPublished",
			Message: "Desired version successfully published",
		})
		changed = true
	}

	// Clear the RollbackFailedCondition (if we have reached this code it means that
	// at least one untainted revision exists)
	if ec.Status.Conditions.IsTrueFor(envoyv1alpha1.RollbackFailedCondition) {
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:   envoyv1alpha1.RollbackFailedCondition,
			Reason: "Recovered",
			Status: corev1.ConditionFalse,
		})
		changed = true
	}

	// Only write if something needs changing to reduce API calls
	if changed {
		if err := r.Client.Status().Patch(ctx, ec, patch); err != nil {
			return err
		}
	}

	return nil
}

func (r *EnvoyConfigReconciler) finalizeEnvoyConfig(nodeID string) {
	(*r.ADSCache).ClearSnapshot(nodeID)
}

func (r *EnvoyConfigReconciler) addFinalizer(ctx context.Context, ec *envoyv1alpha1.EnvoyConfig) error {
	controllerutil.AddFinalizer(ec, envoyv1alpha1.EnvoyConfigFinalizer)

	// Update CR
	err := r.Client.Update(ctx, ec)
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

func (r *EnvoyConfigReconciler) setRollbackFailed(ctx context.Context, ec *envoyv1alpha1.EnvoyConfig) error {
	if !ec.Status.Conditions.IsTrueFor(envoyv1alpha1.RollbackFailedCondition) {
		patch := client.MergeFrom(ec.DeepCopy())
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:    envoyv1alpha1.RollbackFailedCondition,
			Status:  corev1.ConditionTrue,
			Reason:  "AllRevisionsTainted",
			Message: "All revisions are tainted, rollback failed",
		})
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:    envoyv1alpha1.CacheOutOfSyncCondition,
			Status:  corev1.ConditionTrue,
			Reason:  "AllRevisionsTainted",
			Message: "All revisions are tainted, rollback failed",
		})
		ec.Status.CacheState = envoyv1alpha1.RollbackFailedState

		if err := r.Client.Status().Patch(ctx, ec, patch); err != nil {
			return err
		}
	}
	return nil
}

func (r *EnvoyConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&envoyv1alpha1.EnvoyConfig{}).
		Owns(&envoyv1alpha1.EnvoyConfigRevision{}).
		Complete(r)
}
