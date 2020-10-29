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

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/reconcilers"
	"github.com/go-logr/logr"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1"
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
	clientCertValidFor         int64  = 3600 * 48           // 48 hours
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
// +kubebuilder:rbac:groups=envoy.marin3r.3scale.net,resources=*,verbs=*

// +kubebuilder:rbac:groups="core",resources=services,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="core",resources=serviceaccounts,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="core",resources=configmaps,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="core",resources=secrets,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="core",resources=namespaces,verbs=get;list;watch;watch;patch
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;patch

func (r *DiscoveryServiceReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("discoveryservice", request.NamespacedName)

	// Fetch the DiscoveryService instance
	dsList := &operatorv1alpha1.DiscoveryServiceList{}
	err := r.Client.List(ctx, dsList)
	if err != nil {
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	if len(dsList.Items) == 0 {
		// Request object not found, could have been deleted after reconcile request.
		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
		// Return and don't requeue
		return ctrl.Result{}, nil
	}

	if len(dsList.Items) > 1 {
		err := fmt.Errorf("More than one DiscoveryService object in the cluster, refusing to reconcile")
		r.Log.Error(err, "Only one marin3r installation per cluster is supported")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	// Call reconcilers in the proper installation order
	var result ctrl.Result
	r.ds = &dsList.Items[0]

	result, err = r.reconcileCA(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileServerCertificate(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileServiceAccount(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileClusterRole(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileClusterRoleBinding(ctx)
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

	dr := reconcilers.NewDeploymentReconciler(ctx, r.Log, r.Client, r.Scheme, r.ds)
	result, err = dr.Reconcile(
		types.NamespacedName{Name: OwnedObjectName(r.ds), Namespace: OwnedObjectNamespace(r.ds)},
		deploymentGeneratorFn(r.ds, secret),
	)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileService(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	// TODO: mechanism to cleanup resorces from namespaces
	// TODO: finalizer to cleanup labels in namespaces
	// This is necessary because namespaces are not
	// resources owned by this controller so the usual
	// garbage collection mechanisms won't
	result, err = r.reconcileEnabledNamespaces(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileMutatingWebhook(ctx)
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
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&admissionregistrationv1beta1.MutatingWebhookConfiguration{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{}).
		WithEventFilter(filterTLSTypeCertificatesPredicate()).
		Complete(r)
}
