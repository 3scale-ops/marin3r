package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SidecarConfigSpec defines the desired state of SidecarConfig
type SidecarConfigSpec struct {
	ServiceDiscoveryLabel string                      `json:"serviceDiscoveryLabel"`
	ClientCertificateRef  corev1.LocalObjectReference `json:"clientCertificateRef"`
	BootstrapConfigMapRef corev1.LocalObjectReference `json:"bootstrapConfigMap"`
}

// SidecarConfigStatus defines the observed state of SidecarConfig
type SidecarConfigStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SidecarConfig is the Schema for the sidecarconfigs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=sidecarconfigs,scope=Namespaced
type SidecarConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SidecarConfigSpec   `json:"spec,omitempty"`
	Status SidecarConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SidecarConfigList contains a list of SidecarConfig
type SidecarConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SidecarConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SidecarConfig{}, &SidecarConfigList{})
}
