package reconcilers

import (
	"context"
	"fmt"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/envoy"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/marin3r/envoyconfig/filters"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/marin3r/envoyconfig/revisions"
	"github.com/3scale-ops/marin3r/pkg/util"
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

	desiredRev, err := revisions.Get(r.ctx, r.client, r.Namespace(),
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
	} else {
		// The desired rev already exists so it had been published at some
		// point, but it might not be the rev that SortByPublication() and
		// getVersionToPublish() will choose so "touch" its published
		// timestamp to force them to choose it.
		desiredRev.Status.LastPublishedAt = func(t metav1.Time) *metav1.Time { return &t }(metav1.Now())
		if err := desiredRev.UpdateStatus(r.ctx, r.client, log); err != nil {
			return ctrl.Result{}, err
		}
	}
	log.V(1).Info("Reconcile", "desired version", r.DesiredVersion(), "error", nil)

	list, err := revisions.List(r.ctx, r.client, r.Namespace(), filters.ByNodeID(r.NodeID()), filters.ByEnvoyAPI(r.EnvoyAPI()))
	if err != nil {
		log.Error(err, "unable to list revisions", "Phase", "BuildRevisionList")
		return ctrl.Result{}, err
	}
	r.revisionList = revisions.SortByPublication(r.DesiredVersion(), list)
	publishedVersion, cacheState := r.getVersionToPublish()
	r.cacheState = &cacheState
	r.publishedVersion = &publishedVersion

	shouldBeTrue, shouldBeFalse := r.isRevisionPublishedConditionReconciled(r.PublishedVersion())

	for _, ecr := range shouldBeFalse {
		if err := ecr.UpdateStatus(r.ctx, r.client, log); err != nil {
			return ctrl.Result{}, err
		}
	}

	if shouldBeTrue != nil {
		if err := shouldBeTrue.UpdateStatus(r.ctx, r.client, log); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("updated the published EnvoyConfigRevision", "Namespace/Name", util.ObjectKey(shouldBeTrue))
	}

	shouldBeDeleted := r.isRevisionRetentionReconciled(maxRevisions)
	for _, ecr := range shouldBeDeleted {
		if err := r.client.Delete(r.ctx, &ecr); err != nil {
			log.Error(err, "unable to delete revision", "Phase", "ApplyRevisionRetention", "Name/Namespace", util.ObjectKey(&ecr))
			return ctrl.Result{}, err
		}
		log.Info("deleted old EnvoyConfigRevision", "Namespace/Name", util.ObjectKey(&ecr))
	}

	log.Info(fmt.Sprintf("CacheState is %s after revision reconcile", cacheState))
	return ctrl.Result{}, nil
}

// getVersionToPublish takes an EnvoyConfigRevisionList and returns the version that should be
// published. It also returns the state of the cache based on the position of the revision
// with the returned version in the list of revisions.
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
			shouldBeTrue = ecr.DeepCopy()
		}
	}

	if len(shouldBeFalse) == 0 {
		shouldBeFalse = nil
	}
	return shouldBeTrue, shouldBeFalse
}

// isRevisionRetentionReconciled removes items from the revisionList until the list holds the number of items
// determined by the 'retention' parameter
func (r *RevisionReconciler) isRevisionRetentionReconciled(retention int) []marin3rv1alpha1.EnvoyConfigRevision {

	var toBeDeleted []marin3rv1alpha1.EnvoyConfigRevision = []marin3rv1alpha1.EnvoyConfigRevision{}
	var revisionList *[]marin3rv1alpha1.EnvoyConfigRevision = &(r.GetRevisionList().Items)

	for len(*revisionList) > retention {
		toBeDeleted = append(toBeDeleted, popRevision(revisionList))
	}

	return toBeDeleted
}

// popRevision removes an EnvoyConfigRevision from a list of EnvoyConfigRevision resources, starting from
// the lowest index in the slice. The removed element is returned as a return value and the list is
// modified "in place".
func popRevision(list *[]marin3rv1alpha1.EnvoyConfigRevision) marin3rv1alpha1.EnvoyConfigRevision {
	item := (*list)[0]
	*list = (*list)[1:]
	return item
}

// newRevisionForCurrentResources generates an EnvoyConfigRevision resource for the current
// resources in the spec.EnvoyResources field of the EnvoyConfig resource
func (r *RevisionReconciler) newRevisionForCurrentResources() *marin3rv1alpha1.EnvoyConfigRevision {
	return &marin3rv1alpha1.EnvoyConfigRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", r.NodeID(), r.EnvoyAPI(), r.DesiredVersion()),
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
			Serialization:  pointer.StringPtr(string(r.Instance().GetSerialization())),
			EnvoyResources: r.Instance().Spec.EnvoyResources,
		},
	}
}
