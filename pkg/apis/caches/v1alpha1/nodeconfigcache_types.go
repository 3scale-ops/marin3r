package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeConfigCacheSpec defines the desired state of NodeConfigCache
type NodeConfigCacheSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	NodeID         string `json:"nodeID"`
	CurrentVersion int64  `json:"currentVersion"`
}

// NodeConfigCacheStatus defines the observed state of NodeConfigCache
type NodeConfigCacheStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
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
