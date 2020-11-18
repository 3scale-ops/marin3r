package envoy

import (
	"fmt"

	"github.com/golang/protobuf/proto"
)

// Resource is the base interface for the xDS payload.
// Any envoy resource type implements this interface.
type Resource interface {
	proto.Message
}

// APIVersion is an enum with the supported envoy
// API versions.
type APIVersion string

const (
	// APIv2 is the envoy v2 API version.
	APIv2 APIVersion = "v2"
	// APIv3 is the envoy v2 API version.
	APIv3 APIVersion = "v3"
)

// String returns the string representation of APIVersion
func (version APIVersion) String() string {
	switch version {
	case APIv3:
		return string(APIv3)
	default:
		return string(APIv2)
	}
}

// ParseAPIVersion returns an APIVersion for the given string or an error
func ParseAPIVersion(version string) (APIVersion, error) {
	switch version {
	case string(APIv2):
		return APIv2, nil
	case string(APIv3):
		return APIv3, nil
	default:
		return "", fmt.Errorf("String '%s' is no a valid APIVersion", version)
	}
}

// Type is an enum of the supported envoy resource types
type Type string

const (
	// Endpoint is an envoy endpoint resource
	Endpoint Type = "Endpoint"
	// Cluster is an envoy cluster resource
	Cluster Type = "Cluster"
	// Route is an envoy route resource
	Route Type = "Route"
	// Listener is an envoy listener resource
	Listener Type = "Listener"
	// Secret is an envoy secret resource
	Secret Type = "Secret"
	// Runtime is an envoy runtime resource
	Runtime Type = "Runtime"
)
