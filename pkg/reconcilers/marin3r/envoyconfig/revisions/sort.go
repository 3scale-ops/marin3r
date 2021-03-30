package revisions

import (
	"sort"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SortByPublication sorts a list of EnvoyConfigRevisions using the following criteria
// - if revision matches currentResourcesVersion, it always goes higher
// - if publication date is defined, higher publication date goes higher
// - if publication date is not defined, higher creation date goes higher
func SortByPublication(currentResourcesVersion string, list *marin3rv1alpha1.EnvoyConfigRevisionList) *marin3rv1alpha1.EnvoyConfigRevisionList {

	ll := list.DeepCopy()

	sort.SliceStable(ll.Items, func(i, j int) bool {
		if ll.Items[j].Spec.Version == currentResourcesVersion {
			return true
		}

		var iTime, jTime metav1.Time
		if ll.Items[i].Status.LastPublishedAt.IsZero() {
			iTime = ll.Items[i].GetCreationTimestamp()
		} else {
			iTime = *ll.Items[i].Status.LastPublishedAt
		}

		if ll.Items[j].Status.LastPublishedAt.IsZero() {
			jTime = ll.Items[j].GetCreationTimestamp()
		} else {
			jTime = *ll.Items[j].Status.LastPublishedAt
		}

		return iTime.Before(&jTime)
	})

	return ll
}
