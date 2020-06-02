package v1alpha1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeConfigCacheSpec defines the desired state of NodeConfigCache
type NodeConfigCacheSpec struct {
	// +kubebuilder:validation:Pattern:[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
	NodeID string `json:"nodeID"`
	// TODO: add validations
	Version string `json:"version"`
	// +kubebuilder:validation:Enum=json;b64json;yaml
	Serialization string          `json:"serialization,omitifempty"`
	Resources     *EnvoyResources `json:"resources"`
}

// EnvoyResources holds each envoy api resource type
type EnvoyResources struct {
	Endpoints []EnvoyResource       `json:"endpoints,omitempty"`
	Clusters  []EnvoyResource       `json:"clusters,omitempty"`
	Routes    []EnvoyResource       `json:"routes,omitempty"`
	Listeners []EnvoyResource       `json:"listeners,omitempty"`
	Runtimes  []EnvoyResource       `json:"runtime,omitempty"`
	Secrets   []EnvoySecretResource `json:"secrets,omitempty"`
}

// EnvoyResource holds a single envoy api resources,
// serialized as json, base64 encoded
type EnvoyResource struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// EnvoySecretResource holds a reference to a k8s
// Secret from where to take a secret from
type EnvoySecretResource struct {
	Name string                 `json:"name"`
	Ref  corev1.SecretReference `json:"ref"`
}

// NodeConfigCacheStatus defines the observed state of NodeConfigCache
type NodeConfigCacheStatus struct {
	// PublishedVersion is the config version currently
	// served by the envoy control plane for the node-id
	PublishedVersion string `json:"publishedVersion,omitempty"`
	// Status
	// Conditions represent the latest available observations of an object's state
	Conditions      status.Conditions   `json:"conditions,omitempty"`
	ConfigRevisions []ConfigRevisionRef `json:"revisions,omitempty"`
}

// ConfigRevisionRef holds a reference to NodeConfigRevision object
type ConfigRevisionRef struct {
	Version string                 `json:"version"`
	Ref     corev1.ObjectReference `json:"ref"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeConfigCache is the Schema for the nodeconfigcaches API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodeconfigcaches,scope=Namespaced,shortName=ncc
// +kubebuilder:printcolumn:JSONPath=".spec.nodeID",name=NodeID,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.version",name=Desired Version,type=string
// +kubebuilder:printcolumn:JSONPath=".status.publishedVersion",name=Published Version,type=string
// kubebuilder:printcolumn:JSONPath=".status.status",name=Status,type=string
type NodeConfigCache struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeConfigCacheSpec   `json:"spec,omitempty"`
	Status NodeConfigCacheStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeConfigCacheList contains a list of NodeConfigCache
type NodeConfigCacheList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeConfigCache `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeConfigCache{}, &NodeConfigCacheList{})
}
