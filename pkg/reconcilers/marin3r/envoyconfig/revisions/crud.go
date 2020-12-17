package revisions

import (
	"context"
	"fmt"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/filters"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// List returns the list of EnvoyConfigRevisions owned by the EnvoyConfig
// the reconciler has been instantiated with
func List(ctx context.Context, k8sClient client.Client, namespace string,
	filters ...filters.RevisionFilter) (*marin3rv1alpha1.EnvoyConfigRevisionList, error) {

	list := &marin3rv1alpha1.EnvoyConfigRevisionList{}

	labelSelector := client.MatchingLabels{}
	for _, filter := range filters {
		filter.ApplyToLabelSelector(labelSelector)
	}

	if err := k8sClient.List(ctx, list, labelSelector, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	if len(list.Items) == 0 {
		return nil, NewError(NoMatchesForFilterError, "ListRevisions", fmt.Sprintf("api returned %d EnvoyConfigRevisions", len(list.Items)))
	}
	return list, nil
}

// Get returns the EnvoyConfigRevision that matches the provided filters. If no EnvoyConfigRevisions are returned
// by the API an error is returned. If more than one EnvoyConfigRevision are returned by the API an error is returned.
func Get(ctx context.Context, k8sClient client.Client, namespace string,
	filters ...filters.RevisionFilter) (*marin3rv1alpha1.EnvoyConfigRevision, error) {

	list := &marin3rv1alpha1.EnvoyConfigRevisionList{}

	labelSelector := client.MatchingLabels{}
	for _, filter := range filters {
		filter.ApplyToLabelSelector(labelSelector)
	}

	if err := k8sClient.List(ctx, list, labelSelector, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	if len(list.Items) == 0 {
		return nil, NewError(NoMatchesForFilterError, "GetRevision", "no EnvoyConfigRevisions found")
	} else if len(list.Items) > 1 {
		return nil, NewError(MultipleMatchesForFilterError, "GetRevision", fmt.Sprintf("api returned %d EnvoyConfigRevisions", len(list.Items)))
	}

	return &list.Items[0], nil
}
