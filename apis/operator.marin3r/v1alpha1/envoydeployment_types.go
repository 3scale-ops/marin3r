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

	"github.com/3scale-ops/basereconciler/reconciler"
	defaults "github.com/3scale-ops/marin3r/pkg/envoy/container/defaults"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// DiscoveryServiceCertificateHashLabelKey is the label in the discovery service Deployment that
	// stores the hash of the current server certificate
	EnvoyDeploymentBootstrapConfigHashLabelKey string = "marin3r.3scale.net/bootstrap-config-hash"
	// ClientCertificateDefaultDuration
	ClientCertificateDefaultDuration string = "48h"
	// DefaultReplicas is the default number of replicas for the Deployment
	DefaultReplicas int32 = 1
)

var (
	defaultLivenessProbe ProbeSpec = ProbeSpec{
		InitialDelaySeconds: defaults.LivenessInitialDelaySeconds,
		TimeoutSeconds:      defaults.LivenessTimeoutSeconds,
		PeriodSeconds:       defaults.LivenessPeriodSeconds,
		SuccessThreshold:    defaults.LivenessSuccessThreshold,
		FailureThreshold:    defaults.LivenessFailureThreshold,
	}
	defaultReadinessProbe ProbeSpec = ProbeSpec{
		InitialDelaySeconds: defaults.ReadinessProbeInitialDelaySeconds,
		TimeoutSeconds:      defaults.ReadinessProbeTimeoutSeconds,
		PeriodSeconds:       defaults.ReadinessProbePeriodSeconds,
		SuccessThreshold:    defaults.ReadinessProbeSuccessThreshold,
		FailureThreshold:    defaults.ReadinessProbeFailureThreshold,
	}
	defaultPodDisruptionBudget PodDisruptionBudgetSpec = PodDisruptionBudgetSpec{}
)

// EnvoyDeploymentSpec defines the desired state of EnvoyDeployment
type EnvoyDeploymentSpec struct {
	// EnvoyConfigRef points to an EnvoyConfig in the same namespace
	// that holds the envoy resources for this Deployment
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	EnvoyConfigRef string `json:"envoyConfigRef"`
	// DiscoveryServiceRef points to a DiscoveryService in the same
	// namespace
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	DiscoveryServiceRef string `json:"discoveryServiceRef"`
	// Defines the local service cluster name where Envoy is running. Defaults
	// to the NodeID in the EnvoyConfig if unset
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ClusterID *string `json:"clusterID,omitempty"`
	// Ports exposed by the Envoy container
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Ports []ContainerPort `json:"ports,omitempty"`
	// Image is the envoy image and tag to use
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Image *string `json:"image,omitempty"`
	// Resources holds the resource requirements to use for the Envoy
	// Deployment. Defaults to no resource requests nor limits.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Defines the duration of the client certificate that is used to authenticate
	// with the DiscoveryService
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ClientCertificateDuration *metav1.Duration `json:"duration,omitempty"`
	// Allows the user to define extra command line arguments for the Envoy process
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ExtraArgs []string `json:"extraArgs,omitempty"`
	// Configures envoy's admin port. Defaults to 9901.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AdminPort *uint32 `json:"adminPort,omitempty"`
	// Configures envoy's admin access log path. Defaults to /dev/null.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	AdminAccessLogPath *string `json:"adminAccessLogPath,omitempty"`
	// Replicas configures the number of replicas in the Deployment. One of
	// 'static', 'dynamic' can be set. If both are set, static has precedence.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Replicas *ReplicasSpec `json:"replicas,omitempty"`
	// Liveness probe for the envoy pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	LivenessProbe *ProbeSpec `json:"livenessProbe,omitempty"`
	// Readiness probe for the envoy pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ReadinessProbe *ProbeSpec `json:"readinessProbe,omitempty"`
	// Affinity configuration for the envoy pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Configures PodDisruptionBudget for the envoy Pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	PodDisruptionBudget *PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`
	// ShutdownManager defines configuration for Envoy's shutdown
	// manager, which handles graceful termination of Envoy pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ShutdownManager *ShutdownManager `json:"shutdownManager,omitempty"`
	// InitManager defines configuration for Envoy's init
	// manager, which handles initialization for Envoy pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	InitManager *InitManager `json:"initManager,omitempty"`
}

// Image returns the envoy container image to use
func (ed *EnvoyDeployment) Image() string {
	if ed.Spec.Image == nil {
		return defaults.Image
	}
	return *ed.Spec.Image
}

// Resources returns the Pod resources for the envoy pod
func (ed *EnvoyDeployment) Resources() corev1.ResourceRequirements {
	if ed.Spec.Resources == nil {
		return corev1.ResourceRequirements{}
	}
	return *ed.Spec.Resources
}

