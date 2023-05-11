package reconcilers

import (
	"fmt"
	"math"
	"reflect"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	"github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/stats"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_resources "github.com/3scale-ops/marin3r/pkg/envoy/resources"
	k8sutil "github.com/3scale-ops/marin3r/pkg/util/k8s"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// IsStatusReconciled calculates the status of the resource
func IsStatusReconciled(ecr *marin3rv1alpha1.EnvoyConfigRevision, vt *marin3rv1alpha1.VersionTracker, xdssCache xdss.Cache, dStats *stats.Stats) bool {

	ok := true

	if vt != nil && (ecr.Status.ProvidesVersions == nil || !reflect.DeepEqual(ecr.Status.ProvidesVersions, vt)) {
		ecr.Status.ProvidesVersions = vt
		ok = false
	}

	// Note: tainted condition is never automatically removed to avoid retrying a bad config in the case of
	// loss of statistics (i.e. a restart)
	var taintedCond *metav1.Condition
	if vt != nil {
		taintedCond = calculateRevisionTaintedCondition(ecr, ecr.Status.ProvidesVersions, dStats, 1)
	}

	if taintedCond != nil {
		equal := k8sutil.ConditionsEqual(taintedCond, meta.FindStatusCondition(ecr.Status.Conditions, marin3rv1alpha1.RevisionTaintedCondition))
		if !equal {
			meta.SetStatusCondition(&ecr.Status.Conditions, *taintedCond)
			ok = false
		}
	}

	inSyncCond := calculateResourcesInSyncCondition(ecr, xdssCache)
	if inSyncCond != nil {
		equal := k8sutil.ConditionsEqual(inSyncCond, meta.FindStatusCondition(ecr.Status.Conditions, marin3rv1alpha1.ResourcesInSyncCondition))
		if !equal {
			meta.SetStatusCondition(&ecr.Status.Conditions, *inSyncCond)
			ok = false
		}
	} else {
		if meta.FindStatusCondition(ecr.Status.Conditions, marin3rv1alpha1.ResourcesInSyncCondition) != nil {
			meta.RemoveStatusCondition(&ecr.Status.Conditions, marin3rv1alpha1.ResourcesInSyncCondition)
			ok = false
		}
	}

	// Set status.published and status.lastPublishedAt fields
	if meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionPublishedCondition) && !ecr.Status.IsPublished() {
		ecr.Status.Published = pointer.Bool(true)
		ecr.Status.LastPublishedAt = func(t metav1.Time) *metav1.Time { return &t }(metav1.Now())
		ok = false
	} else if !meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionPublishedCondition) && ecr.Status.IsPublished() {
		ecr.Status.Published = pointer.Bool(false)
		ok = false
	}

	// Set status.tainted field
	if meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionTaintedCondition) && !ecr.Status.IsTainted() {
		ecr.Status.Tainted = pointer.Bool(true)
		ok = false
	} else if !meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionTaintedCondition) && ecr.Status.IsTainted() {
		ecr.Status.Tainted = pointer.Bool(false)
		ok = false
	}

	return ok
}

func calculateResourcesInSyncCondition(ecr *marin3rv1alpha1.EnvoyConfigRevision, xdssCache xdss.Cache) *metav1.Condition {

	if meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionPublishedCondition) {
		// Check what is currently written in the xds server cache
		_, err := xdssCache.GetSnapshot(ecr.Spec.NodeID)
		// OutOfSync if NodeID not found or resources version different that expected
		if err != nil {
			return &metav1.Condition{
				Type:    marin3rv1alpha1.ResourcesInSyncCondition,
				Reason:  "SnapshotDoesNotExist",
				Status:  metav1.ConditionFalse,
				Message: fmt.Sprintf("A snapshot for nodeID %q does not yet exist in the xDS server cache", ecr.Spec.NodeID),
			}
		}

		return &metav1.Condition{
			Type:    marin3rv1alpha1.ResourcesInSyncCondition,
			Reason:  "ResourcesSynced",
			Status:  metav1.ConditionTrue,
			Message: "EnvoyConfigRevision resources successfully synced with xDS server cache",
		}
	}

	return nil
}

func calculateRevisionTaintedCondition(ecr *marin3rv1alpha1.EnvoyConfigRevision, vt *marin3rv1alpha1.VersionTracker, dStats *stats.Stats, threshold float64) *metav1.Condition {

	if dStats.GetPercentageFailing(ecr.Spec.NodeID, envoy_resources.TypeURL(envoy.Endpoint, ecr.GetEnvoyAPIVersion()), vt.Endpoints) == threshold ||
		dStats.GetPercentageFailing(ecr.Spec.NodeID, envoy_resources.TypeURL(envoy.Cluster, ecr.GetEnvoyAPIVersion()), vt.Clusters) == threshold ||
		dStats.GetPercentageFailing(ecr.Spec.NodeID, envoy_resources.TypeURL(envoy.Route, ecr.GetEnvoyAPIVersion()), vt.Routes) == threshold ||
		dStats.GetPercentageFailing(ecr.Spec.NodeID, envoy_resources.TypeURL(envoy.ScopedRoute, ecr.GetEnvoyAPIVersion()), vt.ScopedRoutes) == threshold ||
		dStats.GetPercentageFailing(ecr.Spec.NodeID, envoy_resources.TypeURL(envoy.Listener, ecr.GetEnvoyAPIVersion()), vt.Listeners) == threshold ||
		dStats.GetPercentageFailing(ecr.Spec.NodeID, envoy_resources.TypeURL(envoy.Secret, ecr.GetEnvoyAPIVersion()), vt.Secrets) == threshold ||
		dStats.GetPercentageFailing(ecr.Spec.NodeID, envoy_resources.TypeURL(envoy.Runtime, ecr.GetEnvoyAPIVersion()), vt.Runtimes) == threshold ||
		dStats.GetPercentageFailing(ecr.Spec.NodeID, envoy_resources.TypeURL(envoy.ExtensionConfig, ecr.GetEnvoyAPIVersion()), vt.ExtensionConfigs) == threshold {
		return &metav1.Condition{
			Type:    marin3rv1alpha1.RevisionTaintedCondition,
			Reason:  "ResourcesFailing",
			Status:  metav1.ConditionTrue,
			Message: fmt.Sprintf("EnvoyConfigRevision resources are being rejected by more than %d%% of the Envoy clients", int(math.Round(threshold*100))),
		}
	}

	return nil
}
