package controllers

import (
	"context"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *DiscoveryServiceReconciler) reconcileRoleBinding(ctx context.Context, log logr.Logger) (reconcile.Result, error) {

	existent := &rbacv1.RoleBinding{}
	key := types.NamespacedName{Name: OwnedObjectName(r.ds), Namespace: r.ds.GetNamespace()}
	err := r.Client.Get(ctx, key, existent)

	if err != nil {
		if errors.IsNotFound(err) {
			existent = r.genRoleBindingObject()
			if err := controllerutil.SetControllerReference(r.ds, existent, r.Scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.Client.Create(ctx, existent); err != nil {
				return reconcile.Result{}, err
			}
			log.Info("Created CusterRoleBinding")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// We just reconcile "Subjects" field. "RoleRef" is an immutable field.
	if !equality.Semantic.DeepEqual(existent.RoleRef, r.genRoleBindingObject().RoleRef) ||
		!equality.Semantic.DeepEqual(existent.Subjects, r.genRoleBindingObject().Subjects) {
		patch := client.MergeFrom(existent.DeepCopy())
		existent.Subjects = r.genRoleBindingObject().Subjects
		if err := r.Client.Patch(ctx, existent, patch); err != nil {
			return reconcile.Result{}, err
		}
		log.Info("Patched CusterRoleBinding")
	}

	return reconcile.Result{}, nil
}

func (r *DiscoveryServiceReconciler) genRoleBindingObject() *rbacv1.RoleBinding {

	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OwnedObjectName(r.ds),
			Namespace: r.ds.GetNamespace(),
			Labels:    Labels(r.ds),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     OwnedObjectName(r.ds),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      OwnedObjectName(r.ds),
				Namespace: r.ds.GetNamespace(),
			},
		},
	}
}
