package v1alpha1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// RevisionPublishedCondition is a condition that marks the EnvoyConfigRevision object
	// as the one that should be published in the xds server cache
	RevisionPublishedCondition status.ConditionType = "RevisionPublished"

	// ResourcesOutOfSyncCondition is a condition that other controllers can use to indicate
	// that the respurces need resync
	ResourcesOutOfSyncCondition status.ConditionType = "ResourcesOutOfSync"

	// RevisionTaintedCondition is a condition type that's used to report that this
	// problems have been observed with this revision and should not be published
	RevisionTaintedCondition status.ConditionType = "RevisionTainted"
)

// EnvoyConfigRevisionSpec defines the desired state of EnvoyConfigRevision
type EnvoyConfigRevisionSpec struct {
	// +kubebuilder:validation:Pattern:[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
	NodeID  string `json:"nodeID"`
	Version string `json:"version"`
	// +kubebuilder:validation:Enum=json;b64json;yaml
	Serialization string          `json:"serialization,omitempty"`
	Resources     *EnvoyResources `json:"revision"`
}

// EnvoyConfigRevisionStatus defines the observed state of EnvoyConfigRevision
type EnvoyConfigRevisionStatus struct {
	// Published signals if the EnvoyConfigRevision is the one currently published
	// in the xds server cache
	Published bool `json:"published,omitempty"`
	// LastPublishedAt indicates the last time this config review transitioned to
	// published
	LastPublishedAt metav1.Time `json:"lastPublishedAt,omitempty"`
	// Tainted indicates whether the EnvoyConfigRevision is eligible for publishing
	// or not
	Tainted bool `json:"tainted,omitempty"`
	// Conditions represent the latest available observations of an object's state
	Conditions status.Conditions `json:"conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EnvoyConfigRevision is the Schema for the envoyconfigrevisions API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=envoyconfigrevisions,scope=Namespaced,shortName=ecr
// +kubebuilder:printcolumn:JSONPath=".spec.nodeID",name=NodeID,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.version",name=Version,type=string
// +kubebuilder:printcolumn:JSONPath=".status.published",name=Published,type=boolean
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Created At",type=string,format=date-time
// +kubebuilder:printcolumn:JSONPath=".status.lastPublishedAt",name="Last Published At",type=string,format=date-time
// +kubebuilder:printcolumn:JSONPath=".status.tainted",name=Tainted,type=boolean
type EnvoyConfigRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyConfigRevisionSpec   `json:"spec,omitempty"`
	Status EnvoyConfigRevisionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EnvoyConfigRevisionList contains a list of EnvoyConfigRevision
type EnvoyConfigRevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnvoyConfigRevision `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnvoyConfigRevision{}, &EnvoyConfigRevisionList{})
}
