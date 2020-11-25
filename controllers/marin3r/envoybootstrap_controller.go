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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/envoy"
	reconcilers_marin3r "github.com/3scale/marin3r/pkg/reconcilers/marin3r"
	corev1 "k8s.io/api/core/v1"
)

// EnvoyBootstrapReconciler reconciles a EnvoyBootstrap object
type EnvoyBootstrapReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=marin3r.3scale.net,resources=envoybootstraps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=marin3r.3scale.net,resources=envoybootstraps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.marin3r.3scale.net,resources=discoveryservicecertificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="core",resources=configmaps,verbs=get;list;watch;create;update;patch
func (r *EnvoyBootstrapReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	r.Log = r.Log.WithValues("envoybootstrap", req.NamespacedName)

	// Fetch the EnvoyBootstrap instance
	eb := &marin3rv1alpha1.EnvoyBootstrap{}
	err := r.Client.Get(ctx, req.NamespacedName, eb)
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

	certificateReconciler := reconcilers_marin3r.NewClientCertificateReconciler(ctx, r.Log, r.Client, r.Scheme, eb)
	result, err := certificateReconciler.Reconcile()
	if result.Requeue || err != nil {
		return result, err
	}

	configReconciler := reconcilers_marin3r.NewBootstrapConfigReconciler(ctx, r.Log, r.Client, r.Scheme, eb)

	// Reconcile the v2 config
	result, err = configReconciler.Reconcile(envoy.APIv2)
	if result.Requeue || err != nil {
		return result, err
	}

	// Reconcile the v3 config
	result, err = configReconciler.Reconcile(envoy.APIv3)
	if result.Requeue || err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

func (r *EnvoyBootstrapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&marin3rv1alpha1.EnvoyBootstrap{}).
		Owns(&operatorv1alpha1.DiscoveryServiceCertificate{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
