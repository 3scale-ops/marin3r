package envoy

type Type string

const (
	Endpoint Type = "Endpoint"
	Cluster  Type = "Cluster"
	Route    Type = "Route"
	Listener Type = "Listener"
	Secret   Type = "Secret"
	Runtime  Type = "Runtime"
)
