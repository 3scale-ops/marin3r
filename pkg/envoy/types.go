package envoy

import (
	"fmt"
	"net"

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

// Type is an enum of the supported envoy resource types
type Type string

const (
	// Endpoint is an envoy endpoint resource
	Endpoint Type = "endpoint"
	// Cluster is an envoy cluster resource
	Cluster Type = "cluster"
	// Route is an envoy route resource
	Route Type = "route"
	// ScopedRoute is an envoy scoped route resource
	ScopedRoute Type = "scopedRoute"
	// VirtualHost is an enovy virtual host resource (not implemented)
	VirtualHost Type = "virtualHost"
	// Listener is an envoy listener resource
	Listener Type = "listener"
	// Secret is an envoy secret resource
	Secret Type = "secret"
	// Runtime is an envoy runtime resource
	Runtime Type = "runtime"
	// ExtensionConfig is an envoy extension config resource
	ExtensionConfig Type = "extensionConfig"
)

type EndpointHealthStatus int32

const (
	// The health status is not known. This is interpreted by Envoy as ``HEALTHY``.
	HealthStatus_UNKNOWN EndpointHealthStatus = 0
	// Healthy.
	HealthStatus_HEALTHY EndpointHealthStatus = 1
	// Unhealthy.
	HealthStatus_UNHEALTHY EndpointHealthStatus = 2
	// Connection draining in progress. E.g.,
	// `<https://aws.amazon.com/blogs/aws/elb-connection-draining-remove-instances-from-service-with-care/>`_
	// or
	// `<https://cloud.google.com/compute/docs/load-balancing/enabling-connection-draining>`_.
	// This is interpreted by Envoy as ``UNHEALTHY``.
	HealthStatus_DRAINING EndpointHealthStatus = 3
	// Health check timed out. This is part of HDS and is interpreted by Envoy as
	// ``UNHEALTHY``.
	HealthStatus_TIMEOUT EndpointHealthStatus = 4
	// Degraded.
	HealthStatus_DEGRADED EndpointHealthStatus = 5
)

type UpstreamHost struct {
	IP     net.IP
	Port   uint32
	Health EndpointHealthStatus
}
