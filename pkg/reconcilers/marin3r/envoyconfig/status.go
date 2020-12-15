package reconcilers

import (
	"reflect"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
)

// IsStatusReconciled calculates the status of the resource
func IsStatusReconciled(ec *marin3rv1alpha1.EnvoyConfig, publishedVersion string, list *marin3rv1alpha1.EnvoyConfigRevisionList) bool {

	ok := true

	revisionList := make([]marin3rv1alpha1.ConfigRevisionRef, len(list.Items))
	for idx, ecr := range list.Items {
		revisionList[idx] = marin3rv1alpha1.ConfigRevisionRef{
			Version: ecr.Spec.Version,
			Ref: corev1.ObjectReference{
				APIVersion: ecr.APIVersion,
				Kind:       ecr.Kind,
				Name:       ecr.GetName(),
				Namespace:  ecr.GetNamespace(),
				UID:        ecr.GetUID(),
			},
		}
	}
	if !reflect.DeepEqual(ec.Status.ConfigRevisions, revisionList) {
		ec.Status.ConfigRevisions = revisionList
		ok = false
	}

	desiredVersion := ec.GetEnvoyResourcesVersion()

	if ec.Status.PublishedVersion != publishedVersion {
		ec.Status.PublishedVersion = publishedVersion
		ok = false
	}

	if ec.Status.DesiredVersion != ec.GetEnvoyResourcesVersion() {
		ec.Status.DesiredVersion = desiredVersion
		ok = false
	}

	// Set the cacheStatus field
	if desiredVersion != publishedVersion && ec.Status.CacheState != marin3rv1alpha1.RollbackState {
		ec.Status.CacheState = marin3rv1alpha1.RollbackState
		ok = false
	}
	if desiredVersion == publishedVersion && ec.Status.CacheState != marin3rv1alpha1.InSyncState {
		ec.Status.CacheState = marin3rv1alpha1.InSyncState
		ok = false
	}

	// Set the CacheOutOfSyncCondition
	if desiredVersion != publishedVersion && !ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.CacheOutOfSyncCondition) {
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:    marin3rv1alpha1.CacheOutOfSyncCondition,
			Status:  corev1.ConditionTrue,
			Reason:  "CantPublishDesiredVersion",
			Message: "Desired resources spec cannot be applied",
		})
		ok = false
	} else if desiredVersion == publishedVersion && ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.CacheOutOfSyncCondition) {
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:    marin3rv1alpha1.CacheOutOfSyncCondition,
			Status:  corev1.ConditionFalse,
			Reason:  "DesiredVersionPublished",
			Message: "Desired version successfully published",
		})
		ok = false
	}

	// Clear the RollbackFailedCondition (if we have reached this code it means that
	// at least one untainted revision exists)
	if ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.RollbackFailedCondition) {
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:   marin3rv1alpha1.RollbackFailedCondition,
			Reason: "Recovered",
			Status: corev1.ConditionFalse,
		})
		ok = false
	}

	return ok
}
