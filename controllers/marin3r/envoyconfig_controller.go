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

	"github.com/3scale-ops/basereconciler/reconciler"
	reconciler_util "github.com/3scale-ops/basereconciler/util"
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	envoyconfig "github.com/3scale-ops/marin3r/pkg/reconcilers/marin3r/envoyconfig"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// EnvoyConfigReconciler reconciles a EnvoyConfig object
type EnvoyConfigReconciler struct {
	*reconciler.Reconciler
}

// Reconcile progresses EnvoyConfig resources to its desired state
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigrevisions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigrevisions/status,verbs=get;update;patch

func (r *EnvoyConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	ctx, logger := r.Logger(ctx, "name", req.Name, "namespace", req.Namespace)
	ec := &marin3rv1alpha1.EnvoyConfig{}
	result := r.ManageResourceLifecycle(ctx, req, ec,
		// Apply defaults
		reconciler.WithInitializationFunc(reconciler_util.ResourceDefaulter(ec)),
		// convert spec.EnvoyResources to spec.Resources
		reconciler.WithInMemoryInitializationFunc(func(ctx context.Context, c client.Client, o client.Object) error {
			if ec.Spec.EnvoyResources != nil {
				ec := o.(*marin3rv1alpha1.EnvoyConfig)
				if resources, err := (ec.Spec.EnvoyResources).Resources(ec.GetSerialization()); err != nil {
					return err
				} else {
					ec.Spec.Resources = resources
					ec.Spec.EnvoyResources = nil
				}
			}
			return nil
		}),
	)
	if result.ShouldReturn() {
		return result.Values()
	}

	logger = logger.WithValues("nodeID", ec.Spec.NodeID, "envoyAPI", ec.GetEnvoyAPIVersion())

	revisionReconciler := envoyconfig.NewRevisionReconciler(
		ctx, logger, r.Client, r.Scheme, ec,
	)

	reconcilerResult, err := revisionReconciler.Reconcile()
	if reconcilerResult.Requeue || err != nil {
		return reconcilerResult, err
	}

	if ok := envoyconfig.IsStatusReconciled(ec, revisionReconciler.GetCacheState(), revisionReconciler.PublishedVersion(), revisionReconciler.GetRevisionList()); !ok {
		if err := r.Client.Status().Update(ctx, ec); err != nil {
			logger.Error(err, "unable to update EnvoyConfig status")
			return ctrl.Result{}, err
		}
		logger.Info("status updated for EnvoyConfig resource")
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
