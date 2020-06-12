package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DiscoveryServiceSpec defines the desired state of DiscoveryService
type DiscoveryServiceSpec struct {
	Namespaces           []string               `json:"namespaces,omitempty"`
	ServerCertificateRef corev1.SecretReference `json:"serverCertificateRef,omitempty"`
}

// SidecarInjectorConfig contains config options for the configuration
// of the envoy sidecars
type SidecarInjectorConfig struct {
	ClientCertificateSecretPrefix string `json:"clientCertificateSecretPrefix,omitempty"`
	BootstrapConfigMapPrefix      string `json:"bootstrapConfigMapPrefix,omitempty"`
	WebhookConfigName             string `json:"webhookConfigName,omitempty"`
}

// DiscoveryServiceStatus defines the observed state of DiscoveryService
type DiscoveryServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DiscoveryService is the Schema for the DiscoveryServices API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=discoveryservices,scope=Namespaced
type DiscoveryService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DiscoveryServiceSpec   `json:"spec,omitempty"`
	Status DiscoveryServiceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DiscoveryServiceList contains a list of DiscoveryService
type DiscoveryServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DiscoveryService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DiscoveryService{}, &DiscoveryServiceList{})
}
