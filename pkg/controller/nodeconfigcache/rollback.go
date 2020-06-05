package nodeconfigcache

import (
	"context"
	"fmt"

	"github.com/3scale/marin3r/pkg/apis"
	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	previousVersionPrefix string = "ReceivedPreviousVersion_"
)

// func (r *ReconcileNodeConfigCache) removeRollbackCondition(ctx context.Context, ncc *cachesv1alpha1.NodeConfigCache) error {
// 	patch := client.MergeFrom(ncc.DeepCopy())
// 	ncc.Status.Conditions.RemoveCondition("Rollback")
// 	if err := r.client.Status().Patch(ctx, ncc, patch); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (r *ReconcileNodeConfigCache) removeConfigFailedCondition(ctx context.Context, ncc *cachesv1alpha1.NodeConfigCache) error {
// 	patch := client.MergeFrom(ncc.DeepCopy())
// 	ncc.Status.Conditions.RemoveCondition(cachesv1alpha1.ResourcesOutOfSyncCondition)
// 	if err := r.client.Status().Patch(ctx, ncc, patch); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (r *ReconcileNodeConfigCache) rollback(ctx context.Context, ncc *cachesv1alpha1.NodeConfigCache,
// 	snap *xds_cache.Snapshot, reqLogger logr.Logger) error {

// 	// Read the previous version from the condition
// 	cond := ncc.Status.Conditions.GetCondition(cachesv1alpha1.ResourcesOutOfSyncCondition)
// 	previousVersion := strings.TrimPrefix(string(cond.Reason), previousVersionPrefix)
// 	reqLogger.V(1).Info(fmt.Sprintf("Performing rollback to version '%v'", previousVersion))

// 	// Validate if the rollback has been alredy done
// 	if ncc.Status.PublishedVersion != previousVersion {
// 		// Get the index for the previous version
// 		i := getRevisionIndex(previousVersion, ncc.Status.ConfigRevisions)
// 		if i == nil {
// 			// Version not found in ConfigRevisions
// 			// Update status with "RollbackFailed"
// 			patch := client.MergeFrom(ncc.DeepCopy())
// 			ncc.Status.Conditions.SetCondition(status.Condition{
// 				Type:    "Rollback",
// 				Status:  "True",
// 				Reason:  "RollbackFailed",
// 				Message: fmt.Sprintf("Version '%s' is not in the list of config revisions", previousVersion),
// 			})
// 			if err := r.client.Status().Patch(ctx, ncc, patch); err != nil {
// 				return err
// 			}
// 		}

// 		idx := *i

// 		// Get the ncr for the previous version
// 		revName := ncc.Status.ConfigRevisions[idx].Ref.Name
// 		revNamespace := ncc.Status.ConfigRevisions[idx].Ref.Namespace
// 		ncr := &cachesv1alpha1.NodeConfigRevision{}
// 		if err := r.client.Get(ctx, types.NamespacedName{Name: revName, Namespace: revNamespace}, ncr); err != nil {
// 			return err
// 		}

// 		// Publish snapshot
// 		if err := r.loadResources(ctx, revName, revNamespace, ncc.Spec.Serialization,
// 			&ncr.Spec.Resources, field.NewPath("spec", "resources"), snap); err != nil {
// 			return err
// 		}
// 		if err := (*r.adsCache).SetSnapshot(ncc.Spec.NodeID, *snap); err != nil {
// 			return err
// 		}

// 		// Update status
// 		patch := client.MergeFrom(ncc.DeepCopy())
// 		ncc.Status.PublishedVersion = previousVersion
// 		ncc.Status.Conditions.SetCondition(status.Condition{
// 			Type:    ResourcesUpdateUnsuccessful,
// 			Status:  corev1.ConditionFalse,
// 			Reason:  "RollbackComplete",
// 			Message: fmt.Sprintf("Rollback to version '%s' has been completed", previousVersion),
// 		})
// 		ncc.Status.Conditions.SetCondition(status.Condition{
// 			Type:    "Rollback",
// 			Status:  corev1.ConditionTrue,
// 			Reason:  status.ConditionReason(ResourcesUpdateUnsuccessful),
// 			Message: fmt.Sprintf("Rollback to version '%s' has been completed", previousVersion),
// 		})

// 		err := r.client.Status().Patch(ctx, ncc, patch)
// 		if err != nil {
// 			return fmt.Errorf("rollback: failed to update status: '%v'", err)
// 		}

// 	}
// 	currentIndex := *getRevisionIndex(ncc.Status.PublishedVersion, ncc.Status.ConfigRevisions)
// 	if currentIndex > 0 {

// 	} else {
// 		// Update status with "RollbackFailed"
// 		patch := client.MergeFrom(ncc.DeepCopy())
// 		ncc.Status.Conditions.SetCondition(status.Condition{
// 			Type:    "Rollback",
// 			Status:  "True",
// 			Reason:  "RollbackFailed",
// 			Message: fmt.Sprintf("Rollback failed, no more revisions to try"),
// 		})
// 		// TODO: consider adding retries here
// 		err := r.client.Status().Patch(ctx, ncc, patch)
// 		if err != nil {
// 			return fmt.Errorf("rollback: failed to update status: '%v'", err)
// 		}
// 	}

// 	return nil
// }

// OnError returns a function that should be called when the envoy control plane receives
// a NACK to a discovery response from any of the gateways
func OnError(cfg *rest.Config, namespace string) func(nodeID, version, msg string) error {

	return func(nodeID, version, msg string) error {

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
		ncrList := &cachesv1alpha1.NodeConfigRevisionList{}
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: nodeID, versionTag: version},
		})
		if err != nil {
			return err
		}
		err = cl.List(context.TODO(), ncrList, &client.ListOptions{LabelSelector: selector})
		if err != nil {
			return err
		}
		if len(ncrList.Items) != 1 {
			return fmt.Errorf("Got %v nodeconfigrevision objects when only 1 expected", len(ncrList.Items))
		}

		// Add the "ResourcesUpdateUnsuccessful" condition to the NodeConfigRevision object
		// unless the condition is already set
		ncr := &ncrList.Items[0]
		if !ncr.Status.Conditions.IsTrueFor(cachesv1alpha1.ResourcesOutOfSyncCondition) {
			patch := client.MergeFrom(ncr.DeepCopy())
			ncr.Status.Conditions.SetCondition(status.Condition{
				Type:    "ResourcesUpdateUnsuccessful",
				Status:  "True",
				Reason:  status.ConditionReason(fmt.Sprintf("%s%s", previousVersionPrefix, version)),
				Message: fmt.Sprintf("A gateway returned NACK to the discovery response: '%s'", msg),
			})

			if err := cl.Status().Patch(context.TODO(), ncr, patch); err != nil {
				return err
			}
		}

		return nil
	}
}