// Image returns the envoy container image to use
func (ed *EnvoyDeployment) ClientCertificateDuration() time.Duration {
	if ed.Spec.ClientCertificateDuration == nil {
		d, _ := time.ParseDuration(ClientCertificateDefaultDuration)
		return d
	}
	return ed.Spec.ClientCertificateDuration.Duration
}

func (ed *EnvoyDeployment) AdminPort() uint32 {
	if ed.Spec.AdminPort == nil {
		return defaults.EnvoyAdminPort
	}
	return *ed.Spec.AdminPort
}

func (ed *EnvoyDeployment) AdminAccessLogPath() string {
	if ed.Spec.AdminAccessLogPath == nil {
		return defaults.EnvoyAdminAccessLogPath
	}
	return *ed.Spec.AdminAccessLogPath
}

func (ed *EnvoyDeployment) Replicas() ReplicasSpec {
	if ed.Spec.Replicas == nil {
		return ReplicasSpec{Static: pointer.New(DefaultReplicas)}
	}
	if ed.Spec.Replicas.Static != nil {
		return ReplicasSpec{Static: ed.Spec.Replicas.Static}
	}
	return ReplicasSpec{Dynamic: ed.Spec.Replicas.Dynamic}
}

func (ed *EnvoyDeployment) LivenessProbe() ProbeSpec {
	if ed.Spec.LivenessProbe == nil {
		return defaultLivenessProbe
	}
	return *ed.Spec.LivenessProbe
}

func (ed *EnvoyDeployment) ReadinessProbe() ProbeSpec {
	if ed.Spec.ReadinessProbe == nil {
		return defaultReadinessProbe
	}
	return *ed.Spec.ReadinessProbe
}

func (ed *EnvoyDeployment) Affinity() *corev1.Affinity {
	return ed.Spec.Affinity
}

func (ed *EnvoyDeployment) PodDisruptionBudget() PodDisruptionBudgetSpec {
	if ed.Spec.PodDisruptionBudget == nil {
		return defaultPodDisruptionBudget
	}
	return *ed.Spec.PodDisruptionBudget
}

// ReplicasSpec configures the number of replicas of the Deployment
type ReplicasSpec struct {
	// Configure a static number of replicas. Defaults to 1.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Static *int32 `json:"static,omitempty"`
	// Configure a min and max value for the number of pods to autoscale dynamically.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Dynamic *DynamicReplicasSpec `json:"dynamic,omitempty"`
}

// Validate validates that the received struct is correct
func (rs *ReplicasSpec) Validate() error {
	if rs.Static != nil && rs.Dynamic != nil {
		return fmt.Errorf("only one of 'spec.replicas.static' or 'spec.replicas.dynamic' is allowed")
	}
	return nil
}

type DynamicReplicasSpec struct {
	// minReplicas is the lower limit for the number of replicas to which the autoscaler
	// can scale down.  It defaults to 1 pod.  minReplicas is allowed to be 0 if the
	// alpha feature gate HPAScaleToZero is enabled and at least one Object or External
	// metric is configured.  Scaling is active as long as at least one metric value is
	// available.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// maxReplicas is the upper limit for the number of replicas to which the autoscaler can scale up.
	// It cannot be less that minReplicas.
	MaxReplicas int32 `json:"maxReplicas"`
	// metrics contains the specifications for which to use to calculate the
	// desired replica count (the maximum replica count across all metrics will
	// be used).  The desired replica count is calculated multiplying the
	// ratio between the target value and the current value by the current
	// number of pods.  Ergo, metrics used must decrease as the pod count is
	// increased, and vice-versa.  See the individual metric source types for
	// more information about how each type of metric must respond.
	// If not set, the default metric will be set to 80% average CPU utilization.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Metrics []autoscalingv2.MetricSpec `json:"metrics,omitempty"`
	// behavior configures the scaling behavior of the target
	// in both Up and Down directions (scaleUp and scaleDown fields respectively).
	// If not set, the default HPAScalingRules for scale up and scale down are used.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Behavior *autoscalingv2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
}

