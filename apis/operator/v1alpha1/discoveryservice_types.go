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
	// Resources holds the Resource Requirements to use for the discovery service
	// Deployment. When not set it defaults to no resource requests nor limits.
	// CPU and Memory resources are supported.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// DiscoveryServiceStatus defines the observed state of DiscoveryService
type DiscoveryServiceStatus struct {
	// Conditions represent the latest available observations of an object's state
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Conditions status.Conditions `json:"conditions"`
}

// +kubebuilder:object:root=true

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

func (d *DiscoveryService) Resources() corev1.ResourceRequirements {
	if d.Spec.Resources == nil {
		return d.defaultDeploymentResources()
	}
	return *d.Spec.Resources
}

func (d *DiscoveryService) defaultDeploymentResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{}
}

// +kubebuilder:object:root=true

// DiscoveryServiceList contains a list of DiscoveryService
type DiscoveryServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DiscoveryService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DiscoveryService{}, &DiscoveryServiceList{})
}
