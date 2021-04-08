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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// EnvoyBootstrapKind is Kind of the EnvoyBootstrap resources
	EnvoyBootstrapKind string = "EnvoyBootstrap"
)

// EnvoyBootstrapSpec defines the desired state of EnvoyBootstrap
type EnvoyBootstrapSpec struct {
	// DiscoveryService is the name of the DiscoveryService resource the envoy will be a client of
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	DiscoveryService string `json:"discoveryService"`
	// ClientCertificate is a struct containing options for the certificate used to authenticate with the
	// discovery service
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ClientCertificate ClientCertificate `json:"clientCertificate"`
	// EnvoyStaticConfig is a struct that controls options for the envoy's static config file
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	EnvoyStaticConfig EnvoyStaticConfig `json:"envoyStaticConfig"`
}

// EnvoyStaticConfig allows specifying envoy static config
// options
type EnvoyStaticConfig struct {
	// The ConfigMap where the envoy client v2 static config will be stored
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ConfigMapNameV2 string `json:"configMapNameV2"`
	// The ConfigMap where the envoy client v3 static config will be stored
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ConfigMapNameV3 string `json:"configMapNameV3"`
	// ConfigFile is the path of envoy's bootstrap config file
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ConfigFile string `json:"configFile"`
	// ResourcesDir is the path where resource files are loaded from. It is used to
	// load discovery messages directly from the filesystem, for example in order to be able
	// to bootstrap certificates and support rotation when they are modified.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ResourcesDir string `json:"resourcesDir"`
	// RtdsLayerResourceName is the resource name that the envoy client will request when askikng
	// the discovery service for Runtime resources.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	RtdsLayerResourceName string `json:"rtdsLayerResourceName"`
	// AdminBindAddress is where envoy's admin server binds to.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	AdminBindAddress string `json:"adminBindAddress"`
	// AdminAccessLogPath configures where the envoy's admin server logs are written to
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	AdminAccessLogPath string `json:"adminAccessLogPath"`
}

// ClientCertificate allows specifying options for the
// client certificate used to authenticate with the discovery
// service
type ClientCertificate struct {
	// Directory defines the directory in the envoy container where
	// the certificate will be mounted
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Directory string `json:"directory"`
	// The Secret where the certificate will be stored
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SecretName string `json:"secretName"`
	// The requested ‘duration’ (i.e. lifetime) of the Certificate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Duration metav1.Duration `json:"duration"`
}

// EnvoyBootstrapStatus defines the observed state of EnvoyBootstrap
type EnvoyBootstrapStatus struct {
	// ConfigHashV2 stores the hash of the current V2 bootstrap
	// config generated for the given EnvoyBootstrap parameters
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	ConfigHashV2 *string `json:"configHashV2,omitempty"`
	// ConfigHashV3 stores the hash of the current V3 bootstrap
	// config generated for the given EnvoyBootstrap parameters
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +optional
	ConfigHashV3 *string `json:"configHashV3,omitempty"`
}

// GetConfigHashV2 returns the hash of the v2 bootstrap config.
// Returns an empty string if not set.
func (status *EnvoyBootstrapStatus) GetConfigHashV2() string {
	if status.ConfigHashV2 == nil {
		return ""
	}
	return *status.ConfigHashV2
}

// GetConfigHashV3 returns the hash of the v3 bootstrap config.
// Returns an empty string if not set.
func (status *EnvoyBootstrapStatus) GetConfigHashV3() string {
	if status.ConfigHashV3 == nil {
		return ""
	}
	return *status.ConfigHashV3
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EnvoyBootstrap is the Schema for the envoybootstraps API
// +operator-sdk:csv:customresourcedefinitions:displayName="EnvoyBootstrap"
type EnvoyBootstrap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyBootstrapSpec   `json:"spec,omitempty"`
	Status EnvoyBootstrapStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnvoyBootstrapList contains a list of EnvoyBootstrap
type EnvoyBootstrapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnvoyBootstrap `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnvoyBootstrap{}, &EnvoyBootstrapList{})
}
