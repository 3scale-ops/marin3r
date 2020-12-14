package reconcilers

import (
	"context"
	"fmt"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/envoy"
	"github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/errors"
	"github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/filters"
	"github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/revisions"
	"github.com/go-logr/logr"
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
	ctx     context.Context
	logger  logr.Logger
	client  client.Client
	scheme  *runtime.Scheme
	ec      *marin3rv1alpha1.EnvoyConfig
	version *string
}

// NewRevisionReconciler returns a new RevisionReconciler
func NewRevisionReconciler(ctx context.Context, logger logr.Logger, client client.Client,
	s *runtime.Scheme, ec *marin3rv1alpha1.EnvoyConfig) RevisionReconciler {

	return RevisionReconciler{ctx, logger, client, s, ec, nil}
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

// Reconcile progresses EnvoyConfig revisions to match the desired state. It does so
// by creating/updating/deleting EnvoyConfigRevision API resources.
func (r *RevisionReconciler) Reconcile() (ctrl.Result, error) {
	log := r.logger

	list, err := revisions.List(r.ctx, r.client, r.Namespace(), filters.ByNodeID(r.NodeID()))
	// At this point there might be no revisions yet
	if err != nil && !errors.IsNoMatchesForFilter(err) {
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
		// Revision labels updated, trigger a new reconcile loop
		return ctrl.Result{Requeue: true}, nil
	}

	_, err = revisions.Get(r.ctx, r.client, r.Namespace(),
		filters.ByNodeID(r.NodeID()), filters.ByVersion(r.Version()), filters.ByEnvoyAPI(r.EnvoyAPI()))
	if err != nil {
		if errors.IsNoMatchesForFilter(err) {
			ecr := r.NewRevisionForCurrentResources()
			if err := controllerutil.SetControllerReference(r.Instance(), ecr, r.scheme); err != nil {
				log.Error(err, "unable to SetControllerReference for new EnvoyConfigRevision resource", "Phase", "ReconcileRevisionForCurrentResources")
				return ctrl.Result{}, err
			}
			if err := r.client.Create(r.ctx, ecr); err != nil {
				log.Error(err, "unable to create EnvoyConfigRevision resource", "Phase", "ReconcileRevisionForCurrentResources")
				return ctrl.Result{}, err
			}
			// New EnvoyConfigRevision created, trigger a new reconcile loop
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
		log.Error(err, "unable to list revisions", "Phase", "ReconcileRevisionList")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// AreRevisionLabelsOk ensures all the EnvoyConfigRevisions owned by the EnvoyConfig have
// the appropriate labels. This is important as labels are extensively used to get the lists of
// EnvoyConfigRevision resources.
func (r *RevisionReconciler) AreRevisionLabelsOk(list *marin3rv1alpha1.EnvoyConfigRevisionList) bool {
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

// NewRevisionForCurrentResources generates an EnvoyConfigRevision resource for the current
// resources in the spec.EnvoyResources field of the EnvoyConfig resource
func (r *RevisionReconciler) NewRevisionForCurrentResources() *marin3rv1alpha1.EnvoyConfigRevision {
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
