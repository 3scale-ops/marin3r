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
	"math"
	"time"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/util/pki"
	corev1 "k8s.io/api/core/v1"
)

// DiscoveryServiceCertificateWatcher watches for expiracy of DiscoveryServiceCertificate objects
type DiscoveryServiceCertificateWatcher struct {
	// This Client, initialized using mgr.Client() above, is a split Client
	// that reads objects from the cache and writes to the apiserver
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=operator.marin3r.3scale.net,resources=discoveryservicecertificates,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=operator.marin3r.3scale.net,resources=discoveryservicecertificates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="core",resources=secrets,verbs=get

func (r *DiscoveryServiceCertificateWatcher) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("discoveryservicecertificate_watcher", request.NamespacedName)

	r.Log.V(1).Info("Watching certificate validity")

	// Fetch the DiscoveryServiceCertificate instance
	dsc := &operatorv1alpha1.DiscoveryServiceCertificate{}
	err := r.Client.Get(ctx, request.NamespacedName, dsc)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Only the self-signed and ca-signed certificates have its renewal managed
	// by marin3r. cert-manager already does this for the cert-manager issued ones
	if dsc.Spec.Signer.SelfSigned != nil || dsc.Spec.Signer.CASigned != nil {
		secret := &corev1.Secret{}
		err := r.Client.Get(ctx,
			types.NamespacedName{
				Name:      dsc.Spec.SecretRef.Name,
				Namespace: dsc.Spec.SecretRef.Namespace,
			},
			secret)

		if err != nil {
			return ctrl.Result{}, err
		}

		cert, err := pki.LoadX509Certificate(secret.Data["tls.crt"])
		if err != nil {
			return ctrl.Result{}, err
		}

		// renewalWindow is the 20% of the certificate validity window
		renewalWindow := float64(dsc.Spec.ValidFor) * 0.20
		r.Log.V(1).Info("Debug", "RenewalWindow", renewalWindow)
		r.Log.V(1).Info("Debug", "TimeToExpiracy", cert.NotAfter.Sub(time.Now()).Seconds())

		if cert.NotAfter.Sub(time.Now()).Seconds() < renewalWindow {
			if !dsc.Status.Conditions.IsTrueFor(operatorv1alpha1.CertificateNeedsRenewalCondition) {
				r.Log.Info("Certificate needs renewal")

				// add condition
				patch := client.MergeFrom(dsc.DeepCopy())
				dsc.Status.Conditions.SetCondition(status.Condition{
					Type:    operatorv1alpha1.CertificateNeedsRenewalCondition,
					Status:  corev1.ConditionTrue,
					Reason:  status.ConditionReason("CertificateAboutToExpire"),
					Message: fmt.Sprintf("Certificate wil expire in less than %v seconds", renewalWindow),
				})
				if err := r.Client.Status().Patch(ctx, dsc, patch); err != nil {
					return ctrl.Result{}, err
				}
			}
		}

		poll, _ := time.ParseDuration(fmt.Sprintf("%ds", int64(math.Floor(float64(dsc.Spec.ValidFor)*0.10))))

		return ctrl.Result{RequeueAfter: poll}, nil
	}

	return ctrl.Result{}, nil
}

func (r *DiscoveryServiceCertificateWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.DiscoveryServiceCertificate{}).
		Complete(r)
}
