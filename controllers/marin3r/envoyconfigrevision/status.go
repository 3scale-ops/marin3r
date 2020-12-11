package controllers

import (
	"fmt"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	xdss "github.com/3scale/marin3r/pkg/discoveryservice/xdss"
	"github.com/3scale/marin3r/pkg/envoy"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// IsStatusReconciled calculates the status of the resource
func IsStatusReconciled(ecr *marin3rv1alpha1.EnvoyConfigRevision, xdssCache xdss.Cache) bool {

	ok := true

	ooSyncCond := calculateOutOfSyncCondition(ecr, xdssCache)
	if ooSyncCond != nil {
		equal := equality.Semantic.DeepEqual(ecr.Status.Conditions.GetCondition(marin3rv1alpha1.ResourcesOutOfSyncCondition), ooSyncCond)
		if !equal {
			ecr.Status.Conditions.SetCondition(*ooSyncCond)
			ok = false
		}
	} else {
		if ecr.Status.Conditions.GetCondition(marin3rv1alpha1.ResourcesOutOfSyncCondition) != nil {
			ecr.Status.Conditions.RemoveCondition(marin3rv1alpha1.ResourcesOutOfSyncCondition)
			ok = false
		}
	}

	// Set status.published and status.lastPublishedAt fields
	if ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) && !ecr.Status.IsPublished() {
		ecr.Status.Published = pointer.BoolPtr(true)
		ecr.Status.LastPublishedAt = func(t metav1.Time) *metav1.Time { return &t }(metav1.Now())
		ok = false
	} else if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) && ecr.Status.IsPublished() {
		ecr.Status.Published = pointer.BoolPtr(false)
		ok = false
	}

	// Set status.tainted field
	if ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionTaintedCondition) && !ecr.Status.IsTainted() {
		ecr.Status.Tainted = pointer.BoolPtr(true)
		ok = false
	} else if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionTaintedCondition) && ecr.Status.IsTainted() {
		ecr.Status.Tainted = pointer.BoolPtr(false)
		ok = false
	}

	return ok
}

func calculateOutOfSyncCondition(ecr *marin3rv1alpha1.EnvoyConfigRevision, xdssCache xdss.Cache) *status.Condition {

	if ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
		// Check what is currently written in the xds server cache
		snap, err := xdssCache.GetSnapshot(ecr.Spec.NodeID)
		// OutOfSync if NodeID not found or resources version different that expected
		if err != nil {
			return &status.Condition{
				Type:    marin3rv1alpha1.ResourcesOutOfSyncCondition,
				Reason:  "SnapshotDoesNotExist",
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("A snapshot for nodeID %q does not yet exist in the xDS server cache", ecr.Spec.NodeID),
			}
		}

		if snap.GetVersion(envoy.Cluster) != ecr.Spec.Version {
			return &status.Condition{
				Type:    marin3rv1alpha1.ResourcesOutOfSyncCondition,
				Reason:  "SnapshotVersionDiffers",
				Status:  corev1.ConditionTrue,
				Message: fmt.Sprintf("The snapshot for nodeID %q holds resources version %q", ecr.Spec.NodeID, snap.GetVersion(envoy.Cluster)),
			}
		}

		return &status.Condition{
			Type:    marin3rv1alpha1.ResourcesOutOfSyncCondition,
			Reason:  "EnvoyConficRevisionResourcesSynced",
			Status:  corev1.ConditionFalse,
			Message: "EnvoyConfigRevision resources successfully synced with xDS server cache",
		}
	}

	return nil
}
