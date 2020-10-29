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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	/* Conditions */

	// CacheOutOfSyncCondition is a condition that indicates that the
	// envoyconfig cannot reach the desired status specified in the spec,
	// usually because the config in the spec is incorrect or has caused failures
	// in the envoy clients
	CacheOutOfSyncCondition status.ConditionType = "CacheOutOfSync"

	// RollbackFailedCondition indicates that the EnvoyConfig object
	// is not able to publish a config revision because all revisions are
	// tainted
	RollbackFailedCondition status.ConditionType = "RollbackFailed"

	/* Finalizers */

	// EnvoyConfigFinalizer is the finalizer for EnvoyConfig objects
	EnvoyConfigFinalizer string = "finalizer.marin3r.3scale.net"

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
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	NodeID string `json:"nodeID"`
	// Serialization specicifies the serialization format used to describe the resources. "json" and "yaml"
	// are supported. "json" is used if unset.
	// +kubebuilder:validation:Enum=json;b64json;yaml
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Serialization string `json:"serialization,omitempty"`
	// EnvoyResources holds the different types of resources suported by the envoy discovery service
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	EnvoyResources *EnvoyResources `json:"envoyResources"`
}

// EnvoyResources holds each envoy api resource type
type EnvoyResources struct {
	// Endpoints is a list of the envoy ClusterLoadAssignment resource type.
	// Reference: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/endpoint.proto
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Endpoints []EnvoyResource `json:"endpoints,omitempty"`
	// Clusters is a list of the envoy Cluster resource type.
	// Reference: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/cluster.proto
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Clusters []EnvoyResource `json:"clusters,omitempty"`
	// Routes is a list of the envoy Route resource type.
	// Reference: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/route.proto
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Routes []EnvoyResource `json:"routes,omitempty"`
	// Listeners is a list of the envoy Listener resource type.
	// Referece: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/listener.proto
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Listeners []EnvoyResource `json:"listeners,omitempty"`
	// Runtimes is a list of the envoy Runtime resource type.
	// Reference: https://www.envoyproxy.io/docs/envoy/latest/api-v2/service/discovery/v2/rtds.proto
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Runtimes []EnvoyResource `json:"runtime,omitempty"`
	// Secrets is a list of references to Kubernetes Secret objects.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Secrets []EnvoySecretResource `json:"secrets,omitempty"`
}

// EnvoyResource holds serialized representation of an envoy
// resource
type EnvoyResource struct {
	// Name of the envoy resource
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Name string `json:"name"`
	// Value is the serialized representation of the envoy resource
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Value string `json:"value"`
}

// EnvoySecretResource holds a reference to a k8s
// Secret from where to take a secret from
type EnvoySecretResource struct {
	// Name of the envoy resource
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Name string `json:"name"`
	// Ref is a reference to a Kubernetes Secret of type "kubernetes.io/tls" from which
	// an envoy Secret resource will be automatically created.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:io.kubernetes:SecretReference"
	Ref corev1.SecretReference `json:"ref"`
}

// EnvoyConfigStatus defines the observed state of EnvoyConfig
type EnvoyConfigStatus struct {
	// CacheState summarizes all the observations about the EnvoyConfig
	// to give the user a concrete idea on the general status of the discovery servie cache.
	// It is intended only for human consumption. Other controllers should relly on conditions
	// to determine the status of the discovery server cache.
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	CacheState string `json:"cacheState,omitempty"`
	// PublishedVersion is the config version currently
	// served by the envoy discovery service for the give nodeID
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	PublishedVersion string `json:"publishedVersion,omitempty"`
	// DesiredVersion represents the resources version described in
	// the spec of the EnvoyConfig object
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	DesiredVersion string `json:"desiredVersion,omitempty"`
	// Conditions represent the latest available observations of an object's state
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Conditions status.Conditions `json:"conditions,omitempty"`
	// ConfigRevisions is an ordered list of references to EnvoyConfigRevision
	// objects
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	ConfigRevisions []ConfigRevisionRef `json:"revisions,omitempty"`
}

// ConfigRevisionRef holds a reference to EnvoyConfigRevision object
type ConfigRevisionRef struct {
	// Version is a hash of the EnvoyResources field
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Version string `json:"version"`
	// Ref is a reference to the EnvoyConfigRevision object that
	// holds the configuration matching the Version field.
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Ref corev1.ObjectReference `json:"ref"`
}

// +kubebuilder:object:root=true

// EnvoyConfig holds the configuration for a given envoy nodeID. The spec of an EnvoyConfig
// object holds the envoy resources that conform the desired configuration for the given nodeID
// and that the discovery service will send to any envoy client that identifies itself with that
// nodeID.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=envoyconfigs,scope=Namespaced,shortName=ec
// +kubebuilder:printcolumn:JSONPath=".spec.nodeID",name=NodeID,type=string
// +kubebuilder:printcolumn:JSONPath=".status.desiredVersion",name=Desired Version,type=string
// +kubebuilder:printcolumn:JSONPath=".status.publishedVersion",name=Published Version,type=string
// +kubebuilder:printcolumn:JSONPath=".status.cacheState",name=Cache State,type=string
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="EnvoyConfig"
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`EnvoyConfigRevision,v1alpha1`
type EnvoyConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyConfigSpec   `json:"spec,omitempty"`
	Status EnvoyConfigStatus `json:"status,omitempty"`
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
