package generators

import (
	"testing"
	"time"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
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
		{"Generates a Deployment",
			GeneratorOptions{
				InstanceName:                      "test",
				Namespace:                         "default",
				RootCertificateNamePrefix:         "ca-cert",
				RootCertificateCommonNamePrefix:   "test",
				RootCertificateDuration:           time.Duration(10), // 3 years
				ServerCertificateNamePrefix:       "server-cert",
				ServerCertificateCommonNamePrefix: "test",
				ServerCertificateDuration:         time.Duration(10), // 90 days,
				ClientCertificateDuration:         time.Duration(10),
				XdsServerPort:                     1000,
				MetricsServerPort:                 1001,
				ServiceType:                       operatorv1alpha1.ClusterIPType,
				DeploymentImage:                   "test:latest",
				DeploymentResources:               corev1.ResourceRequirements{},
				Debug:                             true,
			},
			args{hash: "hash"},
			&appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: appsv1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "marin3r-test",
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "marin3r",
						"app.kubernetes.io/managed-by": "marin3r-operator",
						"app.kubernetes.io/component":  "discovery-service",
						"app.kubernetes.io/instance":   "test",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: pointer.Int32Ptr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/name":       "marin3r",
							"app.kubernetes.io/managed-by": "marin3r-operator",
							"app.kubernetes.io/component":  "discovery-service",
							"app.kubernetes.io/instance":   "test",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							CreationTimestamp: metav1.Time{},
							Labels: map[string]string{
								"app.kubernetes.io/name":                                 "marin3r",
								"app.kubernetes.io/managed-by":                           "marin3r-operator",
								"app.kubernetes.io/component":                            "discovery-service",
								"app.kubernetes.io/instance":                             "test",
								operatorv1alpha1.DiscoveryServiceCertificateHashLabelKey: "hash",
							}},
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{
									Name: "server-cert",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName:  "server-cert-test",
											DefaultMode: pointer.Int32Ptr(420),
										},
									},
								},
								{
									Name: "ca-cert",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName:  "ca-cert-test",
											DefaultMode: pointer.Int32Ptr(420),
										},
									},
								},
							},
							Containers: []corev1.Container{
								{
									Name:  "marin3r",
									Image: "test:latest",
									Args: []string{
										"discovery-service",
										"--server-certificate-path=/etc/marin3r/tls/server",
										"--ca-certificate-path=/etc/marin3r/tls/ca",
										"--xdss-port=1000",
										"--metrics-bind-address=:1001",
										"--debug",
									},
									Ports: []corev1.ContainerPort{
										{
											Name:          "discovery",
											ContainerPort: int32(1000),
											Protocol:      corev1.ProtocolTCP,
										},
										{
											Name:          "metrics",
											ContainerPort: int32(1001),
											Protocol:      corev1.ProtocolTCP,
										},
									},
									Env: []corev1.EnvVar{
										{Name: "WATCH_NAMESPACE", Value: "default"},
										{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{
											FieldRef: &corev1.ObjectFieldSelector{
												APIVersion: corev1.SchemeGroupVersion.Version,
												FieldPath:  "metadata.name",
											},
										}},
									},
									Resources: corev1.ResourceRequirements{},
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
							ServiceAccountName:            "marin3r-test",
							DeprecatedServiceAccount:      "marin3r-test",
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
