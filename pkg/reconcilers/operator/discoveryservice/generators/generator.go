package generators

import (
	"fmt"
	"time"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type GeneratorOptions struct {
	InstanceName                      string
	Namespace                         string
	RootCertificateNamePrefix         string
	RootCertificateCommonNamePrefix   string
	RootCertificateDuration           time.Duration
	ServerCertificateNamePrefix       string
	ServerCertificateCommonNamePrefix string
	ServerCertificateDuration         time.Duration
	ClientCertificateDuration         time.Duration
	XdsServerPort                     int32
	MetricsServerPort                 int32
	ServiceType                       operatorv1alpha1.ServiceType
	DeploymentImage                   string
	DeploymentResources               corev1.ResourceRequirements
	Debug                             bool
}

func (cfg *GeneratorOptions) labels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "marin3r",
		"app.kubernetes.io/managed-by": "marin3r-operator",
		"app.kubernetes.io/component":  "discovery-service",
		"app.kubernetes.io/instance":   cfg.InstanceName,
	}
}

func (cfg *GeneratorOptions) rootCertName() string {
	return fmt.Sprintf("%s-%s", cfg.RootCertificateNamePrefix, cfg.InstanceName)
}

func (cfg *GeneratorOptions) serverCertName() string {
	return fmt.Sprintf("%s-%s", cfg.ServerCertificateNamePrefix, cfg.InstanceName)
}

func (cfg *GeneratorOptions) resourceName() string {
	return fmt.Sprintf("%s-%s", "marin3r", cfg.InstanceName)
}
