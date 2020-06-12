package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServiceDiscoveryCertificateSpec defines the desired state of ServiceDiscoveryCertificate
type ServiceDiscoveryCertificateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// ServiceDiscoveryCertificateStatus defines the observed state of ServiceDiscoveryCertificate
type ServiceDiscoveryCertificateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceDiscoveryCertificate is the Schema for the servicediscoverycertificates API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=servicediscoverycertificates,scope=Namespaced
type ServiceDiscoveryCertificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceDiscoveryCertificateSpec   `json:"spec,omitempty"`
	Status ServiceDiscoveryCertificateStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceDiscoveryCertificateList contains a list of ServiceDiscoveryCertificate
type ServiceDiscoveryCertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceDiscoveryCertificate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceDiscoveryCertificate{}, &ServiceDiscoveryCertificateList{})
}
