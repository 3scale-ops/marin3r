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

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/common"
	"github.com/3scale/marin3r/pkg/reconcilers/lockedresources"
	"github.com/3scale/marin3r/pkg/reconcilers/operator/discoveryservice/generators"
	"github.com/go-logr/logr"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/redhat-cop/operator-utils/pkg/util/lockedresourcecontroller/lockedpatch"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/handler"
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

var defaultExcludedPaths = []string{".metadata", ".status"}

// DiscoveryServiceReconciler reconciles a DiscoveryService object
type DiscoveryServiceReconciler struct {
	lockedresources.Reconciler
	Log logr.Logger
}

// +kubebuilder:rbac:groups=operator.marin3r.3scale.net,namespace=placeholder,resources=*,verbs=*
// +kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=*,verbs=*
// +kubebuilder:rbac:groups="core",namespace=placeholder,resources=services,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="core",namespace=placeholder,resources=serviceaccounts,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="apps",namespace=placeholder,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",namespace=placeholder,resources=roles,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",namespace=placeholder,resources=rolebindings,verbs=get;list;watch;create;patch

func (r *DiscoveryServiceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("name", request.Name, "namespace", request.Namespace)

	ds := &operatorv1alpha1.DiscoveryService{}
	key := types.NamespacedName{Name: request.Name, Namespace: request.Namespace}
	err := r.GetClient().Get(ctx, key, ds)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if ok := r.IsInitialized(ds, operatorv1alpha1.DiscoveryServiceFinalizer); !ok {
		err := r.GetClient().Update(ctx, ds)
		if err != nil {
			log.Error(err, "unable to initialize instance")
			return r.ManageError(ctx, ds, err)
		}
		return ctrl.Result{}, nil
	}

	if util.IsBeingDeleted(ds) {
		if !util.HasFinalizer(ds, operatorv1alpha1.DiscoveryServiceFinalizer) {
			return ctrl.Result{}, nil
		}
		err := r.ManageCleanUpLogic(ds, log)
		if err != nil {
			log.Error(err, "unable to delete instance")
			return r.ManageError(ctx, ds, err)
		}
		util.RemoveFinalizer(ds, operatorv1alpha1.DiscoveryServiceFinalizer)
		err = r.GetClient().Update(ctx, ds)
		if err != nil {
			log.Error(err, "unable to update instance")
			return r.ManageError(ctx, ds, err)
		}
		return ctrl.Result{}, nil
	}

	generate := generators.GeneratorOptions{
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
	}

	hash, err := r.calculateServerCertificateHash(ctx, common.ObjectKey(generate.ServerCertificate()()))
	if err != nil {
		return r.ManageError(ctx, ds, err)
	}

	resources, err := r.NewLockedResources(
		[]lockedresources.LockedResource{
			{GeneratorFn: generate.RootCertificationAuthority(), ExcludePaths: defaultExcludedPaths},
			{GeneratorFn: generate.ServerCertificate(), ExcludePaths: defaultExcludedPaths},
			{GeneratorFn: generate.ServiceAccount(), ExcludePaths: defaultExcludedPaths},
			{GeneratorFn: generate.Role(), ExcludePaths: defaultExcludedPaths},
			{GeneratorFn: generate.RoleBinding(), ExcludePaths: defaultExcludedPaths},
			{GeneratorFn: generate.Service(), ExcludePaths: append(defaultExcludedPaths, ".spec.clusterIP")},
			{GeneratorFn: generate.Deployment(hash), ExcludePaths: defaultExcludedPaths},
			{GeneratorFn: generate.EnvoyBootstrap(), ExcludePaths: defaultExcludedPaths},
		},
		ds,
	)

	err = r.UpdateLockedResources(ctx, ds, resources, []lockedpatch.LockedPatch{})
	if err != nil {
		log.Error(err, "unable to update locked resources")
		return r.ManageError(ctx, ds, err)
	}

	return r.ManageSuccess(ctx, ds)
}

func (r *DiscoveryServiceReconciler) calculateServerCertificateHash(ctx context.Context, key types.NamespacedName) (string, error) {
	// Fetch the server certificate to calculate the hash and
	// populate the deployment's label.
	// This will trigger rollouts on server certificate changes.
	serverDSC := &operatorv1alpha1.DiscoveryServiceCertificate{}
	err := r.GetClient().Get(ctx, key, serverDSC)
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

	return ctrl.NewControllerManagedBy(mgr).For(&operatorv1alpha1.DiscoveryService{}).
		Watches(&source.Channel{Source: r.GetStatusChangeChannel()}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
