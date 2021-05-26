package reconcilers

import (
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	"github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/stats"
	"github.com/go-logr/logr"
)

// CleanupLogic executes finalization code for EnvoyConfigRevision resources
func CleanupLogic(ecr *marin3rv1alpha1.EnvoyConfigRevision, xdssCache xdss.Cache, discoveryStats *stats.Stats, log logr.Logger) {

	if ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
		discoveryStats.DeleteKeysByFilter(ecr.Spec.NodeID)
		xdssCache.ClearSnapshot(ecr.Spec.NodeID)
		log.Info("Successfully cleared xDS server cache", "XDSS", string(ecr.GetEnvoyAPIVersion()), "NodeID", ecr.Spec.NodeID)
	}
}
