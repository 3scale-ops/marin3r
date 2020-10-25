package controllers

import (
	"context"
	"fmt"
	"hash/fnv"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	common "github.com/3scale/marin3r/pkg/common"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	nodeIDTag    = "marin3r.3scale.net/node-id"
	versionTag   = "marin3r.3scale.net/config-version"
	maxRevisions = 10
)

func (r *EnvoyConfigReconciler) ensureEnvoyConfigRevision(ctx context.Context,
	ec *envoyv1alpha1.EnvoyConfig, version string) error {

	// Get the list of revisions for the current version
	ecrList := &envoyv1alpha1.EnvoyConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID, versionTag: version},
	})
	if err != nil {
		return newCacheError(UnknownError, "ensureEnvoyConfigRevision", err.Error())
	}
	err = r.Client.List(ctx, ecrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		if err != nil {
			return newCacheError(UnknownError, "ensureEnvoyConfigRevision", err.Error())
		}
	}

	// Got wrong number of revisions
	if len(ecrList.Items) > 1 {
		return newCacheError(UnknownError, "ensureEnvoyConfigRevision", fmt.Sprintf("more than one revision exists for config version '%s', cannot reconcile", version))
	}

	// Revision does not yet exists, create one
	if len(ecrList.Items) == 0 {
		// Create the revision for this config version
		ecr := &envoyv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", ec.Spec.NodeID, version),
				Namespace: ec.ObjectMeta.Namespace,
				Labels: map[string]string{
					nodeIDTag:  ec.Spec.NodeID,
					versionTag: version,
				},
			},
			Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
				NodeID:         ec.Spec.NodeID,
				Version:        version,
				Serialization:  ec.Spec.Serialization,
				EnvoyResources: ec.Spec.EnvoyResources,
			},
		}
		// Set the ec as the owner and controller of the revision
		if err := controllerutil.SetControllerReference(ec, ecr, r.Scheme); err != nil {
			return newCacheError(UnknownError, "ensureEnvoyConfigRevision", err.Error())
		}
		err = r.Client.Create(ctx, ecr)
		if err != nil {
			return newCacheError(UnknownError, "ensureEnvoyConfigRevision", err.Error())
		}
	}

	return nil
}

func (r *EnvoyConfigReconciler) consolidateRevisionList(ctx context.Context,
	ec *envoyv1alpha1.EnvoyConfig, version string) error {

	// This code handles the case in which a revision already exists for
	// this version. We must ensure that this version is at the last position
	// of the ConfigRevision list to keep the order of published versions
	{
		if idx := getRevisionIndex(version, ec.Status.ConfigRevisions); idx != nil {
			// The version is already present in the ConfigRevision list
			if *idx < len(ec.Status.ConfigRevisions)-1 {
				patch := client.MergeFrom(ec.DeepCopy())
				ec.Status.ConfigRevisions = moveRevisionToLast(ec.Status.ConfigRevisions, *idx)
				if err := r.Client.Status().Patch(ctx, ec, patch); err != nil {
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
		ecrList := &envoyv1alpha1.EnvoyConfigRevisionList{}
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID, versionTag: version},
		})
		if err != nil {
			return newCacheError(UnknownError, "consolidateRevisionList", err.Error())
		}
		err = r.Client.List(ctx, ecrList, &client.ListOptions{LabelSelector: selector})
		if err != nil {
			if err != nil {
				return newCacheError(UnknownError, "consolidateRevisionList", err.Error())
			}
		}

		if len(ecrList.Items) == 1 {

			// Update the revision list in the EC status
			patch := client.MergeFrom(ec.DeepCopy())
			ec.Status.ConfigRevisions = append(ec.Status.ConfigRevisions, envoyv1alpha1.ConfigRevisionRef{
				Version: version,
				Ref: corev1.ObjectReference{
					Kind:       ecrList.Items[0].Kind,
					Name:       ecrList.Items[0].ObjectMeta.Name,
					Namespace:  ecrList.Items[0].Namespace,
					UID:        ecrList.Items[0].UID,
					APIVersion: ecrList.Items[0].APIVersion,
				},
			})

			// Remove old revisions if max have been reached
			ec.Status.ConfigRevisions = trimRevisions(ec.Status.ConfigRevisions, maxRevisions)

			// TODO: might need to do retries here, this is pretty critical
			err = r.Client.Status().Patch(ctx, ec, patch)
			if err != nil {
				return newCacheError(UnknownError, "consolidateRevisionList", err.Error())
			}
		} else {
			return newCacheError(UnknownError, "consolidateRevisionList", fmt.Sprintf("expected just one revision for version '%s', but got '%v'", version, len(ecrList.Items)))
		}
	}

	return nil
}

