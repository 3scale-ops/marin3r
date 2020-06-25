package v1alpha1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	/* Conditions */

	// CacheOutOfSyncCondition is a condition that indicates that the
	// envoyconfig cannot reach the desired status specified in the spec,
	// usually because the config in the spec is incorrect or has caused failures
	// in the envoy clients
	CacheOutOfSyncCondition status.ConditionType = "CacheOutOfSync"

	// RollbackFailedCondition indicates that the EnvoyConfig object
	// is not able to publish a config revision because all revisions are
	// tainted
	RollbackFailedCondition status.ConditionType = "RollbackFailed"

	/* Finalizers */

	// EnvoyConfigFinalizer is the finalizer for EnvoyConfig objects
	EnvoyConfigFinalizer string = "finalizer.marin3r.3scale.net"

	/* State */

	//InSyncState indicates that a NodeCacheConfig object has its resources spec
	// in sync with the xds server cache
	InSyncState string = "InSync"

	// RollbackState indicates that a EnvoyConfig object has performed a
	// rollback to a previous version of the resources spec
	RollbackState string = "Rollback"

	// RollbackFailedState indicates that there is no untainted revision that
	// can be pusblished in the xds server cache
	RollbackFailedState string = "RollbackFailed"
)

// EnvoyConfigSpec defines the desired state of EnvoyConfig
type EnvoyConfigSpec struct {
	// +kubebuilder:validation:Pattern:[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
	NodeID string `json:"nodeID"`
	// +kubebuilder:validation:Enum=json;b64json;yaml
	Serialization string          `json:"serialization,omitempty"`
	Resources     *EnvoyResources `json:"resources"`
}

// EnvoyResources holds each envoy api resource type
type EnvoyResources struct {
	Endpoints []EnvoyResource       `json:"endpoints,omitempty"`
	Clusters  []EnvoyResource       `json:"clusters,omitempty"`
	Routes    []EnvoyResource       `json:"routes,omitempty"`
	Listeners []EnvoyResource       `json:"listeners,omitempty"`
	Runtimes  []EnvoyResource       `json:"runtime,omitempty"`
	Secrets   []EnvoySecretResource `json:"secrets,omitempty"`
}

// EnvoyResource holds a single envoy api resources,
// serialized as json, base64 encoded
type EnvoyResource struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// EnvoySecretResource holds a reference to a k8s
// Secret from where to take a secret from
type EnvoySecretResource struct {
	Name string                 `json:"name"`
	Ref  corev1.SecretReference `json:"ref"`
}

// EnvoyConfigStatus defines the observed state of EnvoyConfig
type EnvoyConfigStatus struct {
	// CacheState summarizes all the observations about the NodeCacheConfig
	// to give the user a coecrete idea on the general status of the cache. It is intended
	// only for human consumption. Other controllers should relly on conditions to determine
	// the status of the cache
	CacheState string `json:"cacheState,omitempty"`
	// PublishedVersion is the config version currently
	// served by the envoy control plane for the node-id
	PublishedVersion string `json:"publishedVersion,omitempty"`
	// DesiredVersion represents the resources version described in
	// the spec of the envoyconfigrevision object
	DesiredVersion string `json:"desiredVersion,omitempty"`
	// Conditions represent the latest available observations of an object's state
	Conditions status.Conditions `json:"conditions,omitempty"`
	// ConfigRevisions is an ordered list of references to the envoyconfigrevision
	// object
	ConfigRevisions []ConfigRevisionRef `json:"revisions,omitempty"`
}

// ConfigRevisionRef holds a reference to EnvoyConfigRevision object
type ConfigRevisionRef struct {
	Version string                 `json:"version"`
	Ref     corev1.ObjectReference `json:"ref"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EnvoyConfig is the Schema for the envoyconfigs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=envoyconfigs,scope=Namespaced,shortName=ec
// +kubebuilder:printcolumn:JSONPath=".spec.nodeID",name=NodeID,type=string
// +kubebuilder:printcolumn:JSONPath=".status.desiredVersion",name=Desired Version,type=string
// +kubebuilder:printcolumn:JSONPath=".status.publishedVersion",name=Published Version,type=string
// +kubebuilder:printcolumn:JSONPath=".status.cacheState",name=Cache State,type=string
type EnvoyConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyConfigSpec   `json:"spec,omitempty"`
	Status EnvoyConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EnvoyConfigList contains a list of EnvoyConfig
type EnvoyConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnvoyConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnvoyConfig{}, &EnvoyConfigList{})
}