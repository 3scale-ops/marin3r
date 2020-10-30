package envoy

import (
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/golang/protobuf/proto"
)

// ResourcesEqual validates that the envoy resources are
// exactly the same in for both arrays a and b. It uses proto.Equal()
// to assert the equality between two given envoy resources.
func ResourcesEqual(a, b [6]xds_cache.Resources) bool {

	// Check resources are equal for each resource type
	for rtype, aResources := range a {
		bResources := b[rtype]

		// If lenght is not equal, resources are not equal
		if len(aResources.Items) != len(bResources.Items) {
			return false
		}

		for name, bValue := range bResources.Items {

			aValue, ok := aResources.Items[name]

			// If some key does not exist, resources are not equal
			if !ok {
				return false
			}

			// If value has changed, resources are not equal
			if !proto.Equal(bValue, aValue) {
				return false
			}
		}
	}
	return true
}
