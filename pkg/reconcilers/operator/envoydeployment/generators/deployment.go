package generators

import (
	"strings"

	envoy_container "github.com/3scale-ops/marin3r/pkg/envoy/container"
	defaults "github.com/3scale-ops/marin3r/pkg/envoy/container/defaults"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func (cfg *GeneratorOptions) Deployment() func() *appsv1.Deployment {

	return func() *appsv1.Deployment {

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
			ConfigBasePath:     defaults.EnvoyConfigBasePath,
			ConfigFileName:     defaults.EnvoyConfigFileName,
			ConfigVolume:       defaults.DeploymentConfigVolume,
			TLSBasePath:        defaults.EnvoyTLSBasePath,
			TLSVolume:          defaults.DeploymentTLSVolume,
			NodeID:             cfg.EnvoyNodeID,
			ClusterID:          cfg.EnvoyClusterID,
			ClientCertSecret:   strings.Join([]string{defaults.DeploymentClientCertificate, cfg.InstanceName}, "-"),
			ExtraArgs:          cfg.ExtraArgs,
			Resources:          cfg.DeploymentResources,
			AdminBindAddress:   defaults.EnvoyAdminBindAddress,
			AdminPort:          cfg.AdminPort,
			AdminAccessLogPath: cfg.AdminAccessLogPath,
			LivenessProbe:      cfg.LivenessProbe,
			ReadinessProbe:     cfg.ReadinessProbe,
			InitManagerImage:   defaults.InitMgrImage(),
			XdssHost:           cfg.XdssAdress,
			XdssPort:           cfg.XdssPort,
			APIVersion:         cfg.EnvoyAPIVersion.String(),
		}

		if cfg.ShutdownManager != nil {
			cc.ShutdownManagerImage = cfg.ShutdownManager.GetImage()
			cc.ShutdownManagerEnabled = true
			cc.ShutdownManagerPort = int32(defaults.ShtdnMgrDefaultServerPort)
			cc.ShutdownManagerDrainSeconds = cfg.ShutdownManager.GetDrainTime()
			cc.ShutdownManagerDrainStrategy = cfg.ShutdownManager.GetDrainStrategy()
		}

		if cfg.InitManager != nil {
			cc.InitManagerImage = cfg.InitManager.GetImage()
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
				// this value will be overwritten by the basereconciler
				// if HPA is enabled
				Replicas: cfg.Replicas.Static,
				Selector: &metav1.LabelSelector{
					MatchLabels: cfg.labels(),
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{},
						Labels:            cfg.labels(),
					},
					Spec: corev1.PodSpec{
						Affinity:                 cfg.Affinity,
						Volumes:                  cc.Volumes(),
						InitContainers:           cc.InitContainers(),
						Containers:               cc.Containers(),
						RestartPolicy:            corev1.RestartPolicyAlways,
						DNSPolicy:                corev1.DNSClusterFirst,
						ServiceAccountName:       "default",
						DeprecatedServiceAccount: "default",
						TerminationGracePeriodSeconds: func() *int64 {
							// Match the Popd's TerminationGracePeriodSeconds to the
							// configured Envoy DrainTime
							if cfg.ShutdownManager != nil {
								d := cfg.ShutdownManager.GetDrainTime()
								return &d
							}
							return pointer.Int64(corev1.DefaultTerminationGracePeriodSeconds)
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
				RevisionHistoryLimit:    pointer.Int32(10),
				ProgressDeadlineSeconds: pointer.Int32(600),
			},
		}

		return dep
	}
}
