package generators

import (
	"fmt"

	"github.com/3scale-ops/marin3r/pkg/envoy"
	defaults "github.com/3scale-ops/marin3r/pkg/envoy/bootstrap/defaults"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/lockedresources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (cfg *GeneratorOptions) Deployment(hash string) lockedresources.GeneratorFunction {

	return func() client.Object {

		return &appsv1.Deployment{
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
				Replicas: pointer.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: cfg.labels(),
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{},
						Labels: func() (labels map[string]string) {
							labels = cfg.labels()
							// TODO: Hash the bootstrap config
							// labels[operatorv1alpha1.DiscoveryServiceCertificateHashLabelKey] = hash
							return
						}(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: defaults.DeploymentTLSVolume,
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: defaults.DeploymentClientCertificate,
									},
								},
							},
							{
								Name: defaults.DeploymentConfigVolume,
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: func() string {
												if cfg.EnvoyAPIVersion == envoy.APIv2 {
													return defaults.DeploymentBootstrapConfigMapV2
												}
												return defaults.DeploymentBootstrapConfigMapV3
											}(),
										},
									},
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name:    defaults.DeploymentContainerName,
								Image:   cfg.DeploymentImage,
								Command: []string{"envoy"},
								Args: []string{
									"-c",
									fmt.Sprintf("%s/%s", defaults.EnvoyConfigBasePath, defaults.EnvoyConfigFileName),
									"--service-node",
									cfg.EnvoyNodeID,
									"--service-cluster",
									cfg.EnvoyClusterID,
								},
								// TODO
								// Resources: {},
								// Ports: {},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      defaults.DeploymentTLSVolume,
										ReadOnly:  true,
										MountPath: defaults.EnvoyTLSBasePath,
									},
									{
										Name:      defaults.DeploymentConfigVolume,
										ReadOnly:  true,
										MountPath: defaults.EnvoyConfigBasePath,
									},
								},
								LivenessProbe: &corev1.Probe{
									Handler: corev1.Handler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/ready",
											Port: intstr.IntOrString{IntVal: 9901},
										},
									},
									InitialDelaySeconds: 30,
									TimeoutSeconds:      1,
									PeriodSeconds:       10,
									SuccessThreshold:    1,
									FailureThreshold:    10,
								},
								ReadinessProbe: &corev1.Probe{
									Handler: corev1.Handler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/ready",
											Port: intstr.IntOrString{IntVal: 9901},
										},
									},
									InitialDelaySeconds: 15,
									TimeoutSeconds:      1,
									PeriodSeconds:       5,
									SuccessThreshold:    1,
									FailureThreshold:    1,
								},
							},
						},
						RestartPolicy:                 corev1.RestartPolicyAlways,
						TerminationGracePeriodSeconds: pointer.Int64Ptr(corev1.DefaultTerminationGracePeriodSeconds),
						DNSPolicy:                     corev1.DNSClusterFirst,
						ServiceAccountName:            "default",
						DeprecatedServiceAccount:      "default",
						SecurityContext:               &corev1.PodSecurityContext{},
						SchedulerName:                 corev1.DefaultSchedulerName,
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
	}
}
