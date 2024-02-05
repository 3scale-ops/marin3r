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
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	discoveryservicecertificate "github.com/3scale-ops/marin3r/pkg/reconcilers/operator/discoveryservicecertificate"
	marin3r_provider "github.com/3scale-ops/marin3r/pkg/reconcilers/operator/discoveryservicecertificate/providers/marin3r"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// DiscoveryServiceCertificateReconciler reconciles a DiscoveryServiceCertificate object
type DiscoveryServiceCertificateReconciler struct {
	*reconciler.Reconciler
}

// +kubebuilder:rbac:groups=operator.marin3r.3scale.net,namespace=placeholder,resources=discoveryservicecertificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.marin3r.3scale.net,namespace=placeholder,resources=discoveryservicecertificates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="core",namespace=placeholder,resources=secrets,verbs=get;list;watch;create;update;patch

func (r *DiscoveryServiceCertificateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	ctx, log := r.Logger(ctx, "name", req.Name, "namespace", req.Namespace)
	dsc := &operatorv1alpha1.DiscoveryServiceCertificate{}
	result := r.ManageResourceLifecycle(ctx, req, dsc,
		reconciler.WithInitializationFunc(reconciler_util.ResourceDefaulter(dsc)),
	)
	if result.ShouldReturn() {
		return result.Values()
	}

	// Only the internal certificate provider is currently supported
	provider := marin3r_provider.NewCertificateProvider(ctx, log, r.Client, r.Scheme, dsc)

	certificateReconciler := discoveryservicecertificate.NewCertificateReconciler(ctx, log, r.Client, r.Scheme, dsc, provider)
	reconcilerResult, err := certificateReconciler.Reconcile()
	if reconcilerResult.Requeue || err != nil {
		return reconcilerResult, err
	}

	if ok := discoveryservicecertificate.IsStatusReconciled(dsc, certificateReconciler.GetCertificateHash(),
		certificateReconciler.IsReady(), certificateReconciler.NotBefore(), certificateReconciler.NotAfter()); !ok {
		if err := r.Client.Status().Update(ctx, dsc); err != nil {
			log.Error(err, "unable to update DiscoveryServiceCertificate status")
			return ctrl.Result{}, err
		}
		log.Info("status updated for DiscoveryServiceCertificate resource")
	}

	if certificateReconciler.GetSchedule() == nil {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: *certificateReconciler.GetSchedule()}, nil
}

// IssuerChangedHandler returns an EventHandler that generates
// reconcile requests for Secrets
func (r *DiscoveryServiceCertificateReconciler) IssuerChangedHandler() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(
		func(o client.Object) []reconcile.Request {

			issuer := o.(*operatorv1alpha1.DiscoveryServiceCertificate)
			// Only interested in changes to CA certificates. A change in the CA
			// means that the child certificates need to be re-issued
			if !issuer.IsCA() {
				return []reconcile.Request{}
			}

			list := &operatorv1alpha1.DiscoveryServiceCertificateList{}
			if err := r.Client.List(context.Background(), list); err != nil {
				return []reconcile.Request{}
			}

			reconcileRequests := []reconcile.Request{}

			for _, dsc := range list.Items {
				if dsc.Spec.Signer.CASigned != nil &&
					dsc.Spec.Signer.CASigned.SecretRef.Name == issuer.Spec.SecretRef.Name {

					reconcileRequests = append(reconcileRequests,
						reconcile.Request{NamespacedName: types.NamespacedName{
							Name:      dsc.GetName(),
							Namespace: dsc.GetNamespace(),
						}})
				}
			}

			return reconcileRequests
		},
	)
}

// SetupWithManager adds the controller to the manager
func (r *DiscoveryServiceCertificateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.DiscoveryServiceCertificate{}).
		Owns(&corev1.Secret{}).
		Watches(&source.Kind{Type: &operatorv1alpha1.DiscoveryServiceCertificate{}}, r.IssuerChangedHandler()).
		Complete(r)
}
