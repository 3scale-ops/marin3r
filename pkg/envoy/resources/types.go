package envoy

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
