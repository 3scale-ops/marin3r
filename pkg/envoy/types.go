package envoy

import (
	"fmt"

	"google.golang.org/protobuf/proto"
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
	// APIv3 is the envoy v3 API version.
	APIv3 APIVersion = "v3"
)

// String returns the string representation of APIVersion
func (version APIVersion) String() string {
	return string(APIv3)
}

// ParseAPIVersion returns an APIVersion for the given string or an error
func ParseAPIVersion(version string) (APIVersion, error) {
	switch version {
	case string(APIv3):
		return APIv3, nil
	default:
		return "", fmt.Errorf("String '%s' is no a valid APIVersion", version)
	}
}

// Type is an enum of the supported envoy resource typeswe can just use a strings.Split to create the array of args from the custom resource field, with that each of the "words" will be passed correctly without the quotes around c
type Type string

const (
	// Endpoint is an envoy endpoint resource
	Endpoint Type = "Endpoint"
	// Cluster is an envoy cluster resource
	Cluster Type = "Cluster"
	// Route is an envoy route resource
	Route Type = "Route"
	// ScopedRoute is an envoy scoped route resource
	ScopedRoute Type = "ScopedRoute"
	// Listener is an envoy listener resource
	Listener Type = "Listener"
	// Secret is an envoy secret resource
	Secret Type = "Secret"
	// Runtime is an envoy runtime resource
	Runtime Type = "Runtime"
	// ExtensionConfig is an envoy extension config resource
	ExtensionConfig Type = "ExtensionConfig"
)
