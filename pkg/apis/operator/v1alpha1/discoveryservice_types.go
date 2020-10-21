package v1alpha1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DiscoveryServiceKind                    string = "DiscoveryService"
	DiscoveryServiceEnabledKey              string = "marin3r.3scale.net/status"
	DiscoveryServiceEnabledValue            string = "enabled"
	DiscoveryServiceLabelKey                string = "marin3r.3scale.net/discovery-service"
	DiscoveryServiceCertificateHashLabelKey string = "marin3r.3scale.net/server-certificate-hash"
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
	// Image holds the image to use for the discovery service Deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Image string `json:"image"`
	// Debug enables debugging log level for the discovery service controllers. It is safe to
	// use since secret data is never shown in the logs.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Debug bool `json:"debug,omitempty"`
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