func (r *EnvoyConfigReconciler) deleteUnreferencedRevisions(ctx context.Context, ec *envoyv1alpha1.EnvoyConfig) error {
	// Get all revisions that belong to this ec
	ecrList := &envoyv1alpha1.EnvoyConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID},
	})
	if err != nil {
		return newCacheError(UnknownError, "deleteUnreferencedRevisions", err.Error())
	}
	err = r.Client.List(ctx, ecrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return newCacheError(UnknownError, "deleteUnreferencedRevisions", err.Error())
	}

	// For each of the revisions, check if they are still refrered from the ec
	for _, ecr := range ecrList.Items {
		if getRevisionIndex(ecr.Spec.Version, ec.Status.ConfigRevisions) == nil {
			// Keep going even if the deletion operation returns error, we really don care,
			// the ecr will get eventually deleted in a future reconcile loop
			r.Client.Delete(ctx, &ecr)
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
func (r *EnvoyConfigReconciler) markRevisionPublished(ctx context.Context, nodeID, version, reason, msg string) error {

	// Get all revisions for this NCC
	ecrList := &envoyv1alpha1.EnvoyConfigRevisionList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: nodeID},
	})
	if err != nil {
		return newCacheError(UnknownError, "markRevisionPublished", err.Error())
	}
	err = r.Client.List(ctx, ecrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return newCacheError(UnknownError, "markRevisionPublished", err.Error())
	}

	// Set 'RevisionPublished' to false for all revisions
	for _, ecr := range ecrList.Items {
		if ecr.Spec.Version != version && ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition) {
			patch := client.MergeFrom(ecr.DeepCopy())
			ecr.Status.Conditions.SetCondition(status.Condition{
				Type:    envoyv1alpha1.RevisionPublishedCondition,
				Status:  corev1.ConditionFalse,
				Reason:  status.ConditionReason("OtherVersionPublished"),
				Message: msg,
			})

			if err := r.Client.Status().Patch(ctx, &ecr, patch); err != nil {
				return newCacheError(UnknownError, "markRevisionPublished", err.Error())
			}
		}
	}

	// NOTE: from this point on, if something fails we end up with 0 revisions
	// marked as published. Shouldn't be a problem as the current version
	// is already being served by the xds server and should be fixed eventually
	// in another reconcile

	// Set the the revision that holds the given version with 'RevisionPublished' = True
	ecrList = &envoyv1alpha1.EnvoyConfigRevisionList{}
	selector, err = metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: nodeID, versionTag: version},
	})

	if err != nil {
		return newCacheError(UnknownError, "markRevisionPublished", err.Error())
	}

	err = r.Client.List(ctx, ecrList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return newCacheError(UnknownError, "markRevisionPublished", err.Error())
	}

	if len(ecrList.Items) != 1 {
		return newCacheError(UnknownError, "markRevisionPublished", fmt.Sprintf("found unexpected number of envoyconfigrevisions matching version '%s'", version))
	}

	ecr := ecrList.Items[0]
	patch := client.MergeFrom(ecr.DeepCopy())
	ecr.Status.Conditions.SetCondition(status.Condition{
		Type:    envoyv1alpha1.RevisionPublishedCondition,
		Status:  corev1.ConditionTrue,
		Reason:  status.ConditionReason(reason),
		Message: msg,
	})

	if err := r.Client.Status().Patch(ctx, &ecr, patch); err != nil {
		return newCacheError(UnknownError, "markRevisionPublished", err.Error())
	}

	return nil
}

func trimRevisions(list []envoyv1alpha1.ConfigRevisionRef, max int) []envoyv1alpha1.ConfigRevisionRef {
	for len(list) > max {
		list = list[1:]
	}
	return list
}

func calculateRevisionHash(resources *envoyv1alpha1.EnvoyResources) string {
	resourcesHasher := fnv.New32a()
	common.DeepHashObject(resourcesHasher, resources)
	return rand.SafeEncodeString(fmt.Sprint(resourcesHasher.Sum32()))
}

func getRevisionIndex(version string, revisions []envoyv1alpha1.ConfigRevisionRef) *int {
	for idx, rev := range revisions {
		if rev.Version == version {
			return &idx
		}
	}
	return nil
}

func moveRevisionToLast(list []envoyv1alpha1.ConfigRevisionRef, idx int) []envoyv1alpha1.ConfigRevisionRef {

	return append(list[:idx], append(list[idx+1:], list[idx])...)
}
