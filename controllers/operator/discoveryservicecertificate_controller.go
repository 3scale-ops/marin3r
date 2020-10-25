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
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// DiscoveryServiceCertificateReconciler reconciles a DiscoveryServiceCertificate object
type DiscoveryServiceCertificateReconciler struct {
	// This Client, initialized using mgr.Client() above, is a split Client
	// that reads objects from the cache and writes to the apiserver
	Client           client.Client
	Scheme           *runtime.Scheme
	Log              logr.Logger
	discoveryClient  discovery.DiscoveryInterface
	certificateWatch bool
}

// +kubebuilder:rbac:groups=operator.marin3r.3scale.net,resources=discoveryservicecertificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.marin3r.3scale.net,resources=discoveryservicecertificates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="core",resources=secrets,verbs=get;list;watch;create;update;patch

func (r *DiscoveryServiceCertificateReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("discoveryservicecertificate", request.NamespacedName)

	// Fetch the DiscoveryServiceCertificate instance
	dsc := &operatorv1alpha1.DiscoveryServiceCertificate{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, dsc)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if dsc.Spec.Signer.CASigned != nil {
		r.Log.Info("Reconciling ca-signed certificate")
		if err := r.reconcileCASignedCertificate(ctx, dsc); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		r.Log.Info("Reconciling self-signed certificate")
		if err := r.reconcileSelfSignedCertificate(ctx, dsc); err != nil {
			return ctrl.Result{}, err
		}
	}

	// TODO: set status Ready/NotReady

	return ctrl.Result{}, nil
}

func (r *DiscoveryServiceCertificateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.DiscoveryServiceCertificate{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
