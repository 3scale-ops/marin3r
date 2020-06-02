package nodeconfigcache

import (
	"context"
	"fmt"

	"github.com/3scale/marin3r/pkg/apis"
	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileNodeConfigCache) removeRollbackCondition(ctx context.Context, ncc *cachesv1alpha1.NodeConfigCache) error {
	patch := client.MergeFrom(ncc.DeepCopy())
	ncc.Status.Conditions.RemoveCondition("Rollback")
	if err := r.client.Status().Patch(ctx, ncc, patch); err != nil {
		return err
	}
	return nil
}

func (r *ReconcileNodeConfigCache) removeConfigFailedCondition(ctx context.Context, ncc *cachesv1alpha1.NodeConfigCache) error {
	patch := client.MergeFrom(ncc.DeepCopy())
	ncc.Status.Conditions.RemoveCondition("ConfigFailed")
	if err := r.client.Status().Patch(ctx, ncc, patch); err != nil {
		return err
	}
	return nil
}

func (r *ReconcileNodeConfigCache) rollback(ctx context.Context, ncc *cachesv1alpha1.NodeConfigCache,
	snap *xds_cache.Snapshot, reqLogger logr.Logger) error {
	// TODO: mark current PublishedVersion as failed
	currentIndex := *getRevisionIndex(ncc.Status.PublishedVersion, ncc.Status.ConfigRevisions)
	if currentIndex > 0 {
		rollbackToIndex := currentIndex - 1
		reqLogger.V(1).Info(fmt.Sprintf("Performing rollback to index '%v'", currentIndex))
		// Get the revision
		revName := ncc.Status.ConfigRevisions[rollbackToIndex].Ref.Name
		revNamespace := ncc.Status.ConfigRevisions[rollbackToIndex].Ref.Namespace
		ncr := &cachesv1alpha1.NodeConfigRevision{}
		if err := r.client.Get(ctx, types.NamespacedName{Name: revName, Namespace: revNamespace}, ncr); err != nil {
			return err
		}
		// Load resources from the revision
		if err := r.loadResources(ctx, revName, revNamespace, ncc.Spec.Serialization,
			&ncr.Spec.Resources, field.NewPath("spec", "resources"), snap); err != nil {
			return err
		}

		// Push resources to xds server cache
		if err := (*r.adsCache).SetSnapshot(ncc.Spec.NodeID, *snap); err != nil {
			return err
		}

		// Update status
		patch := client.MergeFrom(ncc.DeepCopy())
		ncc.Status.PublishedVersion = ncc.Status.ConfigRevisions[rollbackToIndex].Version
		ncc.Status.Conditions.SetCondition(status.Condition{
			Type:    "Rollback",
			Status:  "True",
			Reason:  "RollbackComplete",
			Message: fmt.Sprintf("Rollback to version '%s' has been completed", ncc.Status.PublishedVersion),
		})
		// TODO: consider adding retries here
		err := r.client.Status().Patch(ctx, ncc, patch)
		if err != nil {
			return fmt.Errorf("rollback: failed to update status: '%v'", err)
		}
	} else {
		// Update status with "RollbackFailed"
		patch := client.MergeFrom(ncc.DeepCopy())
		ncc.Status.Conditions.SetCondition(status.Condition{
			Type:    "Rollback",
			Status:  "True",
			Reason:  "RollbackFailed",
			Message: fmt.Sprintf("Rollback failed, no more revisions to try"),
		})
		// TODO: consider adding retries here
		err := r.client.Status().Patch(ctx, ncc, patch)
		if err != nil {
			return fmt.Errorf("rollback: failed to update status: '%v'", err)
		}
	}

	return nil
}

// OnError returns a function that should be called when the envoy control plane receives
// an error from any of the gateways
func OnError(cfg *rest.Config, namespace string) func(nodeID string) error {

	return func(nodeID string) error {

		// Create a client and register CRDs
		s := runtime.NewScheme()
		if err := apis.AddToScheme(s); err != nil {
			return err
		}
		cl, err := client.New(cfg, client.Options{Scheme: s})
		if err != nil {
			return err
		}

		// Get the nodeconfigcache that corresponds to the envoy node that returned the error
		nccList := &cachesv1alpha1.NodeConfigCacheList{}
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: nodeID},
		})
		if err != nil {
			return err
		}
		err = cl.List(context.TODO(), nccList, &client.ListOptions{LabelSelector: selector})
		if err != nil {
			return err
		}

		if len(nccList.Items) != 1 {
			return fmt.Errorf("Got %v NodeConfigCache objects for nodeID '%s'", len(nccList.Items), nodeID)
		}
		ncc := &nccList.Items[0]

		// Add the "ConfigFailed" condition to the NodeConfigCache object
		// unless the Rollback condition already exists
		if !ncc.Status.Conditions.IsTrueFor("ConfigFailed") {
			patch := client.MergeFrom(ncc.DeepCopy())
			ncc.Status.Conditions.SetCondition(status.Condition{
				Type:    "ConfigFailed",
				Status:  "True",
				Reason:  "GatewayError",
				Message: "A gateway returned an error trying to load the resources",
			})

			if err := cl.Status().Patch(context.TODO(), ncc, patch); err != nil {
				return err
			}
		}

		return nil
	}
}
