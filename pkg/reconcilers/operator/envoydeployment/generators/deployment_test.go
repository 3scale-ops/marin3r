package generators

import (
	"testing"
	"time"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	defaults "github.com/3scale-ops/marin3r/pkg/envoy/bootstrap/defaults"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGeneratorOptions_Deployment(t *testing.T) {
	type args struct {
		hash string
	}
	tests := []struct {
		name string
		opts GeneratorOptions
		args args
		want client.Object
	}{
		{
			name: "EnvoyDeployment's Deployment generation",
			opts: GeneratorOptions{
				InstanceName:              "instance",
				Namespace:                 "default",
				EnvoyAPIVersion:           "v3",
				EnvoyNodeID:               "test",
				EnvoyClusterID:            "test",
				ClientCertificateDuration: time.Duration(20 * time.Second),
				DeploymentImage:           "test:latest",
				DeploymentResources:       corev1.ResourceRequirements{},
				ExposedPorts:              []operatorv1alpha1.ContainerPort{{Name: "port", Port: 8080}},
			},
			args: args{hash: "hash"},
			want: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: appsv1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "marin3r-envoy-deployment-instance",
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "marin3r",
						"app.kubernetes.io/managed-by": "marin3r-operator",
						"app.kubernetes.io/component":  "envoy-deployment",
						"app.kubernetes.io/instance":   "instance",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: pointer.Int32Ptr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/name":       "marin3r",
							"app.kubernetes.io/managed-by": "marin3r-operator",
							"app.kubernetes.io/component":  "envoy-deployment",
							"app.kubernetes.io/instance":   "instance",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							CreationTimestamp: metav1.Time{},
							Labels: map[string]string{
								"app.kubernetes.io/name":                                    "marin3r",
								"app.kubernetes.io/managed-by":                              "marin3r-operator",
								"app.kubernetes.io/component":                               "envoy-deployment",
								"app.kubernetes.io/instance":                                "instance",
								operatorv1alpha1.EnvoyDeploymentBootstrapConfigHashLabelKey: "hash",
							}},
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{
									Name: defaults.DeploymentTLSVolume,
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: defaults.DeploymentClientCertificate + "-instance",
										},
									},
								},
								{
									Name: defaults.DeploymentConfigVolume,
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: defaults.DeploymentBootstrapConfigMapV3 + "-instance",
											},
										},
									},
								},
							},
							Containers: []corev1.Container{
								{
									Name:    defaults.DeploymentContainerName,
									Image:   "test:latest",
									Command: []string{"envoy"},
									Args: []string{
										"-c",
										"/etc/envoy/bootstrap/config.json",
										"--service-node",
										"test",
										"--service-cluster",
										"test",
									},
									Resources: corev1.ResourceRequirements{},
									Ports: []corev1.ContainerPort{
										{
											Name:          "port",
											ContainerPort: int32(8080),
										},
									},
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
									TerminationMessagePath:   corev1.TerminationMessagePathDefault,
									TerminationMessagePolicy: corev1.TerminationMessageReadFile,
									ImagePullPolicy:          corev1.PullIfNotPresent,
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.opts
			if got := cfg.Deployment(tt.args.hash)(); !equality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("GeneratorOptions.Deployment() = %v, want %v", got, tt.want)
			}
		})
	}
}
