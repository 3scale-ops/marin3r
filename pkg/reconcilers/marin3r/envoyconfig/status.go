package reconcilers

import (
	"reflect"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if ec.Status.PublishedVersion == nil || *ec.Status.PublishedVersion != publishedVersion {
		ec.Status.PublishedVersion = &publishedVersion
		ok = false
	}

	if ec.Status.DesiredVersion == nil || *ec.Status.DesiredVersion != desiredVersion {
		ec.Status.DesiredVersion = &desiredVersion
		ok = false
	}

	if ec.Status.CacheState == nil || *ec.Status.CacheState != cacheState {
		ec.Status.CacheState = &cacheState
		ok = false
	}

	if ec.Status.Conditions == nil {
		ec.Status.Conditions = []metav1.Condition{}
		ok = false
	}

	// Reconcile the CacheOutOfSyncCondition
	if desiredVersion != publishedVersion && !meta.IsStatusConditionTrue(ec.Status.Conditions, marin3rv1alpha1.CacheOutOfSyncCondition) {
		meta.SetStatusCondition(&ec.Status.Conditions, metav1.Condition{
			Type:    marin3rv1alpha1.CacheOutOfSyncCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "CantPublishDesiredVersion",
			Message: "Desired resources spec cannot be applied",
		})
		ok = false

	} else if desiredVersion == publishedVersion && meta.IsStatusConditionTrue(ec.Status.Conditions, marin3rv1alpha1.CacheOutOfSyncCondition) {
		meta.SetStatusCondition(&ec.Status.Conditions, metav1.Condition{
			Type:    marin3rv1alpha1.CacheOutOfSyncCondition,
			Status:  metav1.ConditionFalse,
			Reason:  "DesiredVersionPublished",
			Message: "Desired version successfully published",
		})
		ok = false
	}

	// Reconcile the RollbackFailedCondition
	if cacheState != marin3rv1alpha1.RollbackFailedState && meta.IsStatusConditionTrue(ec.Status.Conditions, marin3rv1alpha1.RollbackFailedCondition) {
		meta.SetStatusCondition(&ec.Status.Conditions, metav1.Condition{
			Type:    marin3rv1alpha1.RollbackFailedCondition,
			Reason:  "Recovered",
			Status:  metav1.ConditionFalse,
			Message: "Recovered from RollbackFailed condition",
		})
		ok = false

	} else if cacheState == marin3rv1alpha1.RollbackFailedState && !meta.IsStatusConditionTrue(ec.Status.Conditions, marin3rv1alpha1.RollbackFailedCondition) {
		meta.SetStatusCondition(&ec.Status.Conditions, metav1.Condition{
			Type:    marin3rv1alpha1.RollbackFailedCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "AllRevisionsTainted",
			Message: "All revisions are tainted, rollback failed",
		})
		ok = false
	}

	// Temporary fix for RollbackFailedCondition conditions that are missing  the .Message property, which
	// will be required in an upcoming release
	if cond := meta.FindStatusCondition(ec.Status.Conditions, marin3rv1alpha1.RollbackFailedCondition); cond != nil && cond.Message == "" {
		cond.Message = "Recovered from RollbackFailed condition"
		meta.SetStatusCondition(&ec.Status.Conditions, *cond)
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
