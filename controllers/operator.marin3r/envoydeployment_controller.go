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

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/lockedresources"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/operator/envoydeployment/generators"
	"github.com/go-logr/logr"
	operatorutil "github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/redhat-cop/operator-utils/pkg/util/lockedresourcecontroller/lockedpatch"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// EnvoyDeploymentReconciler reconciles a EnvoyDeployment object
type EnvoyDeploymentReconciler struct {
	lockedresources.Reconciler
	Log logr.Logger
}

//+kubebuilder:rbac:groups=operator.marin3r.3scale.net,namespace=placeholder,resources=envoydeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.marin3r.3scale.net,namespace=placeholder,resources=envoydeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.marin3r.3scale.net,namespace=placeholder,resources=envoydeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups="core",namespace=placeholder,resources=services,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="apps",namespace=placeholder,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoybootstraps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=marin3r.3scale.net,namespace=placeholder,resources=envoyconfigs,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *EnvoyDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("envoydeployment", req.NamespacedName)

	ed := &operatorv1alpha1.EnvoyDeployment{}
	key := types.NamespacedName{Name: req.Name, Namespace: req.Namespace}
	err := r.GetClient().Get(ctx, key, ed)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if ok := r.IsInitialized(ed, operatorv1alpha1.Finalizer); !ok {
		err := r.GetClient().Update(ctx, ed)
		if err != nil {
			log.Error(err, "unable to initialize instance")
			return r.ManageError(ctx, ed, err)
		}
		return ctrl.Result{}, nil
	}

	if operatorutil.IsBeingDeleted(ed) {
		if !operatorutil.HasFinalizer(ed, operatorv1alpha1.Finalizer) {
			return ctrl.Result{}, nil
		}
		err := r.ManageCleanUpLogic(ed, log)
		if err != nil {
			log.Error(err, "unable to delete instance")
			return r.ManageError(ctx, ed, err)
		}
		operatorutil.RemoveFinalizer(ed, operatorv1alpha1.Finalizer)
		err = r.GetClient().Update(ctx, ed)
		if err != nil {
			log.Error(err, "unable to update instance")
			return r.ManageError(ctx, ed, err)
		}
		return ctrl.Result{}, nil
	}

	// 1. get the DiscoveryService
	discoveryServiceName, err := r.getDiscoveryService(ctx, req.Namespace)
	if err != nil {
		log.Error(err, "unable to get DiscoveryService")
		return r.ManageError(ctx, ed, err)
	}

	// 2. get the EnvoyConfig
	ec, err := r.getEnvoyConfig(ctx, types.NamespacedName{Name: ed.Spec.EnvoyConfigRef, Namespace: ed.GetNamespace()})
	if err != nil {
		log.Error(err, "unable to get EnvoyConfig", "EnvoyConfig", ed.Spec.EnvoyConfigRef)
		return r.ManageError(ctx, ed, err)
	}

	// 3. Calculate hash of bootstrap configuration
	hash := r.getBootstrapConfigHash()

	// 4. Create resources
	generate := generators.GeneratorOptions{
		InstanceName:    ed.GetName(),
		Namespace:       ed.GetNamespace(),
		EnvoyAPIVersion: ec.GetEnvoyAPIVersion(),
		EnvoyNodeID:     ec.Spec.NodeID,
		EnvoyClusterID: func() string {
			if ed.Spec.ClusterID != nil {
				return *ed.Spec.ClusterID
			}
			return ec.Spec.NodeID
		}(),
		ClientCertificateDuration: ed.ClientCertificateDuration(),
		// Function with default
		DeploymentImage:     ed.Image(),
		DeploymentResources: ed.Resources(),
		ExposedPorts:        []operatorv1alpha1.ContainerPort{{Name: "port", Port: 8080}},
		ExtraArgs:           ed.Spec.ExtraArgs,
	}

	resources, err := r.NewLockedResources(
		[]lockedresources.LockedResource{
			{GeneratorFn: generate.Deployment(hash), ExcludePaths: defaultExcludedPaths},
			{GeneratorFn: generate.EnvoyBootstrap(discoveryServiceName), ExcludePaths: defaultExcludedPaths},
		},
		ed,
	)

	err = r.UpdateLockedResources(ctx, ed, resources, []lockedpatch.LockedPatch{})
	if err != nil {
		log.Error(err, "unable to update locked resources")
		return r.ManageError(ctx, ed, err)
	}

	return r.ManageSuccess(ctx, ed)
}

func (r *EnvoyDeploymentReconciler) getDiscoveryService(ctx context.Context, namespace string) (string, error) {
	dsList := &operatorv1alpha1.DiscoveryServiceList{}
	err := r.GetClient().List(ctx, dsList, client.InNamespace(namespace))

	if err != nil {
		return "", err
	}
	if len(dsList.Items) != 1 {
		return "", fmt.Errorf("found an incorrect number of discoveryservices (%d) in namespace '%s'", len(dsList.Items), namespace)
	}
	return dsList.Items[0].GetName(), nil
}

func (r *EnvoyDeploymentReconciler) getEnvoyConfig(ctx context.Context, key types.NamespacedName) (*marin3rv1alpha1.EnvoyConfig, error) {
	ec := &marin3rv1alpha1.EnvoyConfig{}
	err := r.GetClient().Get(ctx, key, ec)

	if err != nil {
		return nil, err
	}

	return ec, nil
}

func (r *EnvoyDeploymentReconciler) getBootstrapConfigHash() string {
	return ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnvoyDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.EnvoyDeployment{}).
		Watches(&source.Channel{Source: r.GetStatusChangeChannel()}, &handler.EnqueueRequestForObject{}).
		// TODO: watch for changes to the EnvoyConfig resource
		// TODO: watch for changes to the DiscoveryService resource
		Complete(r)
}
