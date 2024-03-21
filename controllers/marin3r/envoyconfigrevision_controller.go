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

	"github.com/3scale-ops/basereconciler/reconciler"
	reconciler_util "github.com/3scale-ops/basereconciler/util"
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	"github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/stats"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_resources "github.com/3scale-ops/marin3r/pkg/envoy/resources"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	envoyconfigrevision "github.com/3scale-ops/marin3r/pkg/reconcilers/marin3r/envoyconfigrevision"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// EnvoyConfigRevisionReconciler reconciles a EnvoyConfigRevision object
type EnvoyConfigRevisionReconciler struct {
	*reconciler.Reconciler
	XdsCache       xdss.Cache
	APIVersion     envoy.APIVersion
	DiscoveryStats *stats.Stats
}

// Reconcile progresses EnvoyConfigRevision resources to its desired state
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigrevisions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigrevisions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="core",namespace=placeholder,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="discovery.k8s.io",namespace=placeholder,resources=endpointslices,verbs=get;list;watch
func (r *EnvoyConfigRevisionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	ctx, logger := r.Logger(ctx, "name", req.Name, "namespace", req.Namespace)
	ecr := &marin3rv1alpha1.EnvoyConfigRevision{}
	result := r.ManageResourceLifecycle(ctx, req, ecr,
		// Apply defaults
		reconciler.WithInitializationFunc(reconciler_util.ResourceDefaulter(ecr)),
		// convert spec.EnvoyResources to spec.Resources
		reconciler.WithInMemoryInitializationFunc(func(ctx context.Context, c client.Client, o client.Object) error {
			if ecr.Spec.EnvoyResources != nil {
				ecr := o.(*marin3rv1alpha1.EnvoyConfigRevision)
				if resources, err := (ecr.Spec.EnvoyResources).Resources(ecr.GetSerialization()); err != nil {
					return err
				} else {
					ecr.Spec.Resources = resources
					ecr.Spec.EnvoyResources = nil
				}
			}
			return nil
		}),
		// set finalizer
		reconciler.WithFinalizer(marin3rv1alpha1.EnvoyConfigRevisionFinalizer),
		// cleanup logic
		reconciler.WithFinalizationFunc(func(context.Context, client.Client) error {
			envoyconfigrevision.CleanupLogic(ecr, r.XdsCache, r.DiscoveryStats, logger)
			logger.Info("finalized EnvoyConfigRevision resource")
			return nil
		}),
	)
	if result.ShouldReturn() {
		return result.Values()
	}

	var vt *marin3rv1alpha1.VersionTracker = nil

	// If this ecr has the RevisionPublishedCondition set to "True" pusblish the resources
	// to the xds server cache
	if meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionPublishedCondition) {
		var err error
		decoder := envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, r.APIVersion)

		cacheReconciler := envoyconfigrevision.NewCacheReconciler(
			ctx, logger, r.Client, r.XdsCache,
			decoder,
			envoy_resources.NewGenerator(r.APIVersion),
		)

		vt, err = cacheReconciler.Reconcile(ctx, req.NamespacedName, ecr.Spec.Resources, ecr.Spec.NodeID, ecr.Spec.Version)

		// If a type errors.StatusError is returned it means that the config in spec.resources is wrong
		// and cannot be written into the xDS cache. This is true for any error loading all types of resources
		// except for Secrets and generated Endpoints, which are dynamically loaded from the k8s API. In those cases
		// setting a permanent taint could occur for a transient failure, which is not desirable.
		if err != nil {
			switch err.(type) {
			case *errors.StatusError:
				logger.Error(err, fmt.Sprintf("%v", err))
				if err := r.taintSelf(ctx, ecr, "FailedLoadingResources", err.Error(), logger); err != nil {
					return ctrl.Result{}, err
				}
			default:
				return ctrl.Result{}, err
			}
		}
	}

	if ok := envoyconfigrevision.IsStatusReconciled(ecr, vt, r.XdsCache, r.DiscoveryStats); !ok {
		if err := r.Client.Status().Update(ctx, ecr); err != nil {
			logger.Error(err, "unable to update EnvoyConfigRevision status")
		}
		logger.Info("status updated for EnvoyConfigRevision resource")
	}

	if meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionPublishedCondition) {
		return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil
	}

	return ctrl.Result{}, nil

}

