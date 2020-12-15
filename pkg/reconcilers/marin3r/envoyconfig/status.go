package reconcilers

import (
	"reflect"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
)

// IsStatusReconciled calculates the status of the resource
func IsStatusReconciled(ec *marin3rv1alpha1.EnvoyConfig, cacheState, publishedVersion string, list *marin3rv1alpha1.EnvoyConfigRevisionList) bool {

	ok := true

	revisionList := generateRevisionList(list)
	if !reflect.DeepEqual(ec.Status.ConfigRevisions, revisionList) {
		ec.Status.ConfigRevisions = revisionList
		ok = false
	}

	desiredVersion := ec.GetEnvoyResourcesVersion()

	if ec.Status.PublishedVersion != publishedVersion {
		ec.Status.PublishedVersion = publishedVersion
		ok = false
	}

	if ec.Status.DesiredVersion != desiredVersion {
		ec.Status.DesiredVersion = desiredVersion
		ok = false
	}

	if ec.Status.CacheState != cacheState {
		ec.Status.CacheState = cacheState
		ok = false
	}

	// Reconcile the CacheOutOfSyncCondition
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

	// Reconcile the RollbackFailedCondition
	if cacheState != marin3rv1alpha1.RollbackFailedState && ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.RollbackFailedCondition) {
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:   marin3rv1alpha1.RollbackFailedCondition,
			Reason: "Recovered",
			Status: corev1.ConditionFalse,
		})
		ok = false

	} else if cacheState == marin3rv1alpha1.RollbackFailedState && !ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.RollbackFailedCondition) {
		ec.Status.Conditions.SetCondition(status.Condition{
			Type:    marin3rv1alpha1.RollbackFailedCondition,
			Status:  corev1.ConditionTrue,
			Reason:  "AllRevisionsTainted",
			Message: "All revisions are tainted, rollback failed",
		})
		ok = false
	}

	return ok
}

func generateRevisionList(list *marin3rv1alpha1.EnvoyConfigRevisionList) []marin3rv1alpha1.ConfigRevisionRef {

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

	return revisionList
}
