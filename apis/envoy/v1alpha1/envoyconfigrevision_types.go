/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/operator-framework/operator-lib/status"
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
	// NodeID holds the envoy identifier for the discovery service to know which set
	// of resources to send to each of the envoy clients that connect to it.
	// +kubebuilder:validation:Pattern:[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	NodeID string `json:"nodeID"`
	// Version is a hash of the EnvoyResources field
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Version string `json:"version"`
	// Serialization specicifies the serialization format used to describe the resources. "json" and "yaml"
	// are supported. "json" is used if unset.
	// +kubebuilder:validation:Enum=json;b64json;yaml
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Serialization string `json:"serialization,omitempty"`
	// EnvoyResources holds the different types of resources suported by the envoy discovery service
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	EnvoyResources *EnvoyResources `json:"envoyResources"`
}

// EnvoyConfigRevisionStatus defines the observed state of EnvoyConfigRevision
type EnvoyConfigRevisionStatus struct {
	// Published signals if the EnvoyConfigRevision is the one currently published
	// in the xds server cache
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Published bool `json:"published,omitempty"`
	// LastPublishedAt indicates the last time this config review transitioned to
	// published
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	LastPublishedAt metav1.Time `json:"lastPublishedAt,omitempty"`
	// Tainted indicates whether the EnvoyConfigRevision is eligible for publishing
	// or not
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Tainted bool `json:"tainted,omitempty"`
	// Conditions represent the latest available observations of an object's state
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Conditions status.Conditions `json:"conditions"`
}

// +kubebuilder:object:root=true

// EnvoyConfigRevision holds an specific version of the EnvoyConfig resources.
// EnvoyConfigRevisions are automatically created and deleted  by the EnvoyConfig
// controller and are not intended to be directly used. Use EnvoyConfig objects instead.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=envoyconfigrevisions,scope=Namespaced,shortName=ecr
// +kubebuilder:printcolumn:JSONPath=".spec.nodeID",name=NodeID,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.version",name=Version,type=string
// +kubebuilder:printcolumn:JSONPath=".status.published",name=Published,type=boolean
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Created At",type=string,format=date-time
// +kubebuilder:printcolumn:JSONPath=".status.lastPublishedAt",name="Last Published At",type=string,format=date-time
// +kubebuilder:printcolumn:JSONPath=".status.tainted",name=Tainted,type=boolean
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="EnvoyConfigRevision"
type EnvoyConfigRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyConfigRevisionSpec   `json:"spec,omitempty"`
	Status EnvoyConfigRevisionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnvoyConfigRevisionList contains a list of EnvoyConfigRevision
type EnvoyConfigRevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnvoyConfigRevision `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnvoyConfigRevision{}, &EnvoyConfigRevisionList{})
}
