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

	"github.com/3scale-ops/basereconciler/reconciler"
	"github.com/3scale-ops/basereconciler/resources"
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/envoy/container/defaults"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/operator/envoydeployment/generators"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/resource_extensions"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// EnvoyDeploymentReconciler reconciles a EnvoyDeployment object
type EnvoyDeploymentReconciler struct {
	reconciler.Reconciler
	Log logr.Logger
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
	logger := r.Log.WithValues("envoydeployment", req.NamespacedName)
	ctx = log.IntoContext(ctx, logger)

	ed := &operatorv1alpha1.EnvoyDeployment{}
	key := types.NamespacedName{Name: req.Name, Namespace: req.Namespace}
	err := r.GetInstance(ctx, key, ed, nil, nil)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Temporary code to remove finalizers from EnvoyDeployment resources
	if controllerutil.ContainsFinalizer(ed, operatorv1alpha1.Finalizer) {
		controllerutil.RemoveFinalizer(ed, operatorv1alpha1.Finalizer)
		if err := r.Client.Update(ctx, ed); err != nil {
			logger.Error(err, "unable to remove finalizer")
		}
		return ctrl.Result{}, nil
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

	res := []reconciler.Resource{
		resource_extensions.DiscoveryServiceCertificateTemplate{
			Template:  gen.ClientCertificate(),
			IsEnabled: true,
		},
		resources.DeploymentTemplate{
			Template:        gen.Deployment(),
			EnforceReplicas: ed.Replicas().Dynamic == nil,
			IsEnabled:       true,
		},
		resources.HorizontalPodAutoscalerTemplate{
			Template:  gen.HPA(),
			IsEnabled: ed.Replicas().Dynamic != nil,
		},
		resources.PodDisruptionBudgetTemplate{
			Template:  gen.PDB(),
			IsEnabled: !reflect.DeepEqual(ed.PodDisruptionBudget(), operatorv1alpha1.PodDisruptionBudgetSpec{}),
		},
	}

	if err := r.ReconcileOwnedResources(ctx, ed, res); err != nil {
		logger.Error(err, "unable to update owned resources")
		return ctrl.Result{}, err
	}

	// reconcile the status
	err = r.ReconcileStatus(ctx, ed, []types.NamespacedName{gen.OwnedResourceKey()}, nil,
		func() bool {
			if ed.Status.DeploymentName == nil || *ed.Status.DeploymentName != gen.OwnedResourceKey().Name {
				ed.Status.DeploymentName = pointer.New(gen.OwnedResourceKey().Name)
				return true
			}
			return false
		})
	if err != nil {
		return ctrl.Result{}, err
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

// SetupWithManager sets up the controller with the Manager.
func (r *EnvoyDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.EnvoyDeployment{}).
		Complete(r)
}

// EnvoyConfigHandler returns an EventHandler to watch for EnvoyConfigs
func (r *EnvoyDeploymentReconciler) EnvoyConfigHandler() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(
		func(o client.Object) []reconcile.Request {
			edList := &operatorv1alpha1.EnvoyDeploymentList{}
			if err := r.Client.List(context.TODO(), edList, client.InNamespace(o.GetNamespace())); err != nil {
				r.Log.Error(err, "unable to retrieve the list of EnvoyDeployment resources in the namespace",
					"Type", "EnvoyConfig", "Name", o.GetName(), "Namespace", o.GetNamespace())
				return []reconcile.Request{}
			}

			// Return a reconcile event for all EnvoyDeployments that have a reference to this EnvoyConfig
			req := []reconcile.Request{}
			for _, ed := range edList.Items {
				if ed.Spec.EnvoyConfigRef == o.GetName() {
					req = append(req, reconcile.Request{
						NamespacedName: types.NamespacedName{Name: ed.GetName(), Namespace: ed.GetNamespace()},
					})
				}
			}

			return req
		},
	)
}
