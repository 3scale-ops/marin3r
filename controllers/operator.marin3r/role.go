package controllers

import (
	"context"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/go-logr/logr"
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

func (r *DiscoveryServiceReconciler) reconcileRole(ctx context.Context, log logr.Logger) (reconcile.Result, error) {

	existent := &rbacv1.Role{}
	key := types.NamespacedName{Name: OwnedObjectName(r.ds), Namespace: r.ds.GetNamespace()}
	err := r.Client.Get(ctx, key, existent)

	if err != nil {
		if errors.IsNotFound(err) {
			existent = r.genRoleObject()
			if err := controllerutil.SetControllerReference(r.ds, existent, r.Scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.Client.Create(ctx, existent); err != nil {
				return reconcile.Result{}, err
			}
			log.Info("Created Role")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// We just reconcile the "Rules" for the moment
	if !equality.Semantic.DeepEqual(existent.Rules, r.genRoleObject().Rules) {
		patch := client.MergeFrom(existent.DeepCopy())
		existent.Rules = r.genRoleObject().Rules
		if err := r.Client.Patch(ctx, existent, patch); err != nil {
			return reconcile.Result{}, err
		}
		log.Info("Patched CusterRole")
	}

	return reconcile.Result{}, nil
}

func (r *DiscoveryServiceReconciler) genRoleObject() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OwnedObjectName(r.ds),
			Namespace: r.ds.GetNamespace(),
			Labels:    Labels(r.ds),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{corev1.SchemeGroupVersion.Group},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{marin3rv1alpha1.GroupVersion.Group},
				Resources: []string{rbacv1.ResourceAll},
				Verbs:     []string{rbacv1.VerbAll},
			},
		},
	}
}
