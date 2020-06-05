package v1alpha1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// RevisionPublishedCondition is a condition that marks the NodeConfigRevision object
	// as the one that should be published in the xds server cache
	RevisionPublishedCondition status.ConditionType = "RevisionPublished"

	// ResourcesOutOfSyncCondition is a condition that other controllers can use to indicate
	// that the respurces need resync
	ResourcesOutOfSyncCondition status.ConditionType = "ResourcesOutOfSync"

	// ResourcesUpdateUnsuccessfulCondition is a condition type that's used to report
	// back to the controller that a resources' update has been unsuccesful
	// so the controller can act accordingly
	ResourcesUpdateUnsuccessfulCondition status.ConditionType = "ResourcesUpdateUnsuccessful"
)

// NodeConfigRevisionSpec defines the desired state of NodeConfigRevision
type NodeConfigRevisionSpec struct {
	// +kubebuilder:validation:Pattern:[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
	NodeID  string `json:"nodeID"`
	Version string `json:"version"`
	// +kubebuilder:validation:Enum=json;b64json;yaml
	Serialization string          `json:"serialization,omitifempty"`
	Resources     *EnvoyResources `json:"revision"`
}

// NodeConfigRevisionStatus defines the observed state of NodeConfigRevision
type NodeConfigRevisionStatus struct {
	// Conditions represent the latest available observations of an object's state
	Conditions status.Conditions `json:"conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeConfigRevision is the Schema for the nodeconfigrevisions API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodeconfigrevisions,scope=Namespaced,shortName=ncr
type NodeConfigRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeConfigRevisionSpec   `json:"spec,omitempty"`
	Status NodeConfigRevisionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeConfigRevisionList contains a list of NodeConfigRevision
type NodeConfigRevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeConfigRevision `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeConfigRevision{}, &NodeConfigRevisionList{})
}
