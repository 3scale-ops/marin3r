package generators

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cfg *GeneratorOptions) HPA() *autoscalingv2.HorizontalPodAutoscaler {

	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.resourceName(),
			Namespace: cfg.Namespace,
			Labels:    cfg.labels(),
		},
		Spec: func() autoscalingv2.HorizontalPodAutoscalerSpec {
			if cfg.Replicas.Dynamic == nil {
				return autoscalingv2.HorizontalPodAutoscalerSpec{}
			}
			return autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					APIVersion: appsv1.SchemeGroupVersion.String(),
					Kind:       "Deployment",
					Name:       cfg.resourceName(),
				},
				MinReplicas: cfg.Replicas.Dynamic.MinReplicas,
				MaxReplicas: cfg.Replicas.Dynamic.MaxReplicas,
				Metrics:     cfg.Replicas.Dynamic.Metrics,
				Behavior:    cfg.Replicas.Dynamic.Behavior,
			}
		}(),
	}
}
