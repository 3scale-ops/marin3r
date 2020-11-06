package envoy

import "github.com/golang/protobuf/proto"

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
