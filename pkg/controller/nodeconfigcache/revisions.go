package nodeconfigcache

import (
	"context"
	"fmt"
	"hash/fnv"

	marin3rv1alpha1 "github.com/3scale/marin3r/pkg/apis/marin3r/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	nodeIDTag    = "marin3r.3scale.net/node-id"
	versionTag   = "marin3r.3scale.net/config-version"
	maxRevisions = 10
)

func (r *ReconcileNodeConfigCache) ensureNodeConfigRevision(ctx context.Context,
	ncc *marin3rv1alpha1.NodeConfigCache, version string) error {

	// Get the list of revisions for the current version
	ncrList := &marin3rv1alpha1.NodeConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: ncc.Spec.NodeID, versionTag: version},
	})
	if err != nil {
		return newCacheError(UnknownError, "ensureNodeConfigRevision", err.Error())
	}
	err = r.client.List(ctx, ncrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		if err != nil {
			return newCacheError(UnknownError, "ensureNodeConfigRevision", err.Error())
		}
	}

	// Got wrong number of revisions
	if len(ncrList.Items) > 1 {
		return newCacheError(UnknownError, "ensureNodeConfigRevision", fmt.Sprintf("more than one revision exists for config version '%s', cannot reconcile", version))
	}

	// Revision does not yet exists, create one
	if len(ncrList.Items) == 0 {
		// Create the revision for this config version
		ncr := &marin3rv1alpha1.NodeConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", ncc.Spec.NodeID, version),
				Namespace: ncc.ObjectMeta.Namespace,
				Labels: map[string]string{
					nodeIDTag:  ncc.Spec.NodeID,
					versionTag: version,
				},
			},
			Spec: marin3rv1alpha1.NodeConfigRevisionSpec{
				NodeID:        ncc.Spec.NodeID,
				Version:       version,
				Serialization: ncc.Spec.Serialization,
				Resources:     ncc.Spec.Resources,
			},
		}
		// Set the ncc as the owner and controller of the revision
		if err := controllerutil.SetControllerReference(ncc, ncr, r.scheme); err != nil {
			return newCacheError(UnknownError, "ensureNodeConfigRevision", err.Error())
		}
		err = r.client.Create(ctx, ncr)
		if err != nil {
			return newCacheError(UnknownError, "ensureNodeConfigRevision", err.Error())
		}
	}

	return nil
}

func (r *ReconcileNodeConfigCache) consolidateRevisionList(ctx context.Context,
	ncc *marin3rv1alpha1.NodeConfigCache, version string) error {

	// This code handles the case in which a revision already exists for
	// this version. We must ensure that this version is at the last position
	// of the ConfigRevision list to keep the order of published versions
	{
		if idx := getRevisionIndex(version, ncc.Status.ConfigRevisions); idx != nil {
			// The version is already present in the ConfigRevision list
			if *idx < len(ncc.Status.ConfigRevisions)-1 {
				patch := client.MergeFrom(ncc.DeepCopy())
				ncc.Status.ConfigRevisions = moveRevisionToLast(ncc.Status.ConfigRevisions, *idx)
				if err := r.client.Status().Patch(ctx, ncc, patch); err != nil {
					return newCacheError(UnknownError, "consolidateRevisionList", err.Error())
				}
			}
			return nil
		}
	}

	// This code handles the case in which a revision does not yet exist
	// for the given version
	{
		// Get the revision name that matches nodeID and version
		ncrList := &marin3rv1alpha1.NodeConfigRevisionList{}
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ncc.Spec.NodeID, versionTag: version},
		})
		if err != nil {
			return newCacheError(UnknownError, "consolidateRevisionList", err.Error())
		}
		err = r.client.List(ctx, ncrList, &client.ListOptions{LabelSelector: selector})
		if err != nil {
			if err != nil {
				return newCacheError(UnknownError, "consolidateRevisionList", err.Error())
			}
		}

		if len(ncrList.Items) == 1 {

			// Update the revision list in the NCC status
			patch := client.MergeFrom(ncc.DeepCopy())
			ncc.Status.ConfigRevisions = append(ncc.Status.ConfigRevisions, marin3rv1alpha1.ConfigRevisionRef{
				Version: version,
				Ref: corev1.ObjectReference{
					Kind:       ncrList.Items[0].Kind,
					Name:       ncrList.Items[0].ObjectMeta.Name,
					Namespace:  ncrList.Items[0].Namespace,
					UID:        ncrList.Items[0].UID,
					APIVersion: ncrList.Items[0].APIVersion,
				},
			})

			// Remove old revisions if max have been reached
			ncc.Status.ConfigRevisions = trimRevisions(ncc.Status.ConfigRevisions, maxRevisions)

			// TODO: might need to do retries here, this is pretty critical
			err = r.client.Status().Patch(ctx, ncc, patch)
			if err != nil {
				return newCacheError(UnknownError, "consolidateRevisionList", err.Error())
			}
		} else {
			return newCacheError(UnknownError, "consolidateRevisionList", fmt.Sprintf("expected just one revision for version '%s', but got '%v'", version, len(ncrList.Items)))
		}
	}

	return nil
}

