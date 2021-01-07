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

	"github.com/go-logr/logr"

	"github.com/3scale/marin3r/pkg/reconcilers/operator/discoveryservice/generators"
	"github.com/redhat-cop/operator-utils/pkg/util/lockedresourcecontroller"
	"github.com/redhat-cop/operator-utils/pkg/util/lockedresourcecontroller/lockedpatch"
	"github.com/redhat-cop/operator-utils/pkg/util/lockedresourcecontroller/lockedresource"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	lockedresourcecontroller.EnforcingReconciler
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

	// Fetch the DiscoveryService instance
	dsList := &operatorv1alpha1.DiscoveryServiceList{}
	err := r.GetClient().List(ctx, dsList)
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
		err := fmt.Errorf("More than one DiscoveryService object in the namespace, refusing to reconcile")
		log.Error(err, "Only one marin3r installation per namespace is supported")
		return ctrl.Result{}, err
	}

	// var result ctrl.Result
	ds := &dsList.Items[0]

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

	rootCert, err := newUnstructured(generate.RootCertificationAuthority, ds, r.GetScheme(), log)
	if err != nil {
		return r.ManageError(ctx, ds, err)
	}

	serverCert, err := newUnstructured(generate.ServerCertificate, ds, r.GetScheme(), log)
	if err != nil {
		return r.ManageError(ctx, ds, err)
	}

	serviceAccount, err := newUnstructured(generate.ServiceAccount, ds, r.GetScheme(), log)
	if err != nil {
		return r.ManageError(ctx, ds, err)
	}

	role, err := newUnstructured(generate.Role, ds, r.GetScheme(), log)
	if err != nil {
		return r.ManageError(ctx, ds, err)
	}

	roleBinding, err := newUnstructured(generate.RoleBinding, ds, r.GetScheme(), log)
	if err != nil {
		return r.ManageError(ctx, ds, err)
	}

	service, err := newUnstructured(generate.Service, ds, r.GetScheme(), log)
	if err != nil {
		return r.ManageError(ctx, ds, err)
	}

	deployment, err := newUnstructured(generate.Deployment, ds, r.GetScheme(), log)
	if err != nil {
		return r.ManageError(ctx, ds, err)
	}

	envoyBootstrap, err := newUnstructured(generate.EnvoyBootstrap, ds, r.GetScheme(), log)
	if err != nil {
		return r.ManageError(ctx, ds, err)
	}

	resources := []lockedresource.LockedResource{
		{Unstructured: rootCert, ExcludedPaths: defaultExcludedPaths},
		{Unstructured: serverCert, ExcludedPaths: defaultExcludedPaths},
		{Unstructured: serviceAccount, ExcludedPaths: defaultExcludedPaths},
		{Unstructured: role, ExcludedPaths: defaultExcludedPaths},
		{Unstructured: roleBinding, ExcludedPaths: defaultExcludedPaths},
		{Unstructured: service, ExcludedPaths: append(defaultExcludedPaths, ".spec.clusterIP")},
		{Unstructured: deployment, ExcludedPaths: defaultExcludedPaths},
		{Unstructured: envoyBootstrap, ExcludedPaths: defaultExcludedPaths},
	}

	err = r.UpdateLockedResources(ctx, ds, resources, []lockedpatch.LockedPatch{})
	if err != nil {
		log.Error(err, "unable to update locked resources")
		return r.ManageError(ctx, ds, err)
	}

	// Fetch the server certificate to calculate the hash and
	// populate the deployment's label.
	// This will trigger rollouts on server certificate changes.
	serverDSC := &operatorv1alpha1.DiscoveryServiceCertificate{}
	key := types.NamespacedName{Name: serverCert.GetName(), Namespace: serverCert.GetNamespace()}
	err = r.GetClient().Get(ctx, key, serverDSC)
	if err != nil {
		return r.ManageError(ctx, ds, err)
	}
	if !serverDSC.Status.IsReady() {
		log.Info("Server certificate still not available, requeue")
		return r.ManageErrorWithRequeue(ctx, ds, fmt.Errorf("Server certificate not ready"), 5*time.Second)
	}

	dep := &appsv1.Deployment{}
	key = types.NamespacedName{Name: deployment.GetName(), Namespace: deployment.GetNamespace()}
	if err := r.GetClient().Get(ctx, key, dep); err != nil {
		log.Error(err, "unable to get deployment resource")
		return r.ManageError(ctx, ds, err)
	}

	// TODO: do not patch de label on each reconcile, just when needed
	patch := client.MergeFrom(dep.DeepCopy())
	dep.Spec.Template.ObjectMeta.Labels[operatorv1alpha1.DiscoveryServiceCertificateHashLabelKey] = serverDSC.Status.GetCertificateHash()
	if err := r.GetClient().Patch(ctx, dep, patch); err != nil {
		log.Error(err, "unable to patch deployment resource")
		return r.ManageError(ctx, ds, err)
	}

	return r.ManageSuccess(ctx, ds)
}

func newUnstructured(generator func() client.Object, owner client.Object, scheme *runtime.Scheme, log logr.Logger) (unstructured.Unstructured, error) {
	o := generator()
	if err := controllerutil.SetControllerReference(owner, o, scheme); err != nil {
		log.Error(err, "unable to SetControllerReference on resource",
			"kind", o.GetObjectKind().GroupVersionKind().String(),
			"namespace/name", client.ObjectKeyFromObject(o),
		)
		return unstructured.Unstructured{}, err
	}
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	return unstructured.Unstructured{Object: u}, nil
}

// SetupWithManager adds the controller to the manager
func (r *DiscoveryServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).For(&operatorv1alpha1.DiscoveryService{}).
		Owns(&operatorv1alpha1.DiscoveryServiceCertificate{}).
		Owns(&appsv1.Deployment{}).
		Watches(&source.Channel{Source: r.GetStatusChangeChannel()}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
