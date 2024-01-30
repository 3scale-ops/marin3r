package generators

import (
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (cfg *GeneratorOptions) Service() *corev1.Service {

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.ResourceName(),
			Namespace: cfg.Namespace,
			Labels:    cfg.labels(),
		},
		Spec: corev1.ServiceSpec{
			Type: func() corev1.ServiceType {
				if cfg.ServiceType == operatorv1alpha1.LoadBalancerType {
					return corev1.ServiceTypeLoadBalancer
				}
				return corev1.ServiceTypeClusterIP
			}(),
			ClusterIP: func() string {
				if cfg.ServiceType == operatorv1alpha1.HeadlessType {
					return "None"
				}
				return ""
			}(),
			Selector:        cfg.labels(),
			SessionAffinity: corev1.ServiceAffinityNone,
			Ports: []corev1.ServicePort{
				{
					Name:       "discovery",
					Port:       cfg.XdsServerPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("discovery"),
				},
				{
					Name:       "metrics",
					Port:       cfg.MetricsServerPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("metrics"),
				},
			},
		},
	}
}