func (r *EnvoyConfigRevisionReconciler) taintSelf(ctx context.Context, ecr *marin3rv1alpha1.EnvoyConfigRevision,
	reason, msg string, logger logr.Logger) error {

	if !meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionTaintedCondition) {
		patch := client.MergeFrom(ecr.DeepCopy())
		meta.SetStatusCondition(&ecr.Status.Conditions, metav1.Condition{
			Type:    marin3rv1alpha1.RevisionTaintedCondition,
			Status:  metav1.ConditionTrue,
			Reason:  reason,
			Message: msg,
		})
		ecr.Status.Tainted = pointer.New(true)

		if err := r.Client.Status().Patch(ctx, ecr, patch); err != nil {
			return err
		}

		logger.Info(fmt.Sprintf("Tainted revision: %q", msg))
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
	return r.FilteredEventHandler(
		&marin3rv1alpha1.EnvoyConfigRevisionList{},
		func(event client.Object, o client.Object) bool {
			secret := event.(*corev1.Secret)
			if secret.Type != corev1.SecretTypeTLS && secret.Type != corev1.SecretTypeOpaque {
				return false
			}
			ecr := o.(*marin3rv1alpha1.EnvoyConfigRevision)
			if meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionPublishedCondition) {
				// check if the k8s Secret is relevant for this EnvoyConfigRevision
				for _, s := range ecr.Spec.Resources {
					if s.Type == envoy.Secret {
						if (s.GenerateFromTlsSecret != nil && *s.GenerateFromTlsSecret == secret.GetName()) ||
							(s.GenerateFromOpaqueSecret != nil && s.GenerateFromOpaqueSecret.Name == secret.GetName()) {
							return true
						}
					}

				}
			}
			return false
		},
		logr.Discard(),
	)
}

// EndpointSlicesEventHandler returns an EventHandler that generates
// reconcile requests for EndpointSlices
func (r *EnvoyConfigRevisionReconciler) EndpointSlicesEventHandler() handler.EventHandler {
	return r.FilteredEventHandler(
		&marin3rv1alpha1.EnvoyConfigRevisionList{},
		func(event client.Object, o client.Object) bool {
			endpointSlice := event.(*discoveryv1.EndpointSlice)
			ecr := o.(*marin3rv1alpha1.EnvoyConfigRevision)
			if meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionPublishedCondition) {
				// check if the k8s EndpointSlice is relevant for this EnvoyConfigRevision
				for _, r := range ecr.Spec.Resources {
					if r.Type == envoy.Endpoint && r.GenerateFromEndpointSlices != nil {

						selector, err := metav1.LabelSelectorAsSelector(r.GenerateFromEndpointSlices.Selector)
						if err != nil {
							// skip this item in case of error
							continue
						}

						// generate a reconcile request if this event is relevant for this revision
						if selector.Matches(labels.Set(endpointSlice.GetLabels())) {
							return true
						}
					}
				}
			}
			return false
		},
		logr.Discard(),
	)
}

// SetupWithManager adds the controller to the manager
func (r *EnvoyConfigRevisionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&marin3rv1alpha1.EnvoyConfigRevision{}).
		WithEventFilter(filterByAPIVersionPredicate(r.APIVersion, filterByAPIVersion)).
		Watches(&corev1.Secret{}, r.SecretsEventHandler()).
		Watches(&discoveryv1.EndpointSlice{}, r.EndpointSlicesEventHandler()).
		Complete(r)
}
