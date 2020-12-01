package controllers

import (
	"fmt"
	"hash/fnv"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator.marin3r/v1alpha1"
	common "github.com/3scale/marin3r/pkg/common"
	"github.com/3scale/marin3r/pkg/reconcilers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"
)

const (
	appLabelKey string = "app"
)

func deploymentGeneratorFn(ds *operatorv1alpha1.DiscoveryService, secret *corev1.Secret) reconcilers.DeploymentGeneratorFn {

	return func() *appsv1.Deployment {

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      OwnedObjectName(ds),
				Namespace: OwnedObjectNamespace(ds),
				Labels:    Labels(ds),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: pointer.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: Labels(ds),
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{},
						Labels:            Labels(ds),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "server-cert",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName:  getServerCertName(ds),
										DefaultMode: pointer.Int32Ptr(420),
									},
								},
							},
							{
								Name: "ca-cert",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName:  getCACertName(ds),
										DefaultMode: pointer.Int32Ptr(420),
									},
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name:  "marin3r",
								Image: ds.GetImage(),
								Args: []string{
									"--discovery-service",
									"--server-certificate-path=/etc/marin3r/tls/server",
									"--ca-certificate-path=/etc/marin3r/tls/ca",
									func() string { return fmt.Sprintf("--xdss-port=%v", ds.GetXdsServerPort()) }(),
									func() string { return fmt.Sprintf("--webhook-port=%v", ds.GetWebhookPort()) }(),
									func() string { return fmt.Sprintf("--metrics-addr=:%v", ds.GetMetricsPort()) }(),
								},
								Ports: []corev1.ContainerPort{
									{
										Name:          "discovery",
										ContainerPort: int32(ds.GetXdsServerPort()),
										Protocol:      corev1.ProtocolTCP,
									},
									{
										Name:          "webhook",
										ContainerPort: int32(ds.GetWebhookPort()),
										Protocol:      corev1.ProtocolTCP,
									},
									{
										Name:          "metrics",
										ContainerPort: int32(ds.GetMetricsPort()),
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: []corev1.EnvVar{
									{Name: "WATCH_NAMESPACE", Value: ""},
									{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: corev1.SchemeGroupVersion.Version,
											FieldPath:  "metadata.name",
										},
									}},
								},
								Resources: ds.Resources(),
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
						ServiceAccountName:            OwnedObjectName(ds),
						DeprecatedServiceAccount:      OwnedObjectName(ds),
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

		if ds.Debug() {
			dep.Spec.Template.Spec.Containers[0].Args = append(dep.Spec.Template.Spec.Containers[0].Args, "--debug")
		}

		// Set a label with the server certificate hash
		dep.Spec.Template.ObjectMeta.Labels[operatorv1alpha1.DiscoveryServiceCertificateHashLabelKey] = certificateHash(secret.Data)

		return dep
	}
}

func certificateHash(data map[string][]byte) string {
	resourcesHasher := fnv.New32a()
	common.DeepHashObject(resourcesHasher, data)
	return rand.SafeEncodeString(fmt.Sprint(resourcesHasher.Sum32()))
}
