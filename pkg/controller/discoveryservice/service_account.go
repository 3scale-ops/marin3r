package discoveryservice

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileDiscoveryService) reconcileServiceAccount(ctx context.Context) (reconcile.Result, error) {

	r.logger.V(1).Info("Reconciling ServiceAccount")
	existent := &corev1.ServiceAccount{}
	err := r.client.Get(ctx, types.NamespacedName{Name: OwnedObjectName(r.ds), Namespace: OwnedObjectNamespace(r.ds)}, existent)

	if err != nil {
		if errors.IsNotFound(err) {
			existent = r.genServiceAccountObject()
			if err := controllerutil.SetControllerReference(r.ds, existent, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.client.Create(ctx, existent); err != nil {
				return reconcile.Result{}, err
			}
			r.logger.Info("Created ServiceAccount")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Nothing to reconcile in a ServiceAccount object

	return reconcile.Result{}, nil
}

func (r *ReconcileDiscoveryService) genServiceAccountObject() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OwnedObjectName(r.ds),
			Namespace: OwnedObjectNamespace(r.ds),
		},
	}
}
