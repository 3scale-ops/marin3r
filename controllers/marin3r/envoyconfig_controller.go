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

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/common"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EnvoyConfigReconciler reconciles a EnvoyConfig object
type EnvoyConfigReconciler struct {
	Client client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=marin3r.3scale.net,resources=envoyconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=marin3r.3scale.net,resources=envoyconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=marin3r.3scale.net,resources=envoyconfigrevisions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=marin3r.3scale.net,resources=envoyconfigrevisions/status,verbs=get;update;patch

func (r *EnvoyConfigReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("name", req.Name, "namespace", req.Namespace)

	// Fetch the EnvoyConfig instance
	ec := &marin3rv1alpha1.EnvoyConfig{}
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

	// Set defaults.
	// TODO: remove this when wwe migrate to CRD v1 api as we will
	// be able to set defaults directly in the CRD definition.
	if err := r.reconcileSpecDefaults(ctx, ec, log); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileFinalizer(ctx, ec, log); err != nil {
		return ctrl.Result{}, err
	}

	// This must be done before any other revision related code due to the addition of support for
	// envoy API v3 which comes with the addition of a new label to identify revisions belonging
	// to each api version. The first time this version of the controller runs, it will set the
	// required labels. This will also help with future labels that get added.
	if err := r.reconcileRevisionLabels(ctx, ec, log); err != nil {
		return ctrl.Result{}, err
	}

	// desiredVersion is the version that matches the resources described in the spec
	desiredVersion := common.Hash(ec.Spec.EnvoyResources)

	// ensure that the desiredVersion has a matching revision object
	if err := r.ensureEnvoyConfigRevision(ctx, ec, desiredVersion, log); err != nil {
		return ctrl.Result{}, err
	}

	// Update the ConfigRevisions list in the status
	if err := r.reconcileRevisionList(ctx, ec, desiredVersion, log); err != nil {
		return ctrl.Result{}, err
	}

	// determine the version that should be published
	version, err := r.getVersionToPublish(ctx, ec, log)
	if err != nil {
		if err.(ControllerError).ErrorType == AllRevisionsTaintedError {
			if err := r.setRollbackFailed(ctx, ec, log); err != nil {
				return ctrl.Result{}, err
			}
			// This is an unrecoverable error because there are no
			// revisions to try and the controller cannot reconcile fix
			// this by . Set the RollbackFailedCOndition and exit without requeuing
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Mark the "version" as the published revision
	if err := r.markRevisionPublished(ctx, ec, version, "VersionPublished",
		fmt.Sprintf("Version '%s' has been published", version), log); err != nil {
		return ctrl.Result{}, err
	}

	// Update the status
	if err := r.updateStatus(ctx, ec, desiredVersion, version, log); err != nil {
		return ctrl.Result{}, err
	}

	// Cleanup unreferenced EnvoyConfigRevision objects
	if err := r.deleteUnreferencedRevisions(ctx, ec, log); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *EnvoyConfigReconciler) getVersionToPublish(ctx context.Context, ec *marin3rv1alpha1.EnvoyConfig, log logr.Logger) (string, error) {
	// Get the list of revisions for this nodeID
	ecrList := &marin3rv1alpha1.EnvoyConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID},
	})
	if err != nil {
		return "", NewControllerError(UnknownError, "getVersionToPublish", err.Error())
	}
	err = r.Client.List(ctx, ecrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return "", NewControllerError(UnknownError, "getVersionToPublish", err.Error())
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
	return "", NewControllerError(AllRevisionsTaintedError, "getVersionToPublish", "All available revisions are tainted")
}

func (r *EnvoyConfigReconciler) updateStatus(ctx context.Context, ec *marin3rv1alpha1.EnvoyConfig, desired, published string, log logr.Logger) error {

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
	} else if desired == published && ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.CacheOutOfSyncCondition) {
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
		if err := r.Client.Status().Patch(ctx, ec, patch); err != nil {
			return err
		}
	}

	return nil
}

// reconcileRevisionLabels ensures  all the EnvoyConfigRevisions owned by this EnvoyConfig have
// the appropriate labels. This is important as labels are extensively used to get the lists of
// EnvoyConfigRevision resources.
func (r *EnvoyConfigReconciler) reconcileRevisionLabels(ctx context.Context, ec *marin3rv1alpha1.EnvoyConfig, log logr.Logger) error {
	// Get all revisions for this EnvoyConfig
	ecrList := &marin3rv1alpha1.EnvoyConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			nodeIDTag: ec.Spec.NodeID,
		},
	})
	if err != nil {
		return NewControllerError(UnknownError, "reconcileRevisionLabels", err.Error())
	}
	err = r.Client.List(ctx, ecrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return NewControllerError(UnknownError, "reconcileRevisionLabels", err.Error())
	}

	for _, ecr := range ecrList.Items {
		_, okVersionTag := ecr.GetLabels()[versionTag]
		_, okEnvoyAPITag := ecr.GetLabels()[envoyAPITag]
		_, okNodeIDTag := ecr.GetLabels()[nodeIDTag]
		if !okVersionTag || !okEnvoyAPITag || !okNodeIDTag {
			log.Info("Reconciling labels for EnvoyConfigRevision", "Name", ecr.GetName(), "Namespace", ecr.GetNamespace())
			patch := client.MergeFrom(ecr.DeepCopy())
			ecr.SetLabels(map[string]string{
				versionTag:  ecr.Spec.Version,
				envoyAPITag: string(ec.GetEnvoyAPIVersion()),
				nodeIDTag:   ecr.Spec.NodeID,
			})
			if err := r.Client.Patch(ctx, &ecr, patch); err != nil {
				return NewControllerError(UnknownError, "reconcileRevisionLabels", err.Error())
			}
		}
	}
	return nil
}

func (r *EnvoyConfigReconciler) reconcileSpecDefaults(ctx context.Context, ec *marin3rv1alpha1.EnvoyConfig, log logr.Logger) error {
	changed := false

	if ec.Spec.EnvoyAPI == nil {
		ec.Spec.EnvoyAPI = pointer.StringPtr(string(ec.GetEnvoyAPIVersion()))
		changed = true
	}

	if ec.Spec.Serialization == nil {
		ec.Spec.Serialization = pointer.StringPtr(string(ec.GetSerialization()))
		changed = true
	}

	if changed {
		log.V(1).Info("setting EnvoyConfigRevision defaults")
		if err := r.Client.Update(ctx, ec); err != nil {
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

func (r *EnvoyConfigReconciler) setRollbackFailed(ctx context.Context, ec *marin3rv1alpha1.EnvoyConfig, log logr.Logger) error {
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

		if err := r.Client.Status().Patch(ctx, ec, patch); err != nil {
			return err
		}
	}
	return nil
}

func (r *EnvoyConfigReconciler) reconcileFinalizer(ctx context.Context, ec *marin3rv1alpha1.EnvoyConfig, log logr.Logger) error {

	if len(ec.GetObjectMeta().GetFinalizers()) != 0 {
		controllerutil.RemoveFinalizer(ec, marin3rv1alpha1.EnvoyConfigRevisionFinalizer)
		if err := r.Client.Update(ctx, ec); err != nil {
			return err
		}
	}
	return nil
}

func (r *EnvoyConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&marin3rv1alpha1.EnvoyConfig{}).
		Owns(&marin3rv1alpha1.EnvoyConfigRevision{}).
		Complete(r)
}
