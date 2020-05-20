package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeConfigRevisionSpec defines the desired state of NodeConfigRevision
type NodeConfigRevisionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	NodeID    string `json:"nodeID"`
	Version   int64  `json:"version"`
	Clusters  string `json:"clusters"`
	Listeners string `json:"listeners"`
	Secrets   string `json:"secrets"`
}

// NodeConfigRevisionStatus defines the observed state of NodeConfigRevision
type NodeConfigRevisionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	HasFailed      bool   `json:"hasFailed"`
	FailureMessage string `json:"failureMessage"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeConfigRevision is the Schema for the nodeconfigrevisions API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodeconfigrevisions,scope=Namespaced
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
