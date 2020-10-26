package controllers

import (
	"context"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
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

func (r *DiscoveryServiceReconciler) reconcileClusterRole(ctx context.Context) (reconcile.Result, error) {

	r.Log.V(1).Info("Reconciling CusterRole")
	existent := &rbacv1.ClusterRole{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: OwnedObjectName(r.ds)}, existent)

	if err != nil {
		if errors.IsNotFound(err) {
			existent = r.genClusterRoleObject()
			if err := controllerutil.SetControllerReference(r.ds, existent, r.Scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.Client.Create(ctx, existent); err != nil {
				return reconcile.Result{}, err
			}
			r.Log.Info("Created CusterRole")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// We just reconcile the "Rules" for the moment
	if !equality.Semantic.DeepEqual(existent.Rules, r.genClusterRoleObject().Rules) {
		patch := client.MergeFrom(existent.DeepCopy())
		existent.Rules = r.genClusterRoleObject().Rules
		if err := r.Client.Patch(ctx, existent, patch); err != nil {
			return reconcile.Result{}, err
		}
		r.Log.Info("Patched CusterRole")
	}

	return reconcile.Result{}, nil
}

func (r *DiscoveryServiceReconciler) genClusterRoleObject() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   OwnedObjectName(r.ds),
			Labels: Labels(r.ds),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{corev1.SchemeGroupVersion.Group},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{envoyv1alpha1.GroupVersion.Group},
				Resources: []string{rbacv1.ResourceAll},
				Verbs:     []string{rbacv1.VerbAll},
			},
		},
	}
}
