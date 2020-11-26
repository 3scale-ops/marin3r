package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *DiscoveryServiceReconciler) reconcileServiceAccount(ctx context.Context) (reconcile.Result, error) {

	// r.Log.V(1).Info("Reconciling ServiceAccount")
	existent := &corev1.ServiceAccount{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: OwnedObjectName(r.ds), Namespace: OwnedObjectNamespace(r.ds)}, existent)

	if err != nil {
		if errors.IsNotFound(err) {
			existent = r.genServiceAccountObject()
			if err := controllerutil.SetControllerReference(r.ds, existent, r.Scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.Client.Create(ctx, existent); err != nil {
				return reconcile.Result{}, err
			}
			r.Log.Info("Created ServiceAccount")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Nothing to reconcile in a ServiceAccount object

	return reconcile.Result{}, nil
}

func (r *DiscoveryServiceReconciler) genServiceAccountObject() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OwnedObjectName(r.ds),
			Namespace: OwnedObjectNamespace(r.ds),
			Labels:    Labels(r.ds),
		},
	}
}
