package v1alpha1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DiscoveryServiceCertificateKind is a string that holds the Kind of DiscoveryServiceCertificate
	DiscoveryServiceCertificateKind string = "DiscoveryServiceCertificate"
	// CertificateNeedsRenewalCondition is a condition that indicates that a
	// DiscoveryServiceCertificate is invalid and needs replacement
	CertificateNeedsRenewalCondition status.ConditionType = "CertificateNeedsRenewal"
)

// DiscoveryServiceCertificateSpec defines the desired state of DiscoveryServiceCertificate
type DiscoveryServiceCertificateSpec struct {
	// CommonName is the CommonName of the certificate
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CommonName string `json:"commonName"`
	// IsServerCertificate is a boolean specifying if the certificate should be
	// issued with server auth usage enabled
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	IsServerCertificate bool `json:"server,omitempty"`
	// IsCA is a boolean specifying that the certificate is a CA
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	IsCA bool `json:"isCA,omitempty"`
	// ValidFor specifies the validity of the certificate in seconds
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ValidFor int64 `json:"validFor"`
	// Hosts is the list of hosts the certificate is valid for. Only
	// use when 'IsServerCertificate' is true. If unset, the CommonName
	// field will be used to populate the valid hosts of the certificate.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Hosts []string `json:"hosts,omitempty"`
	// Signer specifies  the signer to use to create this certificate. Supported
	// signers are CertManager and SelfSigned.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Signer DiscoveryServiceCertificateSigner `json:"signer"`
	// SecretRef is a reference to the secret that will hold the certificate
	// and the private key.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	SecretRef corev1.SecretReference `json:"secretRef"`
	// CertificateRenewalConfig configures the certificate renewal process. If unset default
	// behavior is to renew the certificate but not notify of renewals.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CertificateRenewalConfig *CertificateRenewalConfig `json:"certificateRenewalNotification,omitempty"`
}

// DiscoveryServiceCertificateSigner specifies the signer to use to provision the certificate
type DiscoveryServiceCertificateSigner struct {
	// CertManager holds specific configuration for the CertManager signer
	// +kubebuilder:validation:Optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CertManager *CertManagerConfig `json:"certManager,omitempty"`
	// SelfSigned holds specific configuration for the SelfSigned signer
	// +kubebuilder:validation:Optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	SelfSigned *SelfSignedConfig `json:"selfSigned,omitempty"`
	// CASigned holds specific configuration for the CASigned signer
	// +kubebuilder:validation:Optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CASigned *CASignedConfig `json:"caSigned,omitempty"`
}

// CertificateRenewalConfig configures the certificate renewal process.
type CertificateRenewalConfig struct {
	// Enabled is a flag to enable or disable renewal of the certificate
	Enabled bool `json:"enabled"`
	// Notify field holds a reference to another object which will be notified
	// of a certificate renewal through a condition. The other object's controller
	// is in charge of performing the necessary tasks once it has been notified of
	// the renewal.
	Notify *corev1.ObjectReference `json:"notify,omitempty"`
}

// CertManagerConfig is used to generate certificates using the given cert-manager issuer
type CertManagerConfig struct {
	// The name of the cert-manager ClusterIssuer to be used to sign the
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ClusterIssuer string `json:"clusterIssuer"`
}

// SelfSignedConfig is an empty struct to refer to the selfsiged certificates provisioner
type SelfSignedConfig struct{}

// CASignedConfig is used ti generate certificates signed by a CA contained in a Secret
type CASignedConfig struct {
	// A reference to a Secret containing the CA
	SecretRef corev1.SecretReference `json:"caSecretRef"`
}

// DiscoveryServiceCertificateStatus defines the observed state of DiscoveryServiceCertificate
type DiscoveryServiceCertificateStatus struct {
	// Conditions represent the latest available observations of an object's state
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Conditions status.Conditions `json:"conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DiscoveryServiceCertificate is used to create certificates, either self-signed
// or by using a cert-manager CA issuer. This object is used by the DiscoveryService
// controller to create the required certificates for the diferent components of the
// discovery service. Direct use of DiscoveryServiceCertificate objects is discouraged.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=discoveryservicecertificates,scope=Namespaced
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="DiscoveryServiceCertificate"
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Certificate,v1alpha2`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Secret,v1`
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
