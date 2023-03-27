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

	"github.com/3scale-ops/basereconciler/reconciler"
	"github.com/3scale-ops/basereconciler/resources"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/operator/discoveryservice/generators"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/resource_extensions"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	reconciler.Config.AnnotationsDomain = "marin3r.3scale.net"
	reconciler.Config.ResourcePruner = false
	reconciler.Config.ManagedTypes = reconciler.NewManagedTypes().
		Register(&appsv1.DeploymentList{}).
		Register(&corev1.ServiceList{}).
		Register(&rbacv1.RoleList{}).
		Register(&rbacv1.RoleBindingList{}).
		Register(&corev1.ServiceAccountList{}).
		Register(&operatorv1alpha1.DiscoveryServiceCertificateList{})
}

// DiscoveryServiceReconciler reconciles a DiscoveryService object
type DiscoveryServiceReconciler struct {
	reconciler.Reconciler
	Log logr.Logger
}

// +kubebuilder:rbac:groups=operator.marin3r.3scale.net,namespace=placeholder,resources=*,verbs=*
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=*,verbs=*
// +kubebuilder:rbac:groups="core",namespace=placeholder,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="core",namespace=placeholder,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apps",namespace=placeholder,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",namespace=placeholder,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",namespace=placeholder,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="core",namespace=placeholder,resources=pods,verbs=list;watch;get
// +kubebuilder:rbac:groups="core",namespace=placeholder,resources=secrets,verbs=get;list;watch;create;update;patch

func (r *DiscoveryServiceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", request.Name, "namespace", request.Namespace)
	ctx = log.IntoContext(ctx, logger)

	ds := &operatorv1alpha1.DiscoveryService{}
	key := types.NamespacedName{Name: request.Name, Namespace: request.Namespace}
	result, err := r.GetInstance(ctx, key, ds, nil, nil)
	if result != nil || err != nil {
		return *result, err
	}

	// TODO: remove the old finalizer required by operator-utils

	gen := generators.GeneratorOptions{
		InstanceName:                      ds.GetName(),
		Namespace:                         ds.GetNamespace(),
		RootCertificateNamePrefix:         "marin3r-ca-cert",
		RootCertificateCommonNamePrefix:   "marin3r-ca",
		RootCertificateDuration:           func() (d time.Duration) { d, _ = time.ParseDuration("26280h"); return }(), // 3 years
		ServerCertificateNamePrefix:       "marin3r-server-cert",
		ServerCertificateCommonNamePrefix: "marin3r-server",
		ServerCertificateDuration:         func() (d time.Duration) { d, _ = time.ParseDuration("2160h"); return }(), // 90 days,
		ClientCertificateDuration:         func() (d time.Duration) { d, _ = time.ParseDuration("48h"); return }(),
		XdsServerPort:                     int32(ds.GetXdsServerPort()),
		MetricsServerPort:                 int32(ds.GetMetricsPort()),
		ServiceType:                       operatorv1alpha1.ClusterIPType,
		DeploymentImage:                   ds.GetImage(),
		DeploymentResources:               ds.Resources(),
		Debug:                             ds.Debug(),
	}

	hash, err := r.calculateServerCertificateHash(ctx, types.NamespacedName{Name: gen.ServerCertName(), Namespace: gen.Namespace})
	if err != nil {
		return ctrl.Result{}, err
	}

	res := []reconciler.Resource{
		resource_extensions.DiscoveryServiceCertificateTemplate{Template: gen.RootCertificationAuthority(), IsEnabled: true},
		resource_extensions.DiscoveryServiceCertificateTemplate{Template: gen.ServerCertificate(), IsEnabled: true},
		resource_extensions.DiscoveryServiceCertificateTemplate{Template: gen.ClientCertificate(), IsEnabled: true},
		resources.ServiceAccountTemplate{Template: gen.ServiceAccount(), IsEnabled: true},
		resources.RoleTemplate{Template: gen.Role(), IsEnabled: true},
		resources.RoleBindingTemplate{Template: gen.RoleBinding(), IsEnabled: true},
		resources.ServiceTemplate{Template: gen.Service(), IsEnabled: true},
		resources.DeploymentTemplate{
			Template:        gen.Deployment(hash),
			EnforceReplicas: true,
			IsEnabled:       true,
		},
	}

	if err := r.ReconcileOwnedResources(ctx, ds, res); err != nil {
		logger.Error(err, "unable to update owned resources")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DiscoveryServiceReconciler) calculateServerCertificateHash(ctx context.Context, key types.NamespacedName) (string, error) {
	// Fetch the server certificate to calculate the hash and
	// populate the deployment's label.
	// This will trigger rollouts on server certificate changes.
	serverDSC := &operatorv1alpha1.DiscoveryServiceCertificate{}
	err := r.Client.Get(ctx, key, serverDSC)
	if err != nil {
		if errors.IsNotFound(err) {
			// The server certificate hasn't been created yet
			return "", nil
		}
		return "", err
	}
	return serverDSC.Status.GetCertificateHash(), nil
}

// SetupWithManager adds the controller to the manager
func (r *DiscoveryServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.DiscoveryService{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&operatorv1alpha1.DiscoveryServiceCertificate{}).
		Complete(r)
}
