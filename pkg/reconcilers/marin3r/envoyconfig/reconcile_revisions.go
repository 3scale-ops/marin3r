package reconcilers

import (
	"context"
	"fmt"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/common"
	"github.com/3scale/marin3r/pkg/envoy"
	"github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/filters"
	"github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/revisions"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	maxRevisions = 10
)

// RevisionReconciler is a struct with methods to reconcile EnvoyConfig revisions
type RevisionReconciler struct {
	ctx    context.Context
	logger logr.Logger
	client client.Client
	scheme *runtime.Scheme
	ec     *marin3rv1alpha1.EnvoyConfig

	// This fields are only available once Reconcile()
	// has been succesfully run
	desiredVersion   *string
	publishedVersion *string
	cacheState       *string
	revisionList     *marin3rv1alpha1.EnvoyConfigRevisionList
}

// NewRevisionReconciler returns a new RevisionReconciler
func NewRevisionReconciler(ctx context.Context, logger logr.Logger, client client.Client,
	s *runtime.Scheme, ec *marin3rv1alpha1.EnvoyConfig) RevisionReconciler {

	return RevisionReconciler{ctx, logger, client, s, ec, nil, nil, nil, nil}
}

// Instance returns the EnvoyConfig the reconciler has been instantiated with
func (r *RevisionReconciler) Instance() *marin3rv1alpha1.EnvoyConfig {
	return r.ec
}

// Namespace returns the Namespace of the EnvoyConfig the reconciler
// has been instantiated with
func (r *RevisionReconciler) Namespace() string {
	return r.Instance().GetNamespace()
}

// NodeID returns the nodeID of the EnvoyConfig the reconciler
// has been instantiated with
func (r *RevisionReconciler) NodeID() string {
	return r.Instance().Spec.NodeID
}

// DesiredVersion returns the version of the EnvoyConfig the reconciler
// has been instantiated with
func (r *RevisionReconciler) DesiredVersion() string {
	if r.desiredVersion == nil {
		// Store the version to avoid further computation of the same value
		r.desiredVersion = pointer.StringPtr(r.Instance().GetEnvoyResourcesVersion())
	}
	return *r.desiredVersion
}

// EnvoyAPI returns the envoy API version of the EnvoyConfig the reconciler
// has been instantiated with
func (r *RevisionReconciler) EnvoyAPI() envoy.APIVersion {
	return r.Instance().GetEnvoyAPIVersion()
}

// GetRevisionList returns the EnvoyConfigRevisionList that has been used to
// to computed by the Reconcile() function. If Reconcile has not been successfully
// invoked it will return nil.
func (r *RevisionReconciler) GetRevisionList() *marin3rv1alpha1.EnvoyConfigRevisionList {
	return r.revisionList
}

// PublishedVersion returns the version of the published revision.
// If Reconcile has not been successfully invoked it will return nil.
func (r *RevisionReconciler) PublishedVersion() string {
	return *r.publishedVersion
}

// GetCacheState returns the status of the EnvoyConfig the reconciler
// has been instantiated with. If Reconcile has not been successfully
// invoked it will return nil.
func (r *RevisionReconciler) GetCacheState() string {
	return *r.cacheState
}

