package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DiscoveryServiceCertificateSpec defines the desired state of DiscoveryServiceCertificate
type DiscoveryServiceCertificateSpec struct {
	CommonName          string                            `json:"commonName"`
	IsServerCertificate bool                              `json:"server,omitempty"`
	IsCA                bool                              `json:"isCA,omitempty"`
	ValidFor            int64                             `json:"validFor"`
	Hosts               []string                          `json:"hosts,omitempty"`
	Signer              DiscoveryServiceCertificateSigner `json:"signer"`
	SecretRef           corev1.SecretReference            `json:"secretRef"`
}

// DiscoveryServiceCertificateSigner specifies the signer to use to provision the certificate
type DiscoveryServiceCertificateSigner struct {
	// +kubebuilder:validation:Optional
	CertManager *CertManagerConfig `json:"certManager,omitempty"`
	// +kubebuilder:validation:Optional
	SelfSigned *SelfSignedConfig `json:"selfSigned,omitempty"`
}

// CertManagerConfig is used to generate certificates using the given cert-manager issuer
type CertManagerConfig struct {
	ClusterIssuer string `json:"clusterIssuer"`
}

// SelfSignedConfig is an empty struct to refer to the selfsiged certificates provisioner
type SelfSignedConfig struct{}

// DiscoveryServiceCertificateStatus defines the observed state of DiscoveryServiceCertificate
type DiscoveryServiceCertificateStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DiscoveryServiceCertificate is the Schema for the DiscoveryServicecertificates API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=discoveryservicecertificates,scope=Namespaced
type DiscoveryServiceCertificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DiscoveryServiceCertificateSpec   `json:"spec,omitempty"`
	Status DiscoveryServiceCertificateStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DiscoveryServiceCertificateList contains a list of DiscoveryServiceCertificate
type DiscoveryServiceCertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DiscoveryServiceCertificate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DiscoveryServiceCertificate{}, &DiscoveryServiceCertificateList{})
}
