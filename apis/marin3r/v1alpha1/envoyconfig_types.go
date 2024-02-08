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
	reconcilerutil "github.com/3scale-ops/basereconciler/util"
	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	/* Conditions */

	// CacheOutOfSyncCondition is a condition that indicates that the
	// envoyconfig cannot reach the desired status specified in the spec,
	// usually because the config in the spec is incorrect or has caused failures
	// in the envoy clients
	CacheOutOfSyncCondition string = "CacheOutOfSync"

	// RollbackFailedCondition indicates that the EnvoyConfig object
	// is not able to publish a config revision because all revisions are
	// tainted
	RollbackFailedCondition string = "RollbackFailed"

	/* State */

	//InSyncState indicates that a EnvoyConfig object has its resources spec
	// in sync with the xds server cache
	InSyncState string = "InSync"

	// RollbackState indicates that a EnvoyConfig object has performed a
	// rollback to a previous version of the resources spec
	RollbackState string = "Rollback"

	// RollbackFailedState indicates that there is no untainted revision that
	// can be pusblished in the xds server cache
	RollbackFailedState string = "RollbackFailed"
)

// EnvoyConfigSpec defines the desired state of EnvoyConfig
type EnvoyConfigSpec struct {
	// NodeID holds the envoy identifier for the discovery service to know which set
	// of resources to send to each of the envoy clients that connect to it.
	// +kubebuilder:validation:Pattern:[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	NodeID string `json:"nodeID"`
	// Serialization specicifies the serialization format used to describe the resources. "json" and "yaml"
	// are supported. "json" is used if unset.
	// +kubebuilder:validation:Enum=json;yaml
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Serialization *envoy_serializer.Serialization `json:"serialization,omitempty"`
	// EnvoyAPI is the version of envoy's API to use. Defaults to v3.
	// +kubebuilder:validation:Enum=v3
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	EnvoyAPI *envoy.APIVersion `json:"envoyAPI,omitempty"`
	// EnvoyResources holds the different types of resources suported by the envoy discovery service
	// DEPRECATED. Use the `resources` field instead.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	EnvoyResources *EnvoyResources `json:"envoyResources,omitempty"`
	// Resources holds the different types of resources suported by the envoy discovery service
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Resources []Resource `json:"resources,omitempty"`
}

// EnvoyConfigStatus defines the observed state of EnvoyConfig
type EnvoyConfigStatus struct {
	// CacheState summarizes all the observations about the EnvoyConfig
	// to give the user a concrete idea on the general status of the discovery servie cache.
	// It is intended only for human consumption. Other controllers should relly on conditions
	// to determine the status of the discovery server cache.
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	CacheState *string `json:"cacheState,omitempty"`
	// PublishedVersion is the config version currently
	// served by the envoy discovery service for the give nodeID
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	PublishedVersion *string `json:"publishedVersion,omitempty"`
	// DesiredVersion represents the resources version described in
	// the spec of the EnvoyConfig object
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	DesiredVersion *string `json:"desiredVersion,omitempty"`
	// Conditions represent the latest available observations of an object's state
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// ConfigRevisions is an ordered list of references to EnvoyConfigRevision
	// objects
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	ConfigRevisions []ConfigRevisionRef `json:"revisions,omitempty"`
}

// ConfigRevisionRef holds a reference to EnvoyConfigRevision object
type ConfigRevisionRef struct {
	// Version is a hash of the EnvoyResources field
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Version string `json:"version"`
	// Ref is a reference to the EnvoyConfigRevision object that
	// holds the configuration matching the Version field.
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Ref corev1.ObjectReference `json:"ref"`
}

// +kubebuilder:object:root=true

// EnvoyConfig holds the configuration for a given envoy nodeID. The spec of an EnvoyConfig
// object holds the Envoy resources that conform the desired configuration for the given nodeID
// and that the discovery service will send to any envoy client that identifies itself with that
// nodeID.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=envoyconfigs,scope=Namespaced,shortName=ec
// +kubebuilder:printcolumn:JSONPath=".spec.nodeID",name=Node ID,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.envoyAPI",name=Envoy API,type=string
// +kubebuilder:printcolumn:JSONPath=".status.desiredVersion",name=Desired Version,type=string
// +kubebuilder:printcolumn:JSONPath=".status.publishedVersion",name=Published Version,type=string
// +kubebuilder:printcolumn:JSONPath=".status.cacheState",name=Cache State,type=string
// +operator-sdk:csv:customresourcedefinitions:displayName="EnvoyConfig"
// +operator-sdk:csv:customresourcedefinitions:resources={{EnvoyConfigRevision,v1alpha1}}
type EnvoyConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyConfigSpec   `json:"spec,omitempty"`
	Status EnvoyConfigStatus `json:"status,omitempty"`
}

// GetEnvoyAPIVersion returns envoy's API version for the EnvoyConfigRevision
func (ec *EnvoyConfig) GetEnvoyAPIVersion() envoy.APIVersion {
	if ec.Spec.EnvoyAPI == nil {
		return envoy.APIv3
	}
	return envoy.APIVersion(*ec.Spec.EnvoyAPI)
}

// GetSerialization returns the encoding of the envoy resources.
func (ec *EnvoyConfig) GetSerialization() envoy_serializer.Serialization {
	if ec.Spec.Serialization == nil {
		return envoy_serializer.JSON
	}
	return envoy_serializer.Serialization(*ec.Spec.Serialization)
}

// GetEnvoyResourcesVersion returns the hash of the resources in the spec which
// univoquely identifies the version of the resources.
func (ec *EnvoyConfig) GetEnvoyResourcesVersion() string {
	return reconcilerutil.Hash(ec.Spec.Resources)
}

// Default implements defaulting for the EnvoyConfig resource
func (ec *EnvoyConfig) Default() {
	if ec.Spec.EnvoyAPI == nil {
		ec.Spec.EnvoyAPI = pointer.New(ec.GetEnvoyAPIVersion())
	}
}

// +kubebuilder:object:root=true

// EnvoyConfigList contains a list of EnvoyConfig
type EnvoyConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnvoyConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnvoyConfig{}, &EnvoyConfigList{})
}