func (r *ReconcileNodeConfigCache) deleteUnreferencedRevisions(ctx context.Context, ncc *marin3rv1alpha1.NodeConfigCache) error {
	// Get all revisions that belong to this ncc
	ncrList := &marin3rv1alpha1.NodeConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: ncc.Spec.NodeID},
	})
	if err != nil {
		return newCacheError(UnknownError, "deleteUnreferencedRevisions", err.Error())
	}
	err = r.client.List(ctx, ncrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return newCacheError(UnknownError, "deleteUnreferencedRevisions", err.Error())
	}

	// For each of the revisions, check if they are still refrered from the ncc
	for _, ncr := range ncrList.Items {
		if getRevisionIndex(ncr.Spec.Version, ncc.Status.ConfigRevisions) == nil {
			// Keep going even if the deletion operation returns error, we really don care,
			// the ncr will get eventually deleted in a future reconcile loop
			r.client.Delete(ctx, &ncr)
		}
	}

	return nil
}

// markRevisionPublished marks the revision that matches the provided version as the one
// to be set in the xds server cache:
//  - It will first set the 'RevisionPublished' condition to false in the current published revision
//  - It will set the 'RevisionPublished' condition to true in the revision that matches the given version
// This ensures that at a given point in time 0 or 1 revisions can have the 'PublishedRevision' to true, being
// 1 the case most of the time
func (r *ReconcileNodeConfigCache) markRevisionPublished(ctx context.Context, nodeID, version, reason, msg string) error {

	// Get all revisions for this NCC
	ncrList := &marin3rv1alpha1.NodeConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: nodeID},
	})
	if err != nil {
		return newCacheError(UnknownError, "markRevisionPublished", err.Error())
	}
	err = r.client.List(ctx, ncrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return newCacheError(UnknownError, "markRevisionPublished", err.Error())
	}

	// Set 'RevisionPublished' to false for all revisions
	for _, ncr := range ncrList.Items {
		if ncr.Spec.Version != version && ncr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			patch := client.MergeFrom(ncr.DeepCopy())
			ncr.Status.Conditions.SetCondition(status.Condition{
				Type:    marin3rv1alpha1.RevisionPublishedCondition,
				Status:  corev1.ConditionFalse,
				Reason:  status.ConditionReason("OtherVersionPublished"),
				Message: msg,
			})

			if err := r.client.Status().Patch(ctx, &ncr, patch); err != nil {
				return newCacheError(UnknownError, "markRevisionPublished", err.Error())
			}
		}
	}

	// NOTE: from this point on, if something fails we end up with 0 revisions
	// marked as published. Shouldn't be a problem as the current version
	// is already being served by the xds server and should be fixed eventually
	// in another reconcile

	// Set the the revision that holds the given version with 'RevisionPublished' = True
	ncrList = &marin3rv1alpha1.NodeConfigRevisionList{}
	selector, err = metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: nodeID, versionTag: version},
	})

	if err != nil {
		return newCacheError(UnknownError, "markRevisionPublished", err.Error())
	}

	err = r.client.List(ctx, ncrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return newCacheError(UnknownError, "markRevisionPublished", err.Error())
	}

	if len(ncrList.Items) != 1 {
		return newCacheError(UnknownError, "markRevisionPublished", fmt.Sprintf("found unexpected number of nodeconfigrevisions matching version '%s'", version))
	}

	ncr := ncrList.Items[0]
	patch := client.MergeFrom(ncr.DeepCopy())
	ncr.Status.Conditions.SetCondition(status.Condition{
		Type:    marin3rv1alpha1.RevisionPublishedCondition,
		Status:  corev1.ConditionTrue,
		Reason:  status.ConditionReason(reason),
		Message: msg,
	})

	if err := r.client.Status().Patch(ctx, &ncr, patch); err != nil {
		return newCacheError(UnknownError, "markRevisionPublished", err.Error())
	}

	return nil
}

func trimRevisions(list []marin3rv1alpha1.ConfigRevisionRef, max int) []marin3rv1alpha1.ConfigRevisionRef {
	for len(list) > max {
		list = list[1:]
	}
	return list
}

func calculateRevisionHash(resources *marin3rv1alpha1.EnvoyResources) string {
	resourcesHasher := fnv.New32a()
	hashutil.DeepHashObject(resourcesHasher, resources)
	return rand.SafeEncodeString(fmt.Sprint(resourcesHasher.Sum32()))
}

func getRevisionIndex(version string, revisions []marin3rv1alpha1.ConfigRevisionRef) *int {
	for idx, rev := range revisions {
		if rev.Version == version {
			return &idx
		}
	}
	return nil
}

func moveRevisionToLast(list []marin3rv1alpha1.ConfigRevisionRef, idx int) []marin3rv1alpha1.ConfigRevisionRef {

	return append(list[:idx], append(list[idx+1:], list[idx])...)
}
