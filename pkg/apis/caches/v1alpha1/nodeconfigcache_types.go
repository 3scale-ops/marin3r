package v1alpha1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeConfigCacheSpec defines the desired state of NodeConfigCache
type NodeConfigCacheSpec struct {
	NodeID          string                   `json:"nodeID"`
	Version         string                   `json:"version"`
	Resources       EnvoyResources           `json:"resources"`
	ConfigRevisions []corev1.ObjectReference `json:"revisions,omitempty"`
}

// EnvoyResources holds each envoy api resource type
type EnvoyResources struct {
	Endpoints []EnvoyResource          `json:"endpoints,omitempty"`
	Clusters  []EnvoyResource          `json:"clusters,omitempty"`
	Routes    []EnvoyResource          `json:"routes,omitempty"`
	Listeners []EnvoyResource          `json:"listeners,omitempty"`
	Runtimes  []EnvoyResource          `json:"runtime,omitempty"`
	Secrets   []corev1.SecretReference `json:"secrets,omitempty"`
}

// EnvoyResource holds a single envoy api resources,
// serialized as json, base64 encoded
type EnvoyResource struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// NodeConfigCacheStatus defines the observed state of NodeConfigCache
type NodeConfigCacheStatus struct {
	// PublishedVersion is the config version currently
	// served by the envoy control plane for the node-id
	PublishedVersion string `json:"publishedVersion"`
	// Conditions represent the latest available observations of an object's state
	Conditions status.Conditions `json:"conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeConfigCache is the Schema for the nodeconfigcaches API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodeconfigcaches,scope=Namespaced
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
