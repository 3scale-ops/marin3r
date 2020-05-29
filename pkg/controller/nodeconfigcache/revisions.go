package nodeconfigcache

import (
	"context"
	"fmt"
	"hash/fnv"

	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
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

func (r *ReconcileNodeConfigCache) ensureNodeConfigRevision(ctx context.Context, ncc *cachesv1alpha1.NodeConfigCache) error {

	// Check if a revision already exists for this config version
	ncrList := &cachesv1alpha1.NodeConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: ncc.Spec.NodeID, versionTag: ncc.Spec.Version},
	})
	if err != nil {
		return fmt.Errorf("ensureNodeConfigRevisionError: '%s'", err)
	}
	err = r.client.List(ctx, ncrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		if err != nil {
			return fmt.Errorf("ensureNodeConfigRevisionError: '%s'", err)
		}
	}

	switch len(ncrList.Items) {
	case 0:
		// Create the revission for this config version
		ncr := &cachesv1alpha1.NodeConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      revisionName(ncc.Spec.NodeID, ncc.Spec.Version, ncc.Spec.Resources),
				Namespace: ncc.ObjectMeta.Namespace,
				Labels: map[string]string{
					nodeIDTag:  ncc.Spec.NodeID,
					versionTag: ncc.Spec.Version,
				},
			},
			Spec: cachesv1alpha1.NodeConfigRevisionSpec{
				NodeID:    ncc.Spec.NodeID,
				Version:   ncc.Spec.Version,
				Resources: *ncc.Spec.Resources,
			},
		}
		// Set the ncc as the owner and controller of the revision
		if err := controllerutil.SetControllerReference(ncc, ncr, r.scheme); err != nil {
			return fmt.Errorf("ensureNodeConfigRevisionError: '%s'", err)
		}
		err = r.client.Create(ctx, ncr)
		if err != nil {
			return fmt.Errorf("ensureNodeConfigRevisionError: '%s'", err)
		}
	case 1:
		// Revision already exists for this config
		return nil
	default:
		return fmt.Errorf("ensureNodeConfigRevision: more than one revision exists for config version '%s', cannot reconcile", ncc.Spec.Version)
	}

	return nil
}

func (r *ReconcileNodeConfigCache) consolidateRevisionList(ctx context.Context, ncc *cachesv1alpha1.NodeConfigCache) error {

	// Check if the revision is already in the list of revisions
	if getRevisionIndex(ncc.Spec.Version, ncc.Status.ConfigRevisions) != nil {
		// The version is already present in the ConfigRevision list
		return nil
	}

	// Get the revision name that matches nodeID and version
	ncrList := &cachesv1alpha1.NodeConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: ncc.Spec.NodeID, versionTag: ncc.Spec.Version},
	})
	if err != nil {
		return fmt.Errorf("consolidateRevisionList: '%s'", err)
	}
	err = r.client.List(ctx, ncrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		if err != nil {
			return fmt.Errorf("consolidateRevisionList: '%s'", err)
		}
	}

	switch len(ncrList.Items) {
	case 0:
		return fmt.Errorf("consolidateRevisionList: no config revision found for version '%s'", ncc.Spec.Version)
	case 1:
		// Update the revision list in the NCC status
		patch := client.MergeFrom(ncc.DeepCopy())
		ncc.Status.ConfigRevisions = append(ncc.Status.ConfigRevisions, cachesv1alpha1.ConfigRevisionRef{
			Version: ncc.Spec.Version,
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
			return fmt.Errorf("consolidateRevisionList: failed to update the config revision list: '%v'", err)
		}
	default:
		return fmt.Errorf("consolidateRevisionList: more than one revision exists for config version '%s', cannot reconcile", ncc.Spec.Version)
	}

	return nil
}

func (r *ReconcileNodeConfigCache) deleteUnreferencedRevisions(ctx context.Context, ncc *cachesv1alpha1.NodeConfigCache) error {
	// Get all revisions that belong to this ncc
	ncrList := &cachesv1alpha1.NodeConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: ncc.Spec.NodeID},
	})
	if err != nil {
		return fmt.Errorf("deleteUnreferencedRevisions: '%s'", err)
	}
	err = r.client.List(ctx, ncrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		if err != nil {
			return fmt.Errorf("deleteUnreferencedRevisions: '%s'", err)
		}
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

func trimRevisions(list []cachesv1alpha1.ConfigRevisionRef, max int) []cachesv1alpha1.ConfigRevisionRef {
	for len(list) > max {
		list = list[1:]
	}
	return list
}

func revisionName(nodeID, version string, resources *cachesv1alpha1.EnvoyResources) string {
	resourcesHasher := fnv.New32a()
	hashutil.DeepHashObject(resourcesHasher, resources)
	hash := rand.SafeEncodeString(fmt.Sprint(resourcesHasher.Sum32()))
	return fmt.Sprintf("%s-%s-%s", nodeID, version, hash)
}

func getRevisionIndex(version string, revisions []cachesv1alpha1.ConfigRevisionRef) *int {
	for idx, rev := range revisions {
		if rev.Version == version {
			return &idx
		}
	}
	return nil
}
