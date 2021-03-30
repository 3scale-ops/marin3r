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

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	envoyconfig "github.com/3scale-ops/marin3r/pkg/reconcilers/marin3r/envoyconfig"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// EnvoyConfigReconciler reconciles a EnvoyConfig object
type EnvoyConfigReconciler struct {
	Client client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile progresses EnvoyConfig resources to its desired state
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigrevisions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigrevisions/status,verbs=get;update;patch

func (r *EnvoyConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("name", req.Name, "namespace", req.Namespace)

	// Fetch the EnvoyConfig instance
	ec := &marin3rv1alpha1.EnvoyConfig{}
	err := r.Client.Get(ctx, req.NamespacedName, ec)
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

	log = log.WithValues("nodeID", ec.Spec.NodeID, "envoyAPI", ec.GetEnvoyAPIVersion())

	if ok := envoyconfig.IsInitialized(ec); !ok {
		if err := r.Client.Update(ctx, ec); err != nil {
			log.Error(err, "unable to update EnvoyConfig")
			return ctrl.Result{}, err
		}
		log.Info("initialized EnvoyConfig resource")
		return reconcile.Result{}, nil
	}

	revisionReconciler := envoyconfig.NewRevisionReconciler(
		ctx, log, r.Client, r.Scheme, ec,
	)

	result, err := revisionReconciler.Reconcile()
	if result.Requeue || err != nil {
		return result, err
	}

	if ok := envoyconfig.IsStatusReconciled(ec, revisionReconciler.GetCacheState(), revisionReconciler.PublishedVersion(), revisionReconciler.GetRevisionList()); !ok {
		if err := r.Client.Status().Update(ctx, ec); err != nil {
			log.Error(err, "unable to update EnvoyConfig status")
			return ctrl.Result{}, err
		}
		log.Info("status updated for EnvoyConfig resource")
		return reconcile.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager adds the controller to the manager
func (r *EnvoyConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&marin3rv1alpha1.EnvoyConfig{}).
		Owns(&marin3rv1alpha1.EnvoyConfigRevision{}).
		Complete(r)
}
