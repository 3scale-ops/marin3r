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
	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	/* Conditions */

	// RevisionPublishedCondition is a condition that marks the EnvoyConfigRevision object
	// as the one that should be published in the xds server cache
	RevisionPublishedCondition string = "RevisionPublished"

	// ResourcesInSyncCondition is a condition that other controllers can use to indicate
	// that the respurces need resync
	ResourcesInSyncCondition string = "ResourcesInSync"

	// RevisionTaintedCondition is a condition type that's used to report that this
	// problems have been observed with this revision and should not be published
	RevisionTaintedCondition string = "RevisionTainted"

	/* Finalizers */

	// EnvoyConfigRevisionFinalizer is the finalizer for EnvoyConfig objects
	EnvoyConfigRevisionFinalizer string = "finalizer.marin3r.3scale.net"
)

// EnvoyConfigRevisionSpec defines the desired state of EnvoyConfigRevision
type EnvoyConfigRevisionSpec struct {
	// NodeID holds the envoy identifier for the discovery service to know which set
	// of resources to send to each of the envoy clients that connect to it.
	// +kubebuilder:validation:Pattern:[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	NodeID string `json:"nodeID"`
	// Version is a hash of the EnvoyResources field
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Version string `json:"version"`
	// EnvoyAPI is the version of envoy's API to use. Defaults to v3.
	// +kubebuilder:validation:Enum=v3
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	EnvoyAPI *envoy.APIVersion `json:"envoyAPI,omitempty"`
	// Serialization specicifies the serialization format used to describe the resources. "json" and "yaml"
	// are supported. "json" is used if unset.
	// +kubebuilder:validation:Enum=json;b64json;yaml
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Serialization *envoy_serializer.Serialization `json:"serialization,omitempty"`
	// EnvoyResources holds the different types of resources suported by the envoy discovery service
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	EnvoyResources *EnvoyResources `json:"envoyResources,omitempty"`
	// Resources holds the different types of resources suported by the envoy discovery service
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Resources []Resource `json:"resources,omitempty"`
}

// EnvoyConfigRevisionStatus defines the observed state of EnvoyConfigRevision
type EnvoyConfigRevisionStatus struct {
	// Published signals if the EnvoyConfigRevision is the one currently published
	// in the xds server cache
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	Published *bool `json:"published,omitempty"`
	// ProvidesVersions keeps track of the version that this revision
	// publishes in the xDS server for each resource type
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	ProvidesVersions *VersionTracker `json:"providesVersions,omitempty"`
	// LastPublishedAt indicates the last time this config review transitioned to
	// published
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	LastPublishedAt *metav1.Time `json:"lastPublishedAt,omitempty"`
	// Tainted indicates whether the EnvoyConfigRevision is eligible for publishing
	// or not
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	Tainted *bool `json:"tainted,omitempty"`
	// Conditions represent the latest available observations of an object's state
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// IsPublished returns true if this revision is published, false otherwise
func (status *EnvoyConfigRevisionStatus) IsPublished() bool {
	if status.Published == nil {
		return false
	}
	return *status.Published
}

// IsTainted returns true if this revision is tainted, false otherwise
func (status *EnvoyConfigRevisionStatus) IsTainted() bool {
	if status.Tainted == nil {
		return false
	}
	return *status.Tainted
}

// VersionTracker tracks the versions of the resources
// that this revision publishes in the xDS server cache
type VersionTracker struct {
	Endpoints        string `json:"endpoints,omitempty"`
	Clusters         string `json:"clusters,omitempty"`
	Routes           string `json:"routes,omitempty"`
	ScopedRoutes     string `json:"scopedRoutes,omitempty"`
	Listeners        string `json:"listeners,omitempty"`
	Secrets          string `json:"secrets,omitempty"`
	Runtimes         string `json:"runtimes,omitempty"`
	ExtensionConfigs string `json:"extensionConfigs,omitempty"`
}

// +kubebuilder:object:root=true

// EnvoyConfigRevision is an internal resource that stores a specific version of an EnvoyConfig
// resource. EnvoyConfigRevisions are automatically created and deleted by the EnvoyConfig
// controller and are not intended to be directly used. Use EnvoyConfig objects instead.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=envoyconfigrevisions,scope=Namespaced,shortName=ecr
// +kubebuilder:printcolumn:JSONPath=".spec.nodeID",name=Node ID,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.envoyAPI",name=Envoy API,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.version",name=Version,type=string
// +kubebuilder:printcolumn:JSONPath=".status.published",name=Published,type=boolean
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Created At",type=string,format=date-time
// +kubebuilder:printcolumn:JSONPath=".status.lastPublishedAt",name="Last Published At",type=string,format=date-time
// +kubebuilder:printcolumn:JSONPath=".status.tainted",name=Tainted,type=boolean
// +operator-sdk:csv:customresourcedefinitions:displayName="EnvoyConfigRevision"
type EnvoyConfigRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyConfigRevisionSpec   `json:"spec,omitempty"`
	Status EnvoyConfigRevisionStatus `json:"status,omitempty"`
}

// GetEnvoyAPIVersion returns envoy's API version for the EnvoyConfigRevision
func (ecr *EnvoyConfigRevision) GetEnvoyAPIVersion() envoy.APIVersion {
	if ecr.Spec.EnvoyAPI == nil {
		return envoy.APIv3
	}
	return envoy.APIVersion(*ecr.Spec.EnvoyAPI)
}

// GetSerialization returns the encoding of the envoy resources.
func (ecr *EnvoyConfigRevision) GetSerialization() envoy_serializer.Serialization {
	if ecr.Spec.Serialization == nil {
		return envoy_serializer.JSON
	}
	return envoy_serializer.Serialization(*ecr.Spec.Serialization)
}

// Default implements defaulting for the EnvoyConfigRevision resource
func (ecr *EnvoyConfigRevision) Default() {
	if ecr.Spec.EnvoyAPI == nil {
		ecr.Spec.EnvoyAPI = pointer.New(ecr.GetEnvoyAPIVersion())
	}
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
