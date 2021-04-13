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
	"reflect"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type SecretReconciler struct {
	Client client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,namespace=placeholder,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigs,verbs=get;list;watch;patch

func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// Fetch the Secret instance
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, req.NamespacedName, secret)
	if err != nil {
		// Error reading the object - requeue the request.
		// NOTE: We skip the IsNotFound error because we want to trigger EnvoyConfig
		// reconciles when referred secrets are deleted so the envoy control-plane
		// stops publishing them. This might cause errors if the reference hasn't been
		// removed from the EnvoyConfig, but that's ok as we do want to surface this
		// inconsistency instead of silently failing
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
	}

	log := r.Log.WithValues("name", req.Name, "namespace", req.Namespace)
	log.Info("Reconciling from 'kubernetes.io/tls' Secret")

	// Get the list of EnvoyConfigRevisions published and
	// check which of them contain refs to this secret
	list := &marin3rv1alpha1.EnvoyConfigRevisionList{}
	if err := r.Client.List(ctx, list); err != nil {
		return reconcile.Result{}, err
	}

	for _, ecr := range list.Items {

		if ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {

			for _, secret := range ecr.Spec.EnvoyResources.Secrets {
				if reflect.DeepEqual(secret.GetSecretKey(req.Namespace), req.NamespacedName) {
					log.Info("Triggered EnvoyConfigRevision reconcile",
						"EnvoyConfigRevision_Name", ecr.ObjectMeta.Name, "EnvoyConfigRevision_Namespace", ecr.GetNamespace())
					if err != nil {
						return reconcile.Result{}, err
					}

					if ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.ResourcesInSyncCondition) {
						patch := client.MergeFrom(ecr.DeepCopy())
						ecr.Status.Conditions.SetCondition(status.Condition{
							Type:    marin3rv1alpha1.ResourcesInSyncCondition,
							Reason:  "SecretChanged",
							Message: "A secret relevant to this envoyconfigrevision changed",
							Status:  corev1.ConditionFalse,
						})
						if err := r.Client.Status().Patch(ctx, &ecr, patch); err != nil {
							return reconcile.Result{}, err
						}
						log.V(1).Info("Condition should have been added ...")
					}
				}
			}
		}
	}

	return reconcile.Result{}, nil
}

func filterTLSTypeCertificatesPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			switch o := e.Object.(type) {
			case *corev1.Secret:
				if o.Type == "kubernetes.io/tls" {
					return true
				}
				return false

			default:
				return true
			}
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			switch o := e.ObjectNew.(type) {
			case *corev1.Secret:
				if o.Type == "kubernetes.io/tls" {
					return true
				}
				return false
			default:
				return true
			}
		},
		DeleteFunc: func(e event.DeleteEvent) bool { return false },
	}
}

func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&corev1.Secret{}).
		WithEventFilter(filterTLSTypeCertificatesPredicate()).
		Complete(r)
}
