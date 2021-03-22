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

	"github.com/3scale/marin3r/pkg/version"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DiscoveryServiceKind is Kind of the DiscoveryService resources
	DiscoveryServiceKind string = "DiscoveryService"
	// DiscoveryServiceListKind is the Kind of the DiscoveryServiceList resources
	DiscoveryServiceListKind string = "DiscoveryServiceList"
	// DiscoveryServiceEnabledKey is the label key that the mutating webhook uses
	// to determine if mutation is enabled for a Pod
	DiscoveryServiceEnabledKey string = "marin3r.3scale.net/status"
	// DiscoveryServiceEnabledValue is the label value that the mutating webhook uses
	// to determine if mutation is enabled for a Pod
	DiscoveryServiceEnabledValue string = "enabled"
	// DiscoveryServiceLabelKey is the label key that the mutating webhook uses to determine if
	// Pod mutation is enabled in a namespace
	DiscoveryServiceLabelKey string = "marin3r.3scale.net/discovery-service"
	// DiscoveryServiceCertificateHashLabelKey is the label in the discovery service Deployment that
	// stores the hash of the current server certificate
	DiscoveryServiceCertificateHashLabelKey string = "marin3r.3scale.net/server-certificate-hash"

	// DiscoveryServiceFinalizer is the finalizer for DiscoveryService objects
	DiscoveryServiceFinalizer string = "finalizer.operator.marin3r.3scale.net"

	/* Default values */

	// DefaultMetricsPort is the default port where the discovery service metrics server listens
	DefaultMetricsPort uint32 = 8383
	// DefaultWebhookPort is the default port where the discovery service webhook server listens
	DefaultWebhookPort uint32 = 9443
	// DefaultXdsServerPort is the default port where the discovery service xds server port listens
	DefaultXdsServerPort uint32 = 18000
	// DefaultRootCertificateDuration is the default root CA certificate duration
	DefaultRootCertificateDuration string = "26280h" // 3 years
	// DefaultRootCertificateSecretNamePrefix is the default prefix for the Secret
	// where the root CA certificate is stored
	DefaultRootCertificateSecretNamePrefix string = "marin3r-ca-cert"
	// DefaultServerCertificateDuration is the default discovery service server certificate duration
	DefaultServerCertificateDuration string = "2160h" // 3 months
	// DefaultServerCertificateSecretNamePrefix is the default prefix for the Secret
	// where the server certificate is stored
	DefaultServerCertificateSecretNamePrefix string = "marin3r-server-cert"
	// DefaultImageRegistry is the default registry to pull discovery service images from
	DefaultImageRegistry string = "quay.io/3scale/marin3r"
)

// ServiceType is an enum with the available discovery service Service types
type ServiceType string

const (
	// ClusterIPType represents a ClusterIP Service
	ClusterIPType ServiceType = "ClusterIP"
	// LoadBalancerType represents a LoadBalancer Service
	LoadBalancerType ServiceType = "LoadBalancer"
	// HeadlessType represents a headless Service
	HeadlessType ServiceType = "Headless"
)

// DiscoveryServiceSpec defines the desired state of DiscoveryService
type DiscoveryServiceSpec struct {
	// Image holds the image to use for the discovery service Deployment
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Image *string `json:"image,omitempty"`
	// Debug enables debugging log level for the discovery service controllers. It is safe to
	// use since secret data is never shown in the logs.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Debug *bool `json:"debug,omitempty"`
	// Resources holds the Resource Requirements to use for the discovery service
	// Deployment. When not set it defaults to no resource requests nor limits.
	// CPU and Memory resources are supported.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// PKIConfig has configuration for the PKI that marin3r manages for the
	// different certificates it requires
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	PKIConfig *PKIConfig `json:"pkiConfg,omitempty"`
	// XdsServerPort is the port where the xDS server listens. Defaults to 18000.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	XdsServerPort *uint32 `json:"xdsServerPort,omitempty"`
	// MetricsPort is the port where metrics are served. Defaults to 8383.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	MetricsPort *uint32 `json:"metricsPort,omitempty"`
	// ServiceConfig configures the way the DiscoveryService endpoints are exposed
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ServiceConfig *ServiceConfig `json:"serviceConfig,omitempty"`
}

// DiscoveryServiceStatus defines the observed state of DiscoveryService
type DiscoveryServiceStatus struct {
	// Conditions represent the latest available observations of an object's state
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	Conditions status.Conditions `json:"conditions,omitempty"`
}

// PKIConfig has configuration for the PKI that marin3r manages for the
// different certificates it requires
type PKIConfig struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	RootCertificateAuthority *CertificateOptions `json:"rootCertificateAuthority"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ServerCertificate *CertificateOptions `json:"serverCertificate"`
}

// CertificateOptions specifies options to generate the server certificate used both
// for the xDS server and the mutating webhook server.
type CertificateOptions struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SecretName string `json:"secretName"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Duration metav1.Duration `json:"duration"`
}

// ServiceConfig has options to configure the way the Service
// is deployed
type ServiceConfig struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Type ServiceType `json:"type,omitempty"`
}

// +kubebuilder:object:root=true

// DiscoveryService represents an envoy discovery service server. Currently
// only one DiscoveryService per cluster is supported.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=discoveryservices,scope=Namespaced
// +operator-sdk:csv:customresourcedefinitions:displayName="DiscoveryService"
// +operator-sdk:csv:customresourcedefinitions.resources={{Deployment,v1},{Service,v1},{DiscoveryServiceCertificate,v1alpha1}
type DiscoveryService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DiscoveryServiceSpec   `json:"spec,omitempty"`
	Status DiscoveryServiceStatus `json:"status,omitempty"`
}

// Resources returns the Pod resources for the discovery service pod
func (d *DiscoveryService) Resources() corev1.ResourceRequirements {
	if d.Spec.Resources == nil {
		return d.defaultDeploymentResources()
	}
	return *d.Spec.Resources
}

// GetImage returns the DiscoveryService image that matches the current version of the operator
// or the one defined by the user if the filed is set in the resource
func (d *DiscoveryService) GetImage() string {
	if d.Spec.Image == nil {
		return d.defaultImage()
	}
	return *d.Spec.Image
}

func (d *DiscoveryService) defaultImage() string {
	return fmt.Sprintf("%s:%s", DefaultImageRegistry, version.Current())
}

// Debug returns a boolean value that indicates if debug loggin is enabled
func (d *DiscoveryService) Debug() bool {
	if d.Spec.Debug == nil {
		return false
	}
	return *d.Spec.Debug
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

// GetXdsServerPort returns the port the xDS server will listen at
func (d *DiscoveryService) GetXdsServerPort() uint32 {
	if d.Spec.XdsServerPort != nil {
		return *d.Spec.XdsServerPort
	}
	return DefaultXdsServerPort
}

// GetMetricsPort returns the port the metrics server will listen at
func (d *DiscoveryService) GetMetricsPort() uint32 {
	if d.Spec.MetricsPort != nil {
		return *d.Spec.MetricsPort
	}
	return DefaultMetricsPort
}

// GetServiceConfig returns the Service configuration for the discovery service servers
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

// OwnedObjectName returns the name of the resources the discoveryservices controller
// needs to create
func (d *DiscoveryService) OwnedObjectName() string {
	return fmt.Sprintf("%s-%s", "marin3r", d.GetName())
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
