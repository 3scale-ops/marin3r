package generators

import (
	"testing"
	"time"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGeneratorOptions_Deployment(t *testing.T) {
	type args struct {
		hash string
	}
	tests := []struct {
		name string
		opts GeneratorOptions
		args args
		want *appsv1.Deployment
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
				ProbePort:                         1002,
				ServiceType:                       operatorv1alpha1.ClusterIPType,
				DeploymentImage:                   "test:latest",
				DeploymentResources:               corev1.ResourceRequirements{},
				Debug:                             true,
				PodPriorityClass:                  pointer.New("highest"),
			},
			args{hash: "hash"},
			&appsv1.Deployment{
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
					Replicas: pointer.New(int32(1)),
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
											DefaultMode: pointer.New(int32(420)),
										},
									},
								},
								{
									Name: "ca-cert",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName:  "ca-cert-test",
											DefaultMode: pointer.New(int32(420)),
										},
									},
								},
								{
									Name: "client-cert",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName:  "envoy-sidecar-client-cert",
											DefaultMode: pointer.New(int32(420)),
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
										"--client-certificate-path=/etc/marin3r/tls/client",
										"--xdss-port=1000",
										"--metrics-bind-address=:1001",
										"--health-probe-bind-address=:1002",
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
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path:   "/healthz",
												Port:   intstr.FromInt(1002),
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
												Port:   intstr.FromInt(1002),
												Scheme: corev1.URISchemeHTTP,
											},
										},
										FailureThreshold: 3,
										PeriodSeconds:    10,
										SuccessThreshold: 1,
										TimeoutSeconds:   1,
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
							ServiceAccountName:            "marin3r-test",
							DeprecatedServiceAccount:      "marin3r-test",
							PriorityClassName:             "highest",
						},
					},
					Strategy: appsv1.DeploymentStrategy{
						Type: appsv1.RecreateDeploymentStrategyType,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.opts.Deployment(tt.args.hash)(), tt.want); len(diff) > 0 {
				t.Errorf("GeneratorOptions.Deployment() DIFF:\n %v", diff)
			}
		})
	}
}
