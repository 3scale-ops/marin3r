package revisions

import (
	"sort"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SortByPublication sorts a list of EnvoyConfigRevisions using the following criteria
// - if revision matches desiredVersion, it always goes higher
// - if publication date is defined, higher publication date goes higher
// - if publication date is not defined, higher creation date goes higher
func SortByPublication(desiredVersion string, list *marin3rv1alpha1.EnvoyConfigRevisionList) *marin3rv1alpha1.EnvoyConfigRevisionList {

	ll := list.DeepCopy()

	sort.SliceStable(ll.Items, func(i, j int) bool {

		// Override the chronological sort if either of the candidates is
		// the desired one.
		if ll.Items[j].Spec.Version == desiredVersion {
			return true
		}
		if ll.Items[i].Spec.Version == desiredVersion {
			return false
		}

		// Neither candidate is the desired one, so sort based on
		// timestamps.
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
