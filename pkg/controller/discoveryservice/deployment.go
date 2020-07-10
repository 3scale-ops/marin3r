package discoveryservice

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	appLabelKey string = "app"
)

// reconcileDeployment is in charge of keeping the marin3r Deployment object in sync with the spec
func (r *ReconcileDiscoveryService) reconcileDeployment(ctx context.Context) (reconcile.Result, error) {

	r.logger.V(1).Info("Reconciling Deployment")
	existent := &appsv1.Deployment{}
	err := r.client.Get(ctx, types.NamespacedName{Name: r.getName(), Namespace: r.getNamespace()}, existent)

	if err != nil {
		if errors.IsNotFound(err) {
			existent = r.getDeploymentObject()
			if err := controllerutil.SetControllerReference(r.ds, existent, r.scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.client.Create(ctx, existent); err != nil {
				return reconcile.Result{}, err
			}
			r.logger.Info("Created Deployment")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// We just reconcile the spec for the moment
	desired := r.getDeploymentObject()
	// Avoid diff on fields not controlled by this controller
	// NA

	if !equality.Semantic.DeepEqual(existent.Spec, desired.Spec) {
		patch := client.MergeFrom(existent.DeepCopy())
		existent.Spec = desired.Spec
		if err := r.client.Patch(ctx, existent, patch); err != nil {
			return reconcile.Result{}, err
		}
		r.logger.Info("Patched Deployment")
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileDiscoveryService) getDeploymentObject() *appsv1.Deployment {

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getName(),
			Namespace: r.getNamespace(),
			Labels:    map[string]string{appLabelKey: r.getAppLabel()},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					appLabelKey: r.getAppLabel(),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{},
					Labels:            map[string]string{appLabelKey: r.getAppLabel()},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "server-cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  r.getServerCertName(),
									DefaultMode: pointer.Int32Ptr(420),
								},
							},
						},
						{
							// This won't work as the CA is in another namespace
							// when using cert-manager issued certs. We need to
							// sync the CA between namespaces.
							Name: "ca-cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  r.getCACertName(),
									DefaultMode: pointer.Int32Ptr(420),
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "marin3r",
							Image: r.ds.Spec.Image,
							Args: []string{
								"--certificate=/etc/marin3r/tls/server/tls.crt",
								"--private-key=/etc/marin3r/tls/server/tls.key",
								"--ca=/etc/marin3r/tls/ca/tls.crt",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "discovery",
									ContainerPort: 18000,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "webhook",
									ContainerPort: 8443,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "metrics",
									ContainerPort: 8383,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "cr-metrics",
									ContainerPort: 8686,
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
					ServiceAccountName:            r.getName(),
					DeprecatedServiceAccount:      r.getName(),
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

	if r.ds.Spec.Debug {
		dep.Spec.Template.Spec.Containers[0].Args = append(dep.Spec.Template.Spec.Containers[0].Args, "--zap-devel")
	}

	return dep
}