// ProbeSpec specifies configuration for a probe
type ProbeSpec struct {
	// Number of seconds after the container has started before liveness probes are initiated
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	InitialDelaySeconds int32 `json:"initialDelaySeconds"`
	// Number of seconds after which the probe times out
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	TimeoutSeconds int32 `json:"timeoutSeconds"`
	// How often (in seconds) to perform the probe
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PeriodSeconds int32 `json:"periodSeconds"`
	// Minimum consecutive successes for the probe to be considered successful after having failed
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SuccessThreshold int32 `json:"successThreshold"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	FailureThreshold int32 `json:"failureThreshold"`
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

// PodDisruptionBudgetSpec defines the PDB for the component
type PodDisruptionBudgetSpec struct {
	// An eviction is allowed if at least "minAvailable" pods selected by
	// "selector" will still be available after the eviction, i.e. even in the
	// absence of the evicted pod.  So for example you can prevent all voluntary
	// evictions by specifying "100%".
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	MinAvailable *intstr.IntOrString `json:"minAvailable,omitempty"`
	// An eviction is allowed if at most "maxUnavailable" pods selected by
	// "selector" are unavailable after the eviction, i.e. even in absence of
	// the evicted pod. For example, one can prevent all voluntary evictions
	// by specifying 0. This is a mutually exclusive setting with "minAvailable".
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
}

// Validate validates that the received struct is correct
func (pdbs *PodDisruptionBudgetSpec) Validate() error {
	if pdbs.MinAvailable != nil && pdbs.MaxUnavailable != nil {
		return fmt.Errorf("only one of 'spec.podDisruptionBudget.minAvailable' or 'spec.podDisruptionBudget.maxUnavailable' is allowed")
	}
	return nil
}

// ShutdownManager defines configuration for Envoy's shutdown
// manager, which handles graceful termination of Envoy Pods
type ShutdownManager struct {
	// Image is the shutdown manager image and tag to use
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Image *string `json:"image,omitempty"`
	// Configures the sutdown manager's server port. Defaults to 8090.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	ServerPort *uint32 `json:"serverPort,omitempty"`
	// The time in seconds that Envoy will drain connections during shutdown.
	// It also affects drain behaviour when listeners are modified or removed via LDS.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	DrainTime *int64 `json:"drainTime,omitempty"`
	// The drain strategy for the graceful shutdown. It also affects
	// drain when listeners are modified or removed via LDS.
	// +kubebuilder:validation:Enum=gradual;immediate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	DrainStrategy *defaults.DrainStrategy `json:"drainStrategy,omitempty"`
}

func (sm *ShutdownManager) GetDrainTime() int64 {
	if sm.DrainTime != nil {
		return *sm.DrainTime
	}
	return defaults.GracefulShutdownTimeoutSeconds
}

func (sm *ShutdownManager) GetDrainStrategy() defaults.DrainStrategy {
	if sm.DrainStrategy != nil {
		return *sm.DrainStrategy
	}
	return defaults.GracefulShutdownStrategy
}

func (sm *ShutdownManager) GetImage() string {
	if sm.Image != nil {
		return *sm.Image
	}
	return defaults.ShtdnMgrImage()
}

func (sm *ShutdownManager) GetServer() uint32 {
	if sm.ServerPort != nil {
		return *sm.ServerPort
	}
	return defaults.ShtdnMgrDefaultServerPort
}

// InitManager defines configuration for Envoy's shutdown
// manager, which handles initialization for Envoy pods
type InitManager struct {
	// Image is the init manager image and tag to use
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Image *string `json:"image,omitempty"`
}

func (im *InitManager) GetImage() string {
	if im.Image != nil {
		return *im.Image
	}
	return defaults.InitMgrImage()
}

// ensure the status implements the AppStatus interface from "github.com/3scale-ops/basereconciler/status"
var _ reconciler.AppStatus = &EnvoyDeploymentStatus{}

// EnvoyDeploymentStatus defines the observed state of EnvoyDeployment
type EnvoyDeploymentStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	DeploymentName *string `json:"deploymentName,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	*appsv1.DeploymentStatus `json:"deploymentStatus,omitempty"`
	// internal fields
	reconciler.UnimplementedStatefulSetStatus `json:"-"`
}

func (eds *EnvoyDeploymentStatus) GetDeploymentStatus(key types.NamespacedName) *appsv1.DeploymentStatus {
	return eds.DeploymentStatus
}

func (eds *EnvoyDeploymentStatus) SetDeploymentStatus(key types.NamespacedName, s *appsv1.DeploymentStatus) {
	eds.DeploymentStatus = s
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EnvoyDeployment is a resource to deploy and manage a Kubernetes Deployment
// of Envoy Pods.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=envoydeployments,scope=Namespaced
// +operator-sdk:csv:customresourcedefinitions:displayName="EnvoyDeployment"
// +operator-sdk:csv:customresourcedefinitions.resources={{Deployment,v1}}
type EnvoyDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyDeploymentSpec   `json:"spec,omitempty"`
	Status EnvoyDeploymentStatus `json:"status,omitempty"`
}

var _ reconciler.ObjectWithAppStatus = &EnvoyDeployment{}

func (ed *EnvoyDeployment) GetStatus() reconciler.AppStatus {
	return &ed.Status
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
