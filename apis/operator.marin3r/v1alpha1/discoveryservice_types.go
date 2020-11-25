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
	"fmt"
	"time"

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

	/* Default values */
	DefaultMetricsPort                       uint32 = 8383
	DefaultWebhookPort                       uint32 = 8443
	DefaultXdsServerPort                     uint32 = 18000
	DefaultRootCertificateDuration           string = "26280h" // 3 years
	DefaultRootCertificateSecretNamePrefix   string = "marin3r-ca-cert"
	DefaultServerCertificateDuration         string = "2160h" // 3 months
	DefaultServerCertificateSecretNamePrefix string = "marin3r-server-cert"
)

type ServiceType string

const (
	ClusterIPType    ServiceType = "ClusterIP"
	LoadBalancerType ServiceType = "LoadBalancer"
	HeadlessType     ServiceType = "Headless"
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
	// PKIConfig has configuration for the PKI that marin3r manages for the
	// different certificates it requires
	// +optional
	PKIConfig *PKIConfig `json:"pkiConfg,omitempty"`
	// XdsServerPort is the port where the xDS server listens. Defaults to 18000.
	// +optional
	XdsServerPort *uint32 `json:"xdsPort,omitempty"`
	// WebhookPort is the port where the Pod mutating webhooks listens. Defaults to 8443.
	// +optional
	WebhookPort *uint32 `json:"webhookPort,omitempty"`
	// MetricsPort is the port where metrics are served. Defaults to 8383.
	// +optional
	MetricsPort *uint32 `json:"metricsPort,omitempty"`
	// ServiceConfig configures the way the DiscoveryService endpoints are exposed
	// +optional
	ServiceConfig *ServiceConfig `json:"ServiceConfig,omitempty"`
}

// DiscoveryServiceStatus defines the observed state of DiscoveryService
type DiscoveryServiceStatus struct {
	// Conditions represent the latest available observations of an object's state
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Conditions status.Conditions `json:"conditions"`
}

// PKIConfig has configuration for the PKI that marin3r manages for the
// different certificates it requires
type PKIConfig struct {
	RootCertificateAuthority *CertificateOptions `json:"rootCertificateAuthority"`
	ServerCertificate        *CertificateOptions `json:"serverCertificate"`
}

// CertificateOptions specifies options to generate the server certificate used both
// for the xDS server and the mutating webhook server.
type CertificateOptions struct {
	SecretName string          `json:"secretName"`
	Duration   metav1.Duration `json:"duration"`
}

// ServiceConfig has options to configure the way the Service
// is deployed
type ServiceConfig struct {
	Name string      `json:"name,omitempty"`
	Type ServiceType `json:"type,omitempty"`
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

// GetRootCertificateAuthorityOptions returns the CertificateOptions for the root CA
func (d *DiscoveryService) GetRootCertificateAuthorityOptions() *CertificateOptions {
	if d.Spec.PKIConfig != nil && d.Spec.PKIConfig.RootCertificateAuthority != nil {
		return d.Spec.PKIConfig.RootCertificateAuthority
	}
	return d.defaultRootCertificateAuthorityOptions()
}

func (d *DiscoveryService) defaultRootCertificateAuthorityOptions() *CertificateOptions {
	return &CertificateOptions{
		SecretName: fmt.Sprintf("%s-%s", DefaultRootCertificateSecretNamePrefix, d.Name),
		Duration: metav1.Duration{
			Duration: func() time.Duration {
				d, _ := time.ParseDuration(DefaultRootCertificateDuration)
				return d
			}(),
		}}
}

// GetServerCertificateOptions returns the CertificateOptions for the root CA
func (d *DiscoveryService) GetServerCertificateOptions() *CertificateOptions {
	if d.Spec.PKIConfig != nil && d.Spec.PKIConfig.ServerCertificate != nil {
		return d.Spec.PKIConfig.ServerCertificate
	}
	return d.defaultServerCertificateOptions()
}

func (d *DiscoveryService) defaultServerCertificateOptions() *CertificateOptions {
	return &CertificateOptions{
		SecretName: fmt.Sprintf("%s-%s", DefaultServerCertificateSecretNamePrefix, d.Name),
		Duration: metav1.Duration{
			Duration: func() time.Duration {
				d, _ := time.ParseDuration(DefaultServerCertificateDuration)
				return d
			}(),
		}}
}

func (d *DiscoveryService) GetXdsServerPort() uint32 {
	if d.Spec.XdsServerPort != nil {
		return *d.Spec.XdsServerPort
	}
	return DefaultXdsServerPort
}

func (d *DiscoveryService) GetMetricsPort() uint32 {
	if d.Spec.MetricsPort != nil {
		return *d.Spec.MetricsPort
	}
	return DefaultMetricsPort
}

func (d *DiscoveryService) GetWebhookPort() uint32 {
	if d.Spec.WebhookPort != nil {
		return *d.Spec.WebhookPort
	}
	return DefaultWebhookPort
}

func (d *DiscoveryService) GetServiceConfig() *ServiceConfig {
	if d.Spec.ServiceConfig != nil {
		return d.Spec.ServiceConfig
	}
	return d.defaultServiceConfig()
}

func (d *DiscoveryService) defaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		Name: d.OwnedObjectName(),
		Type: ClusterIPType,
	}
}

func (d *DiscoveryService) OwnedObjectName() string {
	return fmt.Sprintf("%s-%s", "discoveryservice", d.GetName())
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
