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
	// DiscoveryServiceCertificateKind is a string that holds the Kind of DiscoveryServiceCertificate
	DiscoveryServiceCertificateKind string = "DiscoveryServiceCertificate"
	// CertificateNeedsRenewalCondition is a condition that indicates that a
	// DiscoveryServiceCertificate is invalid and needs replacement
	CertificateNeedsRenewalCondition status.ConditionType = "CertificateNeedsRenewal"
	// CertificateHashLabelKey is the label that stores the hash of the certificate managed
	// by the DiscoveryServiceCertificate resource
	CertificateHashLabelKey string = "certificate-hash"
	// IssuerCertificateHashLabelKey is the label that stores the hash of the certificate managed
	// by the DiscoveryServiceCertificate resource
	IssuerCertificateHashLabelKey string = "issuer-certificate-hash"
)

// DiscoveryServiceCertificateSpec defines the desired state of DiscoveryServiceCertificate
type DiscoveryServiceCertificateSpec struct {
	// CommonName is the CommonName of the certificate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	CommonName string `json:"commonName"`
	// IsServerCertificate is a boolean specifying if the certificate should be
	// issued with server auth usage enabled
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	IsServerCertificate bool `json:"server,omitempty"`
	// IsCA is a boolean specifying that the certificate is a CA
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	IsCA bool `json:"isCA,omitempty"`
	// ValidFor specifies the validity of the certificate in seconds
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ValidFor int64 `json:"validFor"`
	// Hosts is the list of hosts the certificate is valid for. Only
	// use when 'IsServerCertificate' is true. If unset, the CommonName
	// field will be used to populate the valid hosts of the certificate.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Hosts []string `json:"hosts,omitempty"`
	// Signer specifies  the signer to use to create this certificate. Supported
	// signers are CertManager and SelfSigned.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Signer DiscoveryServiceCertificateSigner `json:"signer"`
	// SecretRef is a reference to the secret that will hold the certificate
	// and the private key.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SecretRef corev1.SecretReference `json:"secretRef"`
	// CertificateRenewalConfig configures the certificate renewal process. If unset default
	// behavior is to renew the certificate but not notify of renewals.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	CertificateRenewalConfig *CertificateRenewalConfig `json:"certificateRenewalNotification,omitempty"`
}

// DiscoveryServiceCertificateSigner specifies the signer to use to provision the certificate
type DiscoveryServiceCertificateSigner struct {
	// SelfSigned holds specific configuration for the SelfSigned signer
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SelfSigned *SelfSignedConfig `json:"selfSigned,omitempty"`
	// CASigned holds specific configuration for the CASigned signer
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	CASigned *CASignedConfig `json:"caSigned,omitempty"`
}

// CertificateRenewalConfig configures the certificate renewal process.
type CertificateRenewalConfig struct {
	// Enabled is a flag to enable or disable renewal of the certificate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Enabled bool `json:"enabled"`
}

// SelfSignedConfig is an empty struct to refer to the selfsiged certificates provisioner
type SelfSignedConfig struct{}

// CASignedConfig is used ti generate certificates signed by a CA contained in a Secret
type CASignedConfig struct {
	// A reference to a Secret containing the CA
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SecretRef corev1.SecretReference `json:"caSecretRef"`
}

// DiscoveryServiceCertificateStatus defines the observed state of DiscoveryServiceCertificate
type DiscoveryServiceCertificateStatus struct {
	// Conditions represent the latest available observations of an object's state
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions status.Conditions `json:"conditions"`
}

// +kubebuilder:object:root=true

// DiscoveryServiceCertificate is used to create certificates, either self-signed
// or by using a cert-manager CA issuer. This object is used by the DiscoveryService
// controller to create the required certificates for the different components of the
// discovery service. Direct use of DiscoveryServiceCertificate objects is discouraged.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=discoveryservicecertificates,scope=Namespaced
// +operator-sdk:csv:customresourcedefinitions:displayName="DiscoveryServiceCertificate"
// +operator-sdk:gen-csv:customresourcedefinitions:resources={{Secret,v1}}
type DiscoveryServiceCertificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DiscoveryServiceCertificateSpec   `json:"spec,omitempty"`
	Status DiscoveryServiceCertificateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DiscoveryServiceCertificateList contains a list of DiscoveryServiceCertificate
type DiscoveryServiceCertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DiscoveryServiceCertificate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DiscoveryServiceCertificate{}, &DiscoveryServiceCertificateList{})
}
