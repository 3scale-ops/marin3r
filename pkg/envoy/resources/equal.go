package envoy

import (
	"github.com/3scale-ops/marin3r/pkg/envoy"
	"google.golang.org/protobuf/proto"
)

// ResourcesEqual validates that the given maps of "name - resource" pairs
// are equal. It uses proto.Equal() to assert the equality between two given envoy resources.
func ResourcesEqual(a, b map[string]envoy.Resource) bool {

	if len(a) != len(b) {
		return false
	}

	for name, resource := range a {
		if !proto.Equal(resource, b[name]) {
			return false
		}
	}

	return true
}
