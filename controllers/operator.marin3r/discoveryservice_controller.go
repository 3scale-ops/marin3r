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
	"time"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator.marin3r/v1alpha1"

	"github.com/3scale/marin3r/pkg/reconcilers"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// as there is currently no renewal mechanism for the CA
	// set a validity sufficiently high. This might be configurable
	// in the future when renewal is managed by the operator
	caCertValidFor             int64  = 3600 * 24 * 365 * 3 // 3 years
	serverCertValidFor         int64  = 3600 * 24 * 90      // 90 days
	clientCertValidFor         string = "48h"
	caCommonName               string = "marin3r-ca"
	caCertSecretNamePrefix     string = "marin3r-ca-cert"
	serverCommonName           string = "marin3r-server"
	serverCertSecretNamePrefix string = "marin3r-server-cert"
)

// DiscoveryServiceReconciler reconciles a DiscoveryService object
type DiscoveryServiceReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
	ds     *operatorv1alpha1.DiscoveryService
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=operator.marin3r.3scale.net,resources=*,verbs=*
// +kubebuilder:rbac:groups=marin3r.3scale.net,resources=*,verbs=*
// +kubebuilder:rbac:groups="core",resources=services,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="core",resources=serviceaccounts,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;patch

func (r *DiscoveryServiceReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("name", request.Name, "namespace", request.Namespace)

	ds := &operatorv1alpha1.DiscoveryService{}
	key := types.NamespacedName{Name: request.Name, Namespace: request.Namespace}
	err := r.Client.Get(ctx, key, ds)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Call reconcilers in the proper installation order
	var result ctrl.Result
	r.ds = ds

	result, err = r.reconcileCA(ctx, log)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileServerCertificate(ctx, log)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileServiceAccount(ctx, log)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileRole(ctx, log)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileRoleBinding(ctx, log)
	if result.Requeue || err != nil {
		return result, err
	}

	// Fetch the server certificate to calculate the hash and
	// populate the deployment's label.
	// This will trigger rollouts on server certificate changes.
	secret := &corev1.Secret{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: getServerCertName(r.ds), Namespace: OwnedObjectNamespace(r.ds)}, secret)
	if err != nil {
		return ctrl.Result{}, err
	}

	dr := reconcilers.NewDeploymentReconciler(ctx, log, r.Client, r.Scheme, r.ds)
	result, err = dr.Reconcile(
		types.NamespacedName{Name: OwnedObjectName(r.ds), Namespace: OwnedObjectNamespace(r.ds)},
		deploymentGeneratorFn(r.ds, secret),
	)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileService(ctx, log)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileEnvoyBootstrap(ctx, log)
	if result.Requeue || err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
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

func (r *DiscoveryServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).For(&operatorv1alpha1.DiscoveryService{}).
		Owns(&operatorv1alpha1.DiscoveryServiceCertificate{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&marin3rv1alpha1.EnvoyBootstrap{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{}).
		WithEventFilter(filterTLSTypeCertificatesPredicate()).
		Complete(r)
}
