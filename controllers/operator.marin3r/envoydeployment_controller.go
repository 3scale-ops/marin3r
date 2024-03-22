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
	"reflect"
	"time"

	"github.com/3scale-ops/basereconciler/mutators"
	"github.com/3scale-ops/basereconciler/reconciler"
	"github.com/3scale-ops/basereconciler/resource"
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/envoy/container/defaults"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/operator/envoydeployment/generators"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// EnvoyDeploymentReconciler reconciles a EnvoyDeployment object
type EnvoyDeploymentReconciler struct {
	*reconciler.Reconciler
}

//+kubebuilder:rbac:groups=operator.marin3r.3scale.net,namespace=placeholder,resources=envoydeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.marin3r.3scale.net,namespace=placeholder,resources=envoydeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.marin3r.3scale.net,namespace=placeholder,resources=envoydeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups="apps",namespace=placeholder,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="autoscaling",namespace=placeholder,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="policy",namespace=placeholder,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.marin3r.3scale.net,namespace=placeholder,resources=discoveryservicecertificates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigs,verbs=get;list;watch
//+kubebuilder:rbac:groups=operator.marin3r.3scale.net,namespace=placeholder,resources=discoveryservices,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *EnvoyDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	ctx, logger := r.Logger(ctx, "name", req.Name, "namespace", req.Namespace)
	ed := &operatorv1alpha1.EnvoyDeployment{}
	result := r.ManageResourceLifecycle(ctx, req, ed)
	if result.ShouldReturn() {
		return result.Values()
	}

	// Get the address of the DiscoveryService instance
	ds := &operatorv1alpha1.DiscoveryService{}
	dsKey := types.NamespacedName{Name: ed.Spec.DiscoveryServiceRef, Namespace: ed.GetNamespace()}
	if err := r.Client.Get(ctx, dsKey, ds); err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "DiscoveryService does not exist", "DiscoveryService", ed.Spec.DiscoveryServiceRef)
		}
		return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, err
	}

	// Get the EnvoyConfig for additional data (like the envoy API version in use)
	ec, err := r.getEnvoyConfig(ctx, types.NamespacedName{Name: ed.Spec.EnvoyConfigRef, Namespace: ed.GetNamespace()})
	if err != nil {
		logger.Error(err, "unable to get EnvoyConfig", "EnvoyConfig", ed.Spec.EnvoyConfigRef)
		return ctrl.Result{}, err
	}

	gen := generators.GeneratorOptions{
		InstanceName:         ed.GetName(),
		Namespace:            ed.GetNamespace(),
		DiscoveryServiceName: ed.Spec.DiscoveryServiceRef,
		XdssAdress:           fmt.Sprintf("%s.%s.%s", ds.GetServiceConfig().Name, ds.GetNamespace(), "svc"),
		XdssPort:             int(ds.GetXdsServerPort()),
		EnvoyAPIVersion:      ec.GetEnvoyAPIVersion(),
		EnvoyNodeID:          ec.Spec.NodeID,
		EnvoyClusterID: func() string {
			if ed.Spec.ClusterID != nil {
				return *ed.Spec.ClusterID
			}
			return ec.Spec.NodeID
		}(),
		ClientCertificateName:     fmt.Sprintf("%s-%s", defaults.DeploymentClientCertificate, ed.GetName()),
		ClientCertificateDuration: ed.ClientCertificateDuration(),
		SigningCertificateName:    ds.GetRootCertificateAuthorityOptions().SecretName,
		DeploymentImage:           ed.Image(),
		DeploymentResources:       ed.Resources(),
		ExposedPorts:              ed.Spec.Ports,
		ExtraArgs:                 ed.Spec.ExtraArgs,
		AdminPort:                 int32(ed.AdminPort()),
		AdminAccessLogPath:        ed.AdminAccessLogPath(),
		Replicas:                  ed.Replicas(),
		LivenessProbe:             ed.LivenessProbe(),
		ReadinessProbe:            ed.ReadinessProbe(),
		Affinity:                  ed.Affinity(),
		PodDisruptionBudget:       ed.PodDisruptionBudget(),
		ShutdownManager:           ed.Spec.ShutdownManager,
		InitManager:               ed.Spec.InitManager,
	}

	resources := []resource.TemplateInterface{
		resource.NewTemplateFromObjectFunction(gen.ClientCertificate).Apply(dscDefaulter),
		resource.NewTemplateFromObjectFunction(gen.Deployment).
			WithMutation(mutators.SetDeploymentReplicas(ed.Replicas().Dynamic == nil)),
		resource.NewTemplateFromObjectFunction(gen.HPA).
			WithEnabled(ed.Replicas().Dynamic != nil),
		resource.NewTemplateFromObjectFunction(gen.PDB).
			WithEnabled(!reflect.DeepEqual(ed.PodDisruptionBudget(), operatorv1alpha1.PodDisruptionBudgetSpec{})),
	}

	result = r.ReconcileOwnedResources(ctx, ed, resources)
	if result.ShouldReturn() {
		return result.Values()
	}

	// reconcile the status
	result = r.ReconcileStatus(ctx, ed, []types.NamespacedName{gen.OwnedResourceKey()}, nil,
		func() bool {
			if ed.Status.DeploymentName == nil || *ed.Status.DeploymentName != gen.OwnedResourceKey().Name {
				ed.Status.DeploymentName = pointer.New(gen.OwnedResourceKey().Name)
				return true
			}
			return false
		})
	if result.ShouldReturn() {
		return result.Values()
	}

	return ctrl.Result{}, nil
}

func (r *EnvoyDeploymentReconciler) getEnvoyConfig(ctx context.Context, key types.NamespacedName) (*marin3rv1alpha1.EnvoyConfig, error) {
	ec := &marin3rv1alpha1.EnvoyConfig{}
	err := r.Client.Get(ctx, key, ec)

	if err != nil {
		return nil, err
	}

	return ec, nil
}

// EnvoyConfigHandler returns an EventHandler to watch for EnvoyConfigs
func (r *EnvoyDeploymentReconciler) EnvoyConfigHandler() handler.EventHandler {
	return r.FilteredEventHandler(
		&operatorv1alpha1.EnvoyDeploymentList{},
		func(event client.Object, o client.Object) bool {
			ec := event.(*marin3rv1alpha1.EnvoyConfig)
			ed := o.(*operatorv1alpha1.EnvoyDeployment)
			if ed.Spec.EnvoyConfigRef == ec.GetName() {
				return true
			}
			return false
		},
		logr.Discard(),
	)
}

// EnvoyConfigHandler returns an EventHandler to watch for DiscoveryServices
func (r *EnvoyDeploymentReconciler) DiscoveryServiceHandler() handler.EventHandler {
	return r.FilteredEventHandler(
		&operatorv1alpha1.EnvoyDeploymentList{},
		func(event client.Object, o client.Object) bool {
			ds := event.(*operatorv1alpha1.DiscoveryService)
			ed := o.(*operatorv1alpha1.EnvoyDeployment)
			if ed.Spec.DiscoveryServiceRef == ds.GetName() {
				return true
			}
			return false
		},
		logr.Discard(),
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnvoyDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.EnvoyDeployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&operatorv1alpha1.DiscoveryServiceCertificate{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Watches(&marin3rv1alpha1.EnvoyConfig{}, r.EnvoyConfigHandler()).
		Watches(&operatorv1alpha1.DiscoveryService{}, r.DiscoveryServiceHandler()).
		Complete(r)
}
