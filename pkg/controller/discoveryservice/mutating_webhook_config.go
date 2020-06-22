package discoveryservice

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileMutatingWebhook keeps the marin3r MutatingWebhookConfiguration object in sync with the desired state
func (r *ReconcileDiscoveryService) reconcileMutatingWebhook(ctx context.Context) (reconcile.Result, error) {

	return reconcile.Result{}, nil
}
