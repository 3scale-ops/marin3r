package discoveryservice

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileDiscoveryService) reconcileClusterRoleBinding(ctx context.Context) (reconcile.Result, error) {

	r.logger.V(1).Info("Reconciling CusterRoleBinding")
	existent := &rbacv1.ClusterRoleBinding{}
	err := r.client.Get(ctx, types.NamespacedName{Name: OwnedObjectName(r.ds)}, existent)

	if err != nil {
		if errors.IsNotFound(err) {
			existent = r.genClusterRoleBindingObject()
			if err := controllerutil.SetControllerReference(r.ds, existent, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.client.Create(ctx, existent); err != nil {
				return reconcile.Result{}, err
			}
			r.logger.Info("Created CusterRoleBinding")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// We just reconcile "Subjects" field. "RoleRef" is an immutable field.
	if !equality.Semantic.DeepEqual(existent.RoleRef, r.genClusterRoleBindingObject().RoleRef) ||
		!equality.Semantic.DeepEqual(existent.Subjects, r.genClusterRoleBindingObject().Subjects) {
		patch := client.MergeFrom(existent.DeepCopy())
		existent.Subjects = r.genClusterRoleBindingObject().Subjects
		if err := r.client.Patch(ctx, existent, patch); err != nil {
			return reconcile.Result{}, err
		}
		r.logger.Info("Patched CusterRoleBinding")
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileDiscoveryService) genClusterRoleBindingObject() *rbacv1.ClusterRoleBinding {

	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   OwnedObjectName(r.ds),
			Labels: Labels(r.ds),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     OwnedObjectName(r.ds),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      OwnedObjectName(r.ds),
				Namespace: OwnedObjectNamespace(r.ds),
			},
		},
	}
}
