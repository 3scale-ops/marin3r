package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DiscoveryServiceKind         string = "DiscoveryService"
	DiscoveryServiceEnabledKey   string = "marin3r.3scale.net/status"
	DiscoveryServiceEnabledValue string = "enabled"
	DiscoveryServiceLabelKey     string = "marin3r.3scale.net/discovery-service"
)

// DiscoveryServiceSpec defines the desired state of DiscoveryService
type DiscoveryServiceSpec struct {
	DiscoveryServiceNamespace string       `json:"discoveryServiceNamespace"`
	EnabledNamespaces         []string     `json:"enabledNamespaces,omitempty"`
	Signer                    SignerConfig `json:"signer"`
	Image                     string       `json:"image"`
	Debug                     bool         `json:"debug,omitempty"`
}

// SignerConfig holds the config for the marin3r instance certificate signer
type SignerConfig struct {
	CertManager *CertManagerSignerConfig `json:"certManager,omitempty"`
}

// CertManagerSignerConfig holds the specific config for the cert-manager signer
type CertManagerSignerConfig struct {
	// When using cert-manager, the CA needs to be syncronized to the namespace where
	// cert-manager runs. To deploy the CA in a different namespace a command line flag
	// has to be passed to cert-manager, which is not ideal.
	// See https://cert-manager.io/docs/configuration/ca/#deployment.
	Namespace string `json:"namespace,omitempty"`
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
// +kubebuilder:resource:path=discoveryservices,scope=Cluster
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
