package filters

import (
	"github.com/3scale/marin3r/pkg/envoy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NodeIDTag   = "marin3r.3scale.net/node-id"
	VersionTag  = "marin3r.3scale.net/config-version"
	EnvoyAPITag = "marin3r.3scale.net/envoy-api"
)

// RevisionFilter is an interface that revision label selectors
// implement
type RevisionFilter interface {
	ApplyToLabelSelector(client.MatchingLabels)
}

// VersionFilter is a struct used to filter revisions
// by their version
type VersionFilter struct {
	Value string
}

// ApplyToLabelSelector applies the VersionFilter to the given
// MatchingLabels selector
func (vf *VersionFilter) ApplyToLabelSelector(selector client.MatchingLabels) {
	selector[VersionTag] = vf.Value
}

// ByVersion returns a VersionFilter
func ByVersion(version string) RevisionFilter {
	return &VersionFilter{Value: version}
}

// NodeIDFilter is a struct used to filter revisions
// by theis version
type NodeIDFilter struct {
	Value string
}

// ApplyToLabelSelector applies the VersionFilter to the given
// MatchingLabels selector
func (nf *NodeIDFilter) ApplyToLabelSelector(selector client.MatchingLabels) {
	selector[NodeIDTag] = nf.Value
}

// ByNodeID returns a NodeIDFilter
func ByNodeID(nodeID string) RevisionFilter {
	return &NodeIDFilter{Value: nodeID}
}

// EnvoyAPIFilter is a struct used to filter revisions
// by theis version
type EnvoyAPIFilter struct {
	Value string
}

// ApplyToLabelSelector applies the VersionFilter to the given
// MatchingLabels selector
func (ef *EnvoyAPIFilter) ApplyToLabelSelector(selector client.MatchingLabels) {
	selector[EnvoyAPITag] = ef.Value
}

// ByEnvoyAPI returns a NodeIDFilter
func ByEnvoyAPI(envoyAPI envoy.APIVersion) RevisionFilter {
	return &EnvoyAPIFilter{Value: envoyAPI.String()}
}
