package generators

import (
	"fmt"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (cfg *GeneratorOptions) Deployment(hash string) func() *appsv1.Deployment {

	return func() *appsv1.Deployment {

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cfg.ResourceName(),
				Namespace: cfg.Namespace,
				Labels:    cfg.labels(),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: pointer.New(int32(1)),
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
										SecretName:  cfg.ServerCertName(),
										DefaultMode: pointer.New(int32(420)),
									},
								},
							},
							{
								Name: "ca-cert",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName:  cfg.RootCertName(),
										DefaultMode: pointer.New(int32(420)),
									},
								},
							},
							{
								Name: "client-cert",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName:  cfg.ClientCertName(),
										DefaultMode: pointer.New(int32(420)),
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
										"--client-certificate-path=/etc/marin3r/tls/client",
										fmt.Sprintf("--xdss-port=%v", cfg.XdsServerPort),
										fmt.Sprintf("--metrics-bind-address=:%v", cfg.MetricsServerPort),
										fmt.Sprintf("--health-probe-bind-address=:%v", cfg.ProbePort),
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
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path:   "/healthz",
											Port:   intstr.FromInt(int(cfg.ProbePort)),
											Scheme: corev1.URISchemeHTTP,
										},
									},
									FailureThreshold: 3,
									PeriodSeconds:    10,
									SuccessThreshold: 1,
									TimeoutSeconds:   1,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path:   "/readyz",
											Port:   intstr.FromInt(int(cfg.ProbePort)),
											Scheme: corev1.URISchemeHTTP,
										},
									},
									FailureThreshold: 3,
									PeriodSeconds:    10,
									SuccessThreshold: 1,
									TimeoutSeconds:   1,
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
									{
										Name:      "client-cert",
										ReadOnly:  true,
										MountPath: "/etc/marin3r/tls/client/",
									},
								},
								ImagePullPolicy: corev1.PullIfNotPresent,
							},
						},
						TerminationGracePeriodSeconds: pointer.New(int64(corev1.DefaultTerminationGracePeriodSeconds)),
						ServiceAccountName:            cfg.ResourceName(),
						DeprecatedServiceAccount:      cfg.ResourceName(),
					},
				},
				Strategy: appsv1.DeploymentStrategy{
					Type: appsv1.RecreateDeploymentStrategyType,
				},
			},
		}

		if cfg.PodPriorityClass != nil {
			deployment.Spec.Template.Spec.PriorityClassName = *cfg.PodPriorityClass
		}

		return deployment
	}
}
