package reconcilers

import (
	"context"
	"fmt"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/envoy"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
}

// NewRevisionReconciler returns a new RevisionReconciler
func NewRevisionReconciler(ctx context.Context, logger logr.Logger, client client.Client,
	s *runtime.Scheme, ec *marin3rv1alpha1.EnvoyConfig) RevisionReconciler {

	return RevisionReconciler{ctx, logger, client, s, ec}
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
	return r.Instance().GetEnvoyResourcesVersion()
}

// EnvoyAPI returns the envoy API version of the EnvoyConfig the reconciler
// has been instantiated with
func (r *RevisionReconciler) EnvoyAPI() envoy.APIVersion {
	return r.Instance().GetEnvoyAPIVersion()
}

// ListRevisions returns the list of EnvoyConfigRevisions owned by the EnvoyConfig
// the reconciler has been instantiated with
func (r *RevisionReconciler) ListRevisions(filters ...RevisionFilter) (*marin3rv1alpha1.EnvoyConfigRevisionList, error) {

	list := &marin3rv1alpha1.EnvoyConfigRevisionList{}

	labelSelector := client.MatchingLabels{}
	for _, filter := range filters {
		filter.ApplyToLabelSelector(labelSelector)
	}

	if err := r.client.List(r.ctx, list, labelSelector, client.InNamespace(r.Instance().GetNamespace())); err != nil {
		return nil, err
	}

	return list, nil
}

// GetRevision returns the EnvoyConfigRevision that matches the provided filters. If no EnvoyConfigRevisions are returned
// by the API an error is returned. If more than one EnvoyConfigRevision are returned by the API an error is returned.
func (r *RevisionReconciler) GetRevision(filters ...RevisionFilter) (*marin3rv1alpha1.EnvoyConfigRevision, error) {

	list := &marin3rv1alpha1.EnvoyConfigRevisionList{}

	labelSelector := client.MatchingLabels{}
	for _, filter := range filters {
		filter.ApplyToLabelSelector(labelSelector)
	}

	if err := r.client.List(r.ctx, list, labelSelector, client.InNamespace(r.Instance().GetNamespace())); err != nil {
		return nil, err
	}

	if len(list.Items) != 1 {
		return nil, fmt.Errorf("api returned %d EnvoyConfigRevisions", len(list.Items))
	}

	return &list.Items[0], nil
}

// Reconcile progresses EnvoyConfig revisions to match the desired state. It does so
// by creating/updating/deleting EnvoyConfigRevision API resources.
func (r *RevisionReconciler) Reconcile() (ctrl.Result, error) {
	log := r.logger

	list, err := r.ListRevisions(FilterByNodeID(r.NodeID()))
	if err != nil {
		log.Error(err, "unable to list revisions", "Phase", "ReconcileRevisionLabels")
		return ctrl.Result{}, err
	}

	ok := r.AreRevisionLabelsOk(list)
	if !ok {
		log.Info("reconciling revision labels")
		if err := r.client.Update(r.ctx, list); err != nil {
			log.Error(err, "unable to update revisions", "Phase", "ReconcileRevisionLabels")
			return ctrl.Result{}, err
		}
		// Revisions labels updated, trigger a new reconcile loop
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// AreRevisionLabelsOk ensures all the EnvoyConfigRevisions owned by the EnvoyConfig have
// the appropriate labels. This is important as labels are extensively used to get the lists of
// EnvoyConfigRevision resources.
func (r *RevisionReconciler) AreRevisionLabelsOk(list *marin3rv1alpha1.EnvoyConfigRevisionList) bool {
	ok := true

	for _, ecr := range list.Items {
		_, okVersionTag := ecr.GetLabels()[versionTag]
		_, okEnvoyAPITag := ecr.GetLabels()[envoyAPITag]
		_, okNodeIDTag := ecr.GetLabels()[nodeIDTag]
		if !okVersionTag || !okEnvoyAPITag || !okNodeIDTag {

			ecr.SetLabels(map[string]string{
				versionTag:  ecr.Spec.Version,
				envoyAPITag: ecr.GetEnvoyAPIVersion().String(),
				nodeIDTag:   ecr.Spec.NodeID,
			})
			ok = false
		}
	}

	return ok
}
