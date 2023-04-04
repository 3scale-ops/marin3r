/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/3scale-ops/basereconciler/util"
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	"github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/stats"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_resources "github.com/3scale-ops/marin3r/pkg/envoy/resources"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	envoyconfigrevision "github.com/3scale-ops/marin3r/pkg/reconcilers/marin3r/envoyconfigrevision"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// EnvoyConfigRevisionReconciler reconciles a EnvoyConfigRevision object
type EnvoyConfigRevisionReconciler struct {
	Client         client.Client
	Log            logr.Logger
	Scheme         *runtime.Scheme
	XdsCache       xdss.Cache
	APIVersion     envoy.APIVersion
	DiscoveryStats *stats.Stats
}

// Reconcile progresses EnvoyConfigRevision resources to its desired state
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigrevisions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigrevisions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="core",namespace=placeholder,resources=secrets,verbs=get;list;watch
func (r *EnvoyConfigRevisionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("name", req.Name, "namespace", req.Namespace)

	// Fetch the EnvoyConfigRevision instance
	ecr := &marin3rv1alpha1.EnvoyConfigRevision{}
	err := r.Client.Get(ctx, req.NamespacedName, ecr)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	if ok := envoyconfigrevision.IsInitialized(ecr); !ok {
		if err := r.Client.Update(ctx, ecr); err != nil {
			log.Error(err, "unable to update EnvoyConfigRevision")
			return ctrl.Result{}, err
		}
		log.Info("initialized EnvoyConfigRevision resource")
		return reconcile.Result{}, nil
	}

	if util.IsBeingDeleted(ecr) {
		if !controllerutil.ContainsFinalizer(ecr, marin3rv1alpha1.EnvoyConfigRevisionFinalizer) {
			return reconcile.Result{}, nil
		}
		envoyconfigrevision.CleanupLogic(ecr, r.XdsCache, r.DiscoveryStats, log)
		controllerutil.RemoveFinalizer(ecr, marin3rv1alpha1.EnvoyConfigRevisionFinalizer)
		if err = r.Client.Update(ctx, ecr); err != nil {
			log.Error(err, "unable to update EnvoyConfigRevision")
			return reconcile.Result{}, err
		}
		log.Info("finalized EnvoyConfigRevision resource")
		return reconcile.Result{}, nil
	}

	var vt *marin3rv1alpha1.VersionTracker = nil

	// If this ecr has the RevisionPublishedCondition set to "True" pusblish the resources
	// to the xds server cache
	if meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionPublishedCondition) {
		var err error
		decoder := envoy_serializer.NewResourceUnmarshaller(ecr.GetSerialization(), r.APIVersion)

		cacheReconciler := envoyconfigrevision.NewCacheReconciler(
			ctx, log, r.Client, r.XdsCache,
			decoder,
			envoy_resources.NewGenerator(r.APIVersion),
		)

		vt, err = cacheReconciler.Reconcile(ctx, req.NamespacedName, ecr.Spec.EnvoyResources, ecr.Spec.NodeID, ecr.Spec.Version)

		// If a type errors.StatusError is returned it means that the config in spec.envoyResources is wrong
		// and cannot be written into the xDS cache. This is true for any error loading all types of resources
		// except for Secrets. Secrets are dynamically loaded from the API and transient failures are possible, so
		// setting a permanent taint could occur for a transient failure, which is not desirable.
		if err != nil {
			switch err.(type) {
			case *errors.StatusError:
				log.Error(err, fmt.Sprintf("%v", err))
				if err := r.taintSelf(ctx, ecr, "FailedLoadingResources", err.Error(), log); err != nil {
					return ctrl.Result{}, err
				}
			default:
				return ctrl.Result{}, err
			}
		}
	}

	if ok := envoyconfigrevision.IsStatusReconciled(ecr, vt, r.XdsCache, r.DiscoveryStats); !ok {
		if err := r.Client.Status().Update(ctx, ecr); err != nil {
			log.Error(err, "unable to update EnvoyConfigRevision status")
		}
		log.Info("status updated for EnvoyConfigRevision resource")
	}

	if meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionPublishedCondition) {
		return ctrl.Result{Requeue: true, RequeueAfter: 60 * time.Second}, nil
	}

	return ctrl.Result{}, nil

}

func (r *EnvoyConfigRevisionReconciler) taintSelf(ctx context.Context, ecr *marin3rv1alpha1.EnvoyConfigRevision,
	reason, msg string, log logr.Logger) error {

	if !meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionTaintedCondition) {
		patch := client.MergeFrom(ecr.DeepCopy())
		meta.SetStatusCondition(&ecr.Status.Conditions, metav1.Condition{
			Type:    marin3rv1alpha1.RevisionTaintedCondition,
			Status:  metav1.ConditionTrue,
			Reason:  reason,
			Message: msg,
		})
		ecr.Status.Tainted = pointer.BoolPtr(true)

		if err := r.Client.Status().Patch(ctx, ecr, patch); err != nil {
			return err
		}

		log.Info(fmt.Sprintf("Tainted revision: %q", msg))
	}
	return nil
}

func filterByAPIVersion(obj runtime.Object, version envoy.APIVersion) bool {
	switch o := obj.(type) {
	case *marin3rv1alpha1.EnvoyConfigRevision:
		if o.GetEnvoyAPIVersion() == version {
			return true
		}
		return false

	default:
		return true
	}
}

func filterByAPIVersionPredicate(version envoy.APIVersion,
	filter func(runtime.Object, envoy.APIVersion) bool) predicate.Predicate {

	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return filter(e.Object, version)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return filter(e.ObjectNew, version)

		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return filter(e.Object, version)
		},
	}
}

// SecretsEventHandler returns an EventHandler that generates
// reconcile requests for Secrets
func (r *EnvoyConfigRevisionReconciler) SecretsEventHandler() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(
		func(o client.Object) []reconcile.Request {
			secret := o.(*corev1.Secret)
			if secret.Type != corev1.SecretTypeTLS {
				return []reconcile.Request{}
			}
			list := &marin3rv1alpha1.EnvoyConfigRevisionList{}
			if err := r.Client.List(context.Background(), list); err != nil {
				return []reconcile.Request{}
			}

			reconcileRequests := []reconcile.Request{}

			for _, ecr := range list.Items {
				if meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionPublishedCondition) {
					// check if the Secret is relevant for this EnvoyConfigRevision
					for _, s := range ecr.Spec.EnvoyResources.Secrets {

						if (s.Ref != nil && s.Ref.Name == secret.GetName()) || (s.Name == secret.GetName()) {
							reconcileRequests = append(reconcileRequests,
								reconcile.Request{NamespacedName: types.NamespacedName{
									Name:      ecr.GetName(),
									Namespace: ecr.GetNamespace(),
								}})
						}
					}
				}
			}

			return reconcileRequests
		},
	)
}

// SetupWithManager adds the controller to the manager
func (r *EnvoyConfigRevisionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&marin3rv1alpha1.EnvoyConfigRevision{}).
		WithEventFilter(filterByAPIVersionPredicate(r.APIVersion, filterByAPIVersion)).
		Watches(&source.Kind{Type: &corev1.Secret{}}, r.SecretsEventHandler()).
		Complete(r)
}
