package reconcilers

import (
	"context"
	"fmt"

	"github.com/3scale/marin3r/pkg/common"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// DeploymentGeneratorFn is a function that when called returns an appsv1.Deployment object
type DeploymentGeneratorFn func() *appsv1.Deployment

// DeploymentReconciler is a generic Deployment reconciler
type DeploymentReconciler struct {
	ctx    context.Context
	logger logr.Logger
	client client.Client
	scheme *runtime.Scheme
	owner  metav1.Object
}

// NewDeploymentReconciler returns a new DeploymentReconciler object
func NewDeploymentReconciler(ctx context.Context, logger logr.Logger, client client.Client,
	scheme *runtime.Scheme, owner metav1.Object) DeploymentReconciler {
	return DeploymentReconciler{
		ctx:    ctx,
		logger: logger,
		client: client,
		scheme: scheme,
		owner:  owner,
	}
}

// Reconcile reconciles the Deployment object using the given generator
func (r *DeploymentReconciler) Reconcile(o types.NamespacedName, generatorFn DeploymentGeneratorFn) (reconcile.Result, error) {

	deployment := &appsv1.Deployment{}
	err := r.client.Get(r.ctx, types.NamespacedName{Name: o.Name, Namespace: o.Namespace}, deployment)

	if err != nil {
		if errors.IsNotFound(err) {
			r.logger.V(1).Info("Deployment not found")
			deployment := generatorFn()
			if err := controllerutil.SetControllerReference(r.owner, deployment, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.client.Create(r.ctx, deployment); err != nil {
				return reconcile.Result{}, err
			}
			r.logger.Info("Created Deployment")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	desired := generatorFn()

	updated, err := r.reconcileDeployment(deployment, desired)
	if err != nil {
		return reconcile.Result{}, err
	}

	if updated {
		if err := r.client.Update(r.ctx, deployment); err != nil {
			return reconcile.Result{}, err
		}
		r.logger.Info("Deployment Updated")
	}

	return reconcile.Result{}, nil
}

func (r *DeploymentReconciler) reconcileDeployment(existentObj, desiredObj common.KubernetesObject) (bool, error) {
	existent, ok := existentObj.(*appsv1.Deployment)
	if !ok {
		return false, fmt.Errorf("%T is not a *appsv1.Deployment", existentObj)
	}
	desired, ok := desiredObj.(*appsv1.Deployment)
	if !ok {
		return false, fmt.Errorf("%T is not a *appsv1.Deployment", desiredObj)
	}

	updated := false

	// Reconcile the labels
	if !equality.Semantic.DeepEqual(existent.GetLabels(), desired.GetLabels()) {
		r.logger.V(1).Info("Deployment labels need reconcile")
		existent.ObjectMeta.Labels = desired.GetLabels()
		updated = true

	}

	// reconcile the spec
	if !equality.Semantic.DeepEqual(existent.Spec, desired.Spec) {
		r.logger.V(1).Info("Deployment spec needs reconcile")
		existent.Spec = desired.Spec
		updated = true
	}

	return updated, nil
}
