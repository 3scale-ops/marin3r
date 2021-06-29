package generators

import (
	"fmt"
	"time"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/envoy"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type GeneratorOptions struct {
	InstanceName              string
	Namespace                 string
	DiscoveryServiceName      string
	XdssAdress                string
	XdssPort                  int
	EnvoyAPIVersion           envoy.APIVersion
	EnvoyNodeID               string
	EnvoyClusterID            string
	ClientCertificateName     string
	ClientCertificateDuration time.Duration
	SigningCertificateName    string
	DeploymentImage           string
	DeploymentResources       corev1.ResourceRequirements
	ExposedPorts              []operatorv1alpha1.ContainerPort
	ExtraArgs                 []string
	AdminPort                 int32
	AdminAccessLogPath        string
	Replicas                  operatorv1alpha1.ReplicasSpec
	LivenessProbe             operatorv1alpha1.ProbeSpec
	ReadinessProbe            operatorv1alpha1.ProbeSpec
	Affinity                  *corev1.Affinity
	PodDisruptionBudget       operatorv1alpha1.PodDisruptionBudgetSpec
	ShutdownManager           *operatorv1alpha1.ShutdownManager
	InitManager               *operatorv1alpha1.InitManager
}

func (cfg *GeneratorOptions) labels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "marin3r",
		"app.kubernetes.io/managed-by": "marin3r-operator",
		"app.kubernetes.io/component":  "envoy-deployment",
		"app.kubernetes.io/instance":   cfg.InstanceName,
	}
}

func (cfg *GeneratorOptions) resourceName() string {
	return fmt.Sprintf("%s-%s", "marin3r-envoydeployment", cfg.InstanceName)
}

func (cfg *GeneratorOptions) OwnedResourceKey() types.NamespacedName {
	return types.NamespacedName{
		Name:      cfg.resourceName(),
		Namespace: cfg.Namespace,
	}
}
