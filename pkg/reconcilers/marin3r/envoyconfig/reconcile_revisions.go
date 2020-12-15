package reconcilers

import (
	"context"
	"fmt"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/common"
	"github.com/3scale/marin3r/pkg/envoy"
	"github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/errors"
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
	ctx            context.Context
	logger         logr.Logger
	client         client.Client
	scheme         *runtime.Scheme
	ec             *marin3rv1alpha1.EnvoyConfig
	version        *string
	desiredVersion *string
	revisionList   *marin3rv1alpha1.EnvoyConfigRevisionList
}

// NewRevisionReconciler returns a new RevisionReconciler
func NewRevisionReconciler(ctx context.Context, logger logr.Logger, client client.Client,
	s *runtime.Scheme, ec *marin3rv1alpha1.EnvoyConfig) RevisionReconciler {

	return RevisionReconciler{ctx, logger, client, s, ec, nil, nil, nil}
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

// Version returns the version of the EnvoyConfig the reconciler
// has been instantiated with
func (r *RevisionReconciler) Version() string {
	if r.version == nil {
		// Store the version to avoid further computation of the same value
		r.version = pointer.StringPtr(r.Instance().GetEnvoyResourcesVersion())
	}
	return *r.version
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
		if !errors.IsNoMatchesForFilter(err) {
			log.Error(err, "unable to list revisions", "Phase", "ReconcileRevisionLabels")
			return ctrl.Result{}, err
		}
	}

	_, err = revisions.Get(r.ctx, r.client, r.Namespace(),
		filters.ByNodeID(r.NodeID()), filters.ByVersion(r.Version()), filters.ByEnvoyAPI(r.EnvoyAPI()))
	if err != nil {
		if errors.IsNoMatchesForFilter(err) {
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
			log.Info("created EnvoyConfigRevision for current resources", "version", r.Version())
			return ctrl.Result{Requeue: true}, nil
		}
		if errors.IsMultipleMatchesForFilter(err) {
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
	r.revisionList = revisions.SortByPublication(r.Version(), list)
	versionToPublish, err := r.getVersionToPublish()
	if err != nil {
		log.Error(err, "unable to reconcile revisions, all tainted", "Phase", "CalculateRevisionToPublish")
		return ctrl.Result{}, err
	}

	shouldBeTrue, shouldBeFalse := r.isRevisionPublishedConditionReconciled(versionToPublish)

	if shouldBeFalse != nil {
		for _, ecr := range shouldBeFalse {
			if err := r.client.Status().Update(r.ctx, &ecr); err != nil {
				log.Error(err, "unable to update revision", "Phase", "UnpublishOldRevisions", "ObjectKey", common.ObjectKey(&ecr))
				return ctrl.Result{}, err
			}
		}
	}

	if shouldBeTrue != nil {
		if err := r.client.Status().Update(r.ctx, shouldBeTrue); err != nil {
			log.Error(err, "unable to update revision", "Phase", "PublishNewRevision", "ObjectKey", common.ObjectKey(shouldBeTrue))
			return ctrl.Result{}, err
		}
	}

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

// newRevisionForCurrentResources generates an EnvoyConfigRevision resource for the current
// resources in the spec.EnvoyResources field of the EnvoyConfig resource
func (r *RevisionReconciler) newRevisionForCurrentResources() *marin3rv1alpha1.EnvoyConfigRevision {
	return &marin3rv1alpha1.EnvoyConfigRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name: func() string {
				if r.EnvoyAPI() == envoy.APIv2 {
					return fmt.Sprintf("%s-%s", r.NodeID(), r.Version())
				}
				return fmt.Sprintf("%s-%s-%s", r.NodeID(), r.EnvoyAPI(), r.Version())
			}(),
			Namespace: r.Namespace(),
			Labels: map[string]string{
				filters.NodeIDTag:   r.NodeID(),
				filters.VersionTag:  r.Version(),
				filters.EnvoyAPITag: r.EnvoyAPI().String(),
			},
		},
		Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
			NodeID:         r.NodeID(),
			EnvoyAPI:       pointer.StringPtr(r.EnvoyAPI().String()),
			Version:        r.Version(),
			Serialization:  r.Instance().Spec.Serialization,
			EnvoyResources: r.Instance().Spec.EnvoyResources,
		},
	}
}

// getVersionToPublish takes an EnvoyConfigRevisionList and returns the revision that should be
// published
func (r *RevisionReconciler) getVersionToPublish() (string, error) {

	// Starting from the highest index in the list and going
	// down, return the first version found that is not tainted
	for i := len(r.revisionList.Items) - 1; i >= 0; i-- {
		ecr := r.revisionList.Items[i]
		if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionTaintedCondition) {
			return ecr.Spec.Version, nil
		}
	}

	// If we get here it means that there is not untainted revision. Return a specific
	// error to the controller loop so it gets handled appropriately
	return "", errors.New(errors.AllRevisionsTaintedError, "GetVersionToPublish", "All available revisions are tainted")
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
