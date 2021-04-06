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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DiscoveryServiceCertificateHashLabelKey is the label in the discovery service Deployment that
	// stores the hash of the current server certificate
	EnvoyDeploymentBootstrapConfigHashLabelKey string = "marin3r.3scale.net/bootstrap-config-hash"
)

// EnvoyDeploymentSpec defines the desired state of EnvoyDeployment
type EnvoyDeploymentSpec struct {
	// EnvoyConfigRef points to an EnvoyConfig in the same namespace
	// that holds the envoy resources for this Deployment
	EnvoyConfigRef string `json:"envoyConfigRef"`
	// Ports exposed by the Envoy container
	// TODO: calculate this inspecting the list of listeners in the
	// published EnvoyConfigRevision
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Ports []ContainerPort `json:"ports,omitempty"`
	// Image is the envoy image and tag to use
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Image *string `json:"image,omitempty"`
	// Resources holds the resource requirements to use for the Envoy
	// Deployment. When not set it defaults to no resource requests nor limits.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// TODO: customizations for labels, annotations and probes
}

// ContainerPort defines port for the Marin3r sidecar container
type ContainerPort struct {
	// Port name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`
	// Port value
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Port int32 `json:"port"`
	// Protocol. Defaults to TCP.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Protocol *corev1.Protocol `json:"protocol,omitempty"`
}

// Resources returns the Pod resources for the envoy pod
func (d *EnvoyDeployment) Resources() corev1.ResourceRequirements {
	if d.Spec.Resources == nil {
		return d.defaultDeploymentResources()
	}
	return *d.Spec.Resources
}

func (d *EnvoyDeployment) defaultDeploymentResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{}
}

// EnvoyDeploymentStatus defines the observed state of EnvoyDeployment
type EnvoyDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EnvoyDeployment is the Schema for the envoydeployments API
type EnvoyDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyDeploymentSpec   `json:"spec,omitempty"`
	Status EnvoyDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EnvoyDeploymentList contains a list of EnvoyDeployment
type EnvoyDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnvoyDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnvoyDeployment{}, &EnvoyDeploymentList{})
}
