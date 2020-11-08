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

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	xdss "github.com/3scale/marin3r/pkg/discoveryservice/xdss"
	envoy "github.com/3scale/marin3r/pkg/envoy"
	envoy_resources "github.com/3scale/marin3r/pkg/envoy/resources"
	envoy_serializer "github.com/3scale/marin3r/pkg/envoy/serializer"
	reconcilers_envoy "github.com/3scale/marin3r/pkg/reconcilers/envoy"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// EnvoyConfigRevisionReconciler reconciles a EnvoyConfigRevision object
type EnvoyConfigRevisionReconciler struct {
	Client     client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	XdsCache   xdss.Cache
	APIVersion envoy.APIVersion
}

func (r *EnvoyConfigRevisionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("envoyconfigrevision", req.NamespacedName)
	r.Log.Info("Reconciling EnvoyConfigRevision")

	// Fetch the EnvoyConfigRevision instance
	ecr := &envoyv1alpha1.EnvoyConfigRevision{}
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

	// If this ecr has the RevisionPublishedCondition set to "True" pusblish the resources
	// to the xds server cache
	if ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition) {
		decoder := envoy_serializer.NewResourceUnmarshaller(ecr.GetSerialization(), r.APIVersion)

		cacheReconciler := reconcilers_envoy.NewCacheReconciler(
			ctx, r.Log, r.Client, r.XdsCache,
			decoder,
			envoy_resources.NewGenerator(r.APIVersion),
		)

		result, err := cacheReconciler.Reconcile(req.NamespacedName, ecr.Spec.EnvoyResources, ecr.Spec.NodeID, ecr.Spec.Version)

		if result.Requeue || err != nil {
			return result, err
		}
	}

	// Update status
	if err := r.updateStatus(ctx, ecr); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *EnvoyConfigRevisionReconciler) updateStatus(ctx context.Context, ecr *envoyv1alpha1.EnvoyConfigRevision) error {

	changed := false
	patch := client.MergeFrom(ecr.DeepCopy())

	// Clear ResourcesOutOfSyncCondition
	if ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.ResourcesOutOfSyncCondition) {
		ecr.Status.Conditions.SetCondition(status.Condition{
			Type:    envoyv1alpha1.ResourcesOutOfSyncCondition,
			Reason:  "NodeConficRevisionSynced",
			Status:  corev1.ConditionFalse,
			Message: "EnvoyConfigRevision successfully synced",
		})
		changed = true

	}

	// Set status.published and status.lastPublishedAt fields
	if ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition) && !ecr.Status.Published {
		ecr.Status.Published = true
		ecr.Status.LastPublishedAt = metav1.Now()
		// We also initialise the "tainted" status property to false
		ecr.Status.Tainted = false
		changed = true
	} else if !ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition) && ecr.Status.Published {
		ecr.Status.Published = false
		changed = true
	}

	// Set status.failed field
	if ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionTaintedCondition) && !ecr.Status.Tainted {
		ecr.Status.Tainted = true
		changed = true
	} else if !ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionTaintedCondition) && ecr.Status.Tainted {
		ecr.Status.Tainted = false
		changed = true
	}

	if changed {
		if err := r.Client.Status().Patch(ctx, ecr, patch); err != nil {
			return err
		}
	}

	return nil
}

func filterByAPIVersion(obj runtime.Object, version envoy.APIVersion) bool {
	switch o := obj.(type) {
	case *envoyv1alpha1.EnvoyConfigRevision:
		if o.GetEnvoyAPIVersion() == version {
			return true
		}
		return false

	default:
		return false
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

func (r *EnvoyConfigRevisionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&envoyv1alpha1.EnvoyConfigRevision{}).
		WithEventFilter(filterByAPIVersionPredicate(r.APIVersion, filterByAPIVersion)).
		Complete(r)
}
