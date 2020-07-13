package v1alpha1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DiscoveryServiceKind           string               = "DiscoveryService"
	DiscoveryServiceEnabledKey     string               = "marin3r.3scale.net/status"
	DiscoveryServiceEnabledValue   string               = "enabled"
	DiscoveryServiceLabelKey       string               = "marin3r.3scale.net/discovery-service"
	ServerRestartRequiredCondition status.ConditionType = "ServerRestartRequired"
)

// DiscoveryServiceSpec defines the desired state of DiscoveryService
type DiscoveryServiceSpec struct {
	// DiscoveryServiceNamespcae is the name of the namespace where the envoy discovery
	// service server should be deployed.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	DiscoveryServiceNamespace string `json:"discoveryServiceNamespace"`
	// EnabledNamespaces is a list of namespaces where the envoy discovery service is
	// enabled. In order to be able to use marin3r from a given namespace its name needs
	// to be included in this list because the operator needs to add some required resources in
	// that namespace.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	EnabledNamespaces []string `json:"enabledNamespaces,omitempty"`
	// Signer holds the configuration for a certificate signer. This signer will be used to
	// setup mTLS between envoy clients and the discovery service server.
	// Currently only the CertManager signer is supported.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Signer SignerConfig `json:"signer"`
	// Image holds the image to use for the discovery service Deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Image string `json:"image"`
	// Debug enables debugging log level for the discovery service controllers. It is safe to
	// use since secret data is never shown in the logs.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Debug bool `json:"debug,omitempty"`
}

// SignerConfig holds the config for the marin3r instance certificate signer
type SignerConfig struct {
	// CertManager holds specific config for the CertManager signer.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CertManager *CertManagerSignerConfig `json:"certManager,omitempty"`
}

// CertManagerSignerConfig holds the specific config for the cert-manager signer
type CertManagerSignerConfig struct {
	// Namespace is the name of the namespace where the cert-manager controller runs.
	// This field is required due to the fact that a CA ClusterIssuer needs the CA Secret
	// to be present in its own namespace. The marin3r-operator syncs the CA Secret from its
	// original namespace to the cert-manager namespace.
	// See https://cert-manager.io/docs/configuration/ca/#deployment.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Namespace string `json:"namespace,omitempty"`
}

// DiscoveryServiceStatus defines the observed state of DiscoveryService
type DiscoveryServiceStatus struct {
	// Conditions represent the latest available observations of an object's state
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Conditions status.Conditions `json:"conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DiscoveryService represents an envoy discovery service server. Currently
// only one DiscoveryService per cluster is supported.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=discoveryservices,scope=Cluster
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="DiscoveryService"
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Deployment,v1`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Service,v1`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`MutatingWebhookConfiguration,v1`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`DiscoveryServiceCertificate,v1alpha1`
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
