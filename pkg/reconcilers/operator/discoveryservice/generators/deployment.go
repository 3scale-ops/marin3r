package generators

import (
	"fmt"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/reconcilers/lockedresources"
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
							labels[operatorv1alpha1.DiscoveryServiceCertificateHashLabelKey] = hash
							return
						}(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "server-cert",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName:  cfg.serverCertName(),
										DefaultMode: pointer.Int32Ptr(420),
									},
								},
							},
							{
								Name: "ca-cert",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName:  cfg.rootCertName(),
										DefaultMode: pointer.Int32Ptr(420),
									},
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name:  "marin3r",
								Image: cfg.DeploymentImage,
								Args: func() (args []string) {
									args = []string{
										"discovery-service",
										"--server-certificate-path=/etc/marin3r/tls/server",
										"--ca-certificate-path=/etc/marin3r/tls/ca",
										func() string { return fmt.Sprintf("--xdss-port=%v", cfg.XdsServerPort) }(),
										func() string { return fmt.Sprintf("--metrics-bind-address=:%v", cfg.MetricsServerPort) }(),
									}
									if cfg.Debug {
										args = append(args, "--debug")
									}
									return
								}(),
								Ports: []corev1.ContainerPort{
									{
										Name:          "discovery",
										ContainerPort: int32(cfg.XdsServerPort),
										Protocol:      corev1.ProtocolTCP,
									},
									{
										Name:          "metrics",
										ContainerPort: int32(cfg.MetricsServerPort),
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: []corev1.EnvVar{
									{Name: "WATCH_NAMESPACE", Value: cfg.Namespace},
									{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: corev1.SchemeGroupVersion.Version,
											FieldPath:  "metadata.name",
										},
									}},
								},
								Resources: cfg.DeploymentResources,
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "server-cert",
										ReadOnly:  true,
										MountPath: "/etc/marin3r/tls/server/",
									},
									{
										Name:      "ca-cert",
										ReadOnly:  true,
										MountPath: "/etc/marin3r/tls/ca/",
									},
								},
								TerminationMessagePath:   corev1.TerminationMessagePathDefault,
								TerminationMessagePolicy: corev1.TerminationMessageReadFile,
								ImagePullPolicy:          corev1.PullIfNotPresent,
							},
						},
						RestartPolicy:                 corev1.RestartPolicyAlways,
						TerminationGracePeriodSeconds: pointer.Int64Ptr(corev1.DefaultTerminationGracePeriodSeconds),
						DNSPolicy:                     corev1.DNSClusterFirst,
						ServiceAccountName:            cfg.resourceName(),
						DeprecatedServiceAccount:      cfg.resourceName(),
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
