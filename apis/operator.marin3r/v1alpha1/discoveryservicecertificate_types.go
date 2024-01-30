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
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DiscoveryServiceCertificateKind is a string that holds the Kind of DiscoveryServiceCertificate
	DiscoveryServiceCertificateKind string = "DiscoveryServiceCertificate"
	// CertificateNeedsRenewalCondition is a condition that indicates that a
	// DiscoveryServiceCertificate is invalid and needs replacement
	CertificateNeedsRenewalCondition string = "CertificateNeedsRenewal"
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
	// +optional
	IsServerCertificate *bool `json:"server,omitempty"`
	// IsCA is a boolean specifying that the certificate is a CA
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	IsCA *bool `json:"isCA,omitempty"`
	// ValidFor specifies the validity of the certificate in seconds
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ValidFor int64 `json:"validFor"`
	// Hosts is the list of hosts the certificate is valid for. Only
	// use when 'IsServerCertificate' is true. If unset, the CommonName
	// field will be used to populate the valid hosts of the certificate.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
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
	// +optional
	CertificateRenewalConfig *CertificateRenewalConfig `json:"certificateRenewal,omitempty"`
}

// IsServerCertificate returns true if the certificate is issued for server
// usage or false if not
func (d *DiscoveryServiceCertificate) IsServerCertificate() bool {
	if d.Spec.IsServerCertificate == nil {
		return false
	}
	return *d.Spec.IsServerCertificate
}

// IsCA returns true if the certificate is issued to function
// as a certificate authority or not
func (d *DiscoveryServiceCertificate) IsCA() bool {
	if d.Spec.IsCA == nil {
		return false
	}
	return *d.Spec.IsCA
}

// GetHosts returns the list of server names that the certificate
// is issued for
func (d *DiscoveryServiceCertificate) GetHosts() []string {
	if d.Spec.Hosts == nil {
		return []string{d.Spec.CommonName}
	}
	return d.Spec.Hosts
}

// GetCertificateRenewalConfig returns the renewal configuration for the issued certificate
func (d *DiscoveryServiceCertificate) GetCertificateRenewalConfig() CertificateRenewalConfig {
	if d.Spec.CertificateRenewalConfig == nil {
		return d.defaultCertificateRenewalConfig()
	}
	return *d.Spec.CertificateRenewalConfig
}

func (d *DiscoveryServiceCertificate) defaultCertificateRenewalConfig() CertificateRenewalConfig {
	return CertificateRenewalConfig{Enabled: true}
}

// DiscoveryServiceCertificateSigner specifies the signer to use to provision the certificate
type DiscoveryServiceCertificateSigner struct {
	// SelfSigned holds specific configuration for the SelfSigned signer
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	SelfSigned *SelfSignedConfig `json:"selfSigned,omitempty"`
	// CASigned holds specific configuration for the CASigned signer
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
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
	// Ready is a boolean that specifies if the certificate is ready to be used
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	Ready *bool `json:"ready,omitempty"`
	// NotBefore is the time at which the certificate starts
	// being valid
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	NotBefore *metav1.Time `json:"notBefore,omitempty"`
	// NotAfter is the time at which the certificate expires
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	NotAfter *metav1.Time `json:"notAfter,omitempty"`
	// CertificateHash stores the current hash of the certificate. It is used
	// for other controllers to validate if a certificate has been re-issued.
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	CertificateHash *string `json:"certificateHash,omitempty"`
	// Conditions represent the latest available observations of an object's state
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// IsReady returns true if the certificate is ready to use, false otherwise
func (status *DiscoveryServiceCertificateStatus) IsReady() bool {
	if status.Ready == nil {
		return false
	}
	return *status.Ready
}

// GetCertificateHash returns the hash of the certificate associated
// with the DiscoveryServiceCertificate resource. Returns an empty
// string if not set.
func (status *DiscoveryServiceCertificateStatus) GetCertificateHash() string {
	if status.CertificateHash == nil {
		return ""
	}
	return *status.CertificateHash
}

// +kubebuilder:object:root=true

// DiscoveryServiceCertificate is an internal resource used to create certificates. This resource
// is used by the DiscoveryService controller to create the required certificates for the different
// components. Direct use of DiscoveryServiceCertificate objects is discouraged.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=discoveryservicecertificates,scope=Namespaced
// +kubebuilder:printcolumn:JSONPath=".status.ready",name="Ready",type=boolean
// +kubebuilder:printcolumn:JSONPath=".status.notBefore",name=Not Before,type=string,format=date-time
// +kubebuilder:printcolumn:JSONPath=".status.notAfter",name=Not After,type=string,format=date-time
// +operator-sdk:csv:customresourcedefinitions:displayName="DiscoveryServiceCertificate"
// +operator-sdk:gen-csv:customresourcedefinitions:resources={{Secret,v1}}
type DiscoveryServiceCertificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DiscoveryServiceCertificateSpec   `json:"spec,omitempty"`
	Status DiscoveryServiceCertificateStatus `json:"status,omitempty"`
}

func (dsc *DiscoveryServiceCertificate) Default() {
	if dsc.Spec.IsServerCertificate == nil {
		dsc.Spec.IsServerCertificate = pointer.New(dsc.IsServerCertificate())
	}
	if dsc.Spec.IsCA == nil {
		dsc.Spec.IsCA = pointer.New(dsc.IsCA())
	}
	if dsc.Spec.Hosts == nil {
		dsc.Spec.Hosts = dsc.GetHosts()
	}
	if dsc.Spec.CertificateRenewalConfig == nil {
		crc := dsc.GetCertificateRenewalConfig()
		dsc.Spec.CertificateRenewalConfig = &crc
	}
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
