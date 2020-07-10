package discoveryservice

import (
	"context"

	marin3rv1alpha1 "github.com/3scale/marin3r/pkg/apis/marin3r/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileDiscoveryService) reconcileClusterRole(ctx context.Context) (reconcile.Result, error) {

	r.logger.V(1).Info("Reconciling CusterRole")
	existent := &rbacv1.ClusterRole{}
	err := r.client.Get(ctx, types.NamespacedName{Name: r.getName()}, existent)

	if err != nil {
		if errors.IsNotFound(err) {
			existent = r.genClusterRoleObject()
			if err := controllerutil.SetControllerReference(r.ds, existent, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.client.Create(ctx, existent); err != nil {
				return reconcile.Result{}, err
			}
			r.logger.Info("Created CusterRole")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// We just reconcile the "Rules" for the moment
	if !equality.Semantic.DeepEqual(existent.Rules, r.genClusterRoleObject().Rules) {
		patch := client.MergeFrom(existent.DeepCopy())
		existent.Rules = r.genClusterRoleObject().Rules
		if err := r.client.Patch(ctx, existent, patch); err != nil {
			return reconcile.Result{}, err
		}
		r.logger.Info("Patched CusterRole")
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileDiscoveryService) genClusterRoleObject() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getName(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{corev1.SchemeGroupVersion.Group},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{corev1.SchemeGroupVersion.Group},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{corev1.SchemeGroupVersion.Group},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list", "watch", "create"},
			},
			{
				APIGroups: []string{marin3rv1alpha1.SchemeGroupVersion.Group},
				Resources: []string{rbacv1.ResourceAll},
				Verbs:     []string{rbacv1.VerbAll},
			},
		},
	}
}
