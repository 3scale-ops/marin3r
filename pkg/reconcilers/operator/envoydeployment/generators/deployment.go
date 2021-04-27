package generators

import (
	"strings"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_container "github.com/3scale-ops/marin3r/pkg/envoy/container"
	defaults "github.com/3scale-ops/marin3r/pkg/envoy/container/defaults"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/lockedresources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (cfg *GeneratorOptions) Deployment(hash string, replicas *int32) lockedresources.GeneratorFunction {

	return func() client.Object {

		cc := envoy_container.ContainerConfig{
			Name:  defaults.DeploymentContainerName,
			Image: cfg.DeploymentImage,
			Ports: func() []corev1.ContainerPort {
				ports := make([]corev1.ContainerPort, len(cfg.ExposedPorts))
				for i := 0; i < len(cfg.ExposedPorts); i++ {
					p := corev1.ContainerPort{
						Name:          cfg.ExposedPorts[i].Name,
						ContainerPort: cfg.ExposedPorts[i].Port,
					}
					if cfg.ExposedPorts[i].Protocol != nil {
						p.Protocol = *cfg.ExposedPorts[i].Protocol
					}
					ports[i] = p
				}
				return ports
			}(),
			BootstrapConfigMap: func() string {
				if cfg.EnvoyAPIVersion == envoy.APIv2 {
					return strings.Join([]string{defaults.DeploymentBootstrapConfigMapV2, cfg.InstanceName}, "-")
				}
				return strings.Join([]string{defaults.DeploymentBootstrapConfigMapV3, cfg.InstanceName}, "-")
			}(),
			ConfigBasePath:   defaults.EnvoyConfigBasePath,
			ConfigFileName:   defaults.EnvoyConfigFileName,
			ConfigVolume:     defaults.DeploymentConfigVolume,
			TLSBasePath:      defaults.EnvoyTLSBasePath,
			TLSVolume:        defaults.DeploymentTLSVolume,
			NodeID:           cfg.EnvoyNodeID,
			ClusterID:        cfg.EnvoyClusterID,
			ClientCertSecret: strings.Join([]string{defaults.DeploymentClientCertificate, cfg.InstanceName}, "-"),
			ExtraArgs:        cfg.ExtraArgs,
			Resources:        cfg.DeploymentResources,
			AdminPort:        cfg.AdminPort,
			LivenessProbe:    cfg.LivenessProbe,
			ReadinessProbe:   cfg.ReadinessProbe,
		}

		if cfg.ShutdownManager != nil {
			cc.ShutdownManagerImage = cfg.ShutdownManager.GetImage()
			cc.ShutdownManagerEnabled = true
			cc.ShutdownManagerPort = int32(defaults.ShtdnMgrDefaultServerPort)
		}

		dep := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: appsv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      cfg.resourceName(),
				Namespace: cfg.Namespace,
				Labels:    cfg.labels(),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: cfg.labels(),
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{},
						Labels: func() (labels map[string]string) {
							labels = cfg.labels()
							labels[operatorv1alpha1.EnvoyDeploymentBootstrapConfigHashLabelKey] = hash
							return
						}(),
					},
					Spec: corev1.PodSpec{
						Affinity:                 cfg.PodAffinity,
						Volumes:                  cc.Volumes(),
						Containers:               cc.Containers(),
						RestartPolicy:            corev1.RestartPolicyAlways,
						DNSPolicy:                corev1.DNSClusterFirst,
						ServiceAccountName:       "default",
						DeprecatedServiceAccount: "default",
						TerminationGracePeriodSeconds: func() *int64 {
							// Increase the TerminationGracePeriod timeout if the shutdown manager
							// is enabled (for graceful termination)
							if cfg.ShutdownManager != nil {
								return pointer.Int64Ptr(defaults.GracefulShutdownTimeoutSeconds)
							}
							return pointer.Int64Ptr(corev1.DefaultTerminationGracePeriodSeconds)
						}(),
						SecurityContext: &corev1.PodSecurityContext{},
						SchedulerName:   corev1.DefaultSchedulerName,
					},
				},
				Strategy: appsv1.DeploymentStrategy{
					Type: appsv1.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &appsv1.RollingUpdateDeployment{
						MaxUnavailable: &intstr.IntOrString{
							Type:   intstr.String,
							StrVal: "25%",
						},
						MaxSurge: &intstr.IntOrString{
							Type:   intstr.String,
							StrVal: "25%",
						},
					},
				},
				RevisionHistoryLimit:    pointer.Int32Ptr(10),
				ProgressDeadlineSeconds: pointer.Int32Ptr(600),
			},
		}

		// Enforce a fixed number of replicas if static replicas is active
		if cfg.Replicas.Static != nil {
			dep.Spec.Replicas = cfg.Replicas.Static
		}

		return dep
	}
}