// Reconcile progresses EnvoyConfig revisions to match the desired state. It does so
// by creating/updating/deleting EnvoyConfigRevision API resources.
func (r *RevisionReconciler) Reconcile() (ctrl.Result, error) {
	log := r.logger

	list, err := revisions.List(r.ctx, r.client, r.Namespace(), filters.ByNodeID(r.NodeID()))
	if err == nil {
		ok := r.areRevisionLabelsOk(list)
		if !ok {
			log.Info("reconciling revision labels")
			if err := r.client.Update(r.ctx, list); err != nil {
				log.Error(err, "unable to update revisions", "Phase", "ReconcileRevisionLabels")
				return ctrl.Result{}, err
			}
			// Revision labels updated, trigger a new reconcile loop
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		if !revisions.ErrorIsNoMatchesForFilter(err) {
			log.Error(err, "unable to list revisions", "Phase", "ReconcileRevisionLabels")
			return ctrl.Result{}, err
		}
	}

	_, err = revisions.Get(r.ctx, r.client, r.Namespace(),
		filters.ByNodeID(r.NodeID()), filters.ByVersion(r.DesiredVersion()), filters.ByEnvoyAPI(r.EnvoyAPI()))
	if err != nil {
		if revisions.ErrorIsNoMatchesForFilter(err) {
			ecr := r.newRevisionForCurrentResources()
			if err := controllerutil.SetControllerReference(r.Instance(), ecr, r.scheme); err != nil {
				log.Error(err, "unable to SetControllerReference for new EnvoyConfigRevision resource", "Phase", "ReconcileRevisionForCurrentResources")
				return ctrl.Result{}, err
			}
			if err := r.client.Create(r.ctx, ecr); err != nil {
				log.Error(err, "unable to create EnvoyConfigRevision resource", "Phase", "ReconcileRevisionForCurrentResources")
				return ctrl.Result{}, err
			}
			// New EnvoyConfigRevision created, trigger a new reconcile loop
			log.Info("created EnvoyConfigRevision for current resources", "version", r.DesiredVersion())
			return ctrl.Result{Requeue: true}, nil
		}
		if revisions.ErrorIsMultipleMatchesForFilter(err) {
			log.Error(err, "found more than one revision that matches current resources", "Phase", "ReconcileRevisionForCurrentResources")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	list, err = revisions.List(r.ctx, r.client, r.Namespace(), filters.ByNodeID(r.NodeID()), filters.ByEnvoyAPI(r.EnvoyAPI()))
	if err != nil {
		log.Error(err, "unable to list revisions", "Phase", "BuildRevisionList")
		return ctrl.Result{}, err
	}
	r.revisionList = revisions.SortByPublication(r.DesiredVersion(), list)
	publishedVersion, cacheState := r.getVersionToPublish()
	r.cacheState = &cacheState
	r.publishedVersion = &publishedVersion

	shouldBeTrue, shouldBeFalse := r.isRevisionPublishedConditionReconciled(r.PublishedVersion())

	if shouldBeFalse != nil {
		for _, ecr := range shouldBeFalse {
			if err := r.client.Status().Update(r.ctx, &ecr); err != nil {
				log.Error(err, "unable to update revision", "Phase", "UnpublishOldRevisions", "ObjectKey", common.ObjectKey(&ecr))
				return ctrl.Result{}, err
			}
			log.Info("updated the published EnvoyConfigRevision", "Namespace/Name", common.ObjectKey(&ecr))
		}
	}

	if shouldBeTrue != nil {
		if err := r.client.Status().Update(r.ctx, shouldBeTrue); err != nil {
			log.Error(err, "unable to update revision", "Phase", "PublishNewRevision", "ObjectKey", common.ObjectKey(shouldBeTrue))
			return ctrl.Result{}, err
		}
	}

	log.Info(fmt.Sprintf("CacheState is %s after revision reconcile", cacheState))
	return ctrl.Result{}, nil
}

// areRevisionLabelsOk ensures all the EnvoyConfigRevisions owned by the EnvoyConfig have
// the appropriate labels. This is important as labels are extensively used to get the lists of
// EnvoyConfigRevision resources.
func (r *RevisionReconciler) areRevisionLabelsOk(list *marin3rv1alpha1.EnvoyConfigRevisionList) bool {
	ok := true

	for _, ecr := range list.Items {
		_, okVersionTag := ecr.GetLabels()[filters.VersionTag]
		_, okEnvoyAPITag := ecr.GetLabels()[filters.EnvoyAPITag]
		_, okNodeIDTag := ecr.GetLabels()[filters.NodeIDTag]
		if !okVersionTag || !okEnvoyAPITag || !okNodeIDTag {

			ecr.SetLabels(map[string]string{
				filters.VersionTag:  ecr.Spec.Version,
				filters.EnvoyAPITag: ecr.GetEnvoyAPIVersion().String(),
				filters.NodeIDTag:   ecr.Spec.NodeID,
			})
			ok = false
		}
	}

	return ok
}

// getVersionToPublish takes an EnvoyConfigRevisionList and returns the revision that should be
// published. If the returned revision is not the one in the top position it returns the
// RollbackOccurredError error. If all of the revisions are tainted it returns the AllRevisionsTaintedError
// error.
func (r *RevisionReconciler) getVersionToPublish() (string, string) {
	var versionToPublish string

	topIdx := len(r.revisionList.Items) - 1

	// Starting from the highest index in the list and going
	// down, take the first version found that is not tainted
	for idx := topIdx; idx >= 0; idx-- {
		ecr := r.revisionList.Items[idx]
		if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionTaintedCondition) {
			versionToPublish = ecr.Spec.Version
			break
		}
	}

	if versionToPublish == "" {
		return "", marin3rv1alpha1.RollbackFailedState

	} else if versionToPublish != r.revisionList.Items[topIdx].Spec.Version {
		return versionToPublish, marin3rv1alpha1.RollbackState
	}

	return versionToPublish, marin3rv1alpha1.InSyncState

}

// isRevisionPublishedConditionReconciled returns the revisions that need the RevisionPublished condition reconciled.
// As the first return value returns the EnvoyConfigRevision that needs the condition set to true, nil if update
// not required. As the second return value returns a list of the EnvoyConfigRevisions that need the condition
// set to false, nil if no revision needs update.
func (r *RevisionReconciler) isRevisionPublishedConditionReconciled(versionToPublish string) (*marin3rv1alpha1.EnvoyConfigRevision,
	[]marin3rv1alpha1.EnvoyConfigRevision) {

	var shouldBeTrue *marin3rv1alpha1.EnvoyConfigRevision = nil
	var shouldBeFalse []marin3rv1alpha1.EnvoyConfigRevision = []marin3rv1alpha1.EnvoyConfigRevision{}

	for _, ecr := range r.revisionList.Items {

		if ecr.Spec.Version != versionToPublish && ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			ecr.Status.Conditions.RemoveCondition(marin3rv1alpha1.RevisionPublishedCondition)
			shouldBeFalse = append(shouldBeFalse, ecr)

		} else if ecr.Spec.Version == versionToPublish && !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			ecr.Status.Conditions.SetCondition(status.Condition{
				Type:    marin3rv1alpha1.RevisionPublishedCondition,
				Status:  corev1.ConditionTrue,
				Reason:  status.ConditionReason("VersionPublished"),
				Message: fmt.Sprintf("Version '%s' has been published", versionToPublish),
			})
			shouldBeTrue = &ecr
		}
	}

	if len(shouldBeFalse) == 0 {
		shouldBeFalse = nil
	}
	return shouldBeTrue, shouldBeFalse
}

// newRevisionForCurrentResources generates an EnvoyConfigRevision resource for the current
// resources in the spec.EnvoyResources field of the EnvoyConfig resource
func (r *RevisionReconciler) newRevisionForCurrentResources() *marin3rv1alpha1.EnvoyConfigRevision {
	return &marin3rv1alpha1.EnvoyConfigRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name: func() string {
				if r.EnvoyAPI() == envoy.APIv2 {
					return fmt.Sprintf("%s-%s", r.NodeID(), r.DesiredVersion())
				}
				return fmt.Sprintf("%s-%s-%s", r.NodeID(), r.EnvoyAPI(), r.DesiredVersion())
			}(),
			Namespace: r.Namespace(),
			Labels: map[string]string{
				filters.NodeIDTag:   r.NodeID(),
				filters.VersionTag:  r.DesiredVersion(),
				filters.EnvoyAPITag: r.EnvoyAPI().String(),
			},
		},
		Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
			NodeID:         r.NodeID(),
			EnvoyAPI:       pointer.StringPtr(r.EnvoyAPI().String()),
			Version:        r.DesiredVersion(),
			Serialization:  r.Instance().Spec.Serialization,
			EnvoyResources: r.Instance().Spec.EnvoyResources,
		},
	}
}
