package reconcilers

import (
	"github.com/3scale/marin3r/pkg/envoy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nodeIDTag   = "marin3r.3scale.net/node-id"
	versionTag  = "marin3r.3scale.net/config-version"
	envoyAPITag = "marin3r.3scale.net/envoy-api"
)

// RevisionFilter is an interface that revision label selectors
// implement
type RevisionFilter interface {
	ApplyToLabelSelector(client.MatchingLabels)
}

// VersionFilter is a struct used to filter revisions
// by theis version
type VersionFilter struct {
	Value string
}

// ApplyToLabelSelector applies the VersionFilter to the given
// MatchingLabels selector
func (vf *VersionFilter) ApplyToLabelSelector(selector client.MatchingLabels) {
	selector[versionTag] = vf.Value
}

// FilterByVersion returns a VersionFilter
func FilterByVersion(version string) RevisionFilter {
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
	selector[nodeIDTag] = nf.Value
}

// FilterByNodeID returns a NodeIDFilter
func FilterByNodeID(nodeID string) RevisionFilter {
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
	selector[envoyAPITag] = ef.Value
}

// FilterByEnvoyAPI returns a NodeIDFilter
func FilterByEnvoyAPI(envoyAPI envoy.APIVersion) RevisionFilter {
	return &EnvoyAPIFilter{Value: envoyAPI.String()}
}
