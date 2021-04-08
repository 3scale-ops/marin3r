package generators

import (
	"github.com/3scale-ops/marin3r/pkg/reconcilers/lockedresources"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (cfg *GeneratorOptions) HPA() lockedresources.GeneratorFunction {

	return func() client.Object {

		return &autoscalingv2beta2.HorizontalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				Kind:       "HorizontalPodAutoscaler",
				APIVersion: autoscalingv2beta2.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      cfg.resourceName(),
				Namespace: cfg.Namespace,
				Labels:    cfg.labels(),
			},
			Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
					APIVersion: appsv1.SchemeGroupVersion.String(),
					Kind:       "Deployment",
					Name:       cfg.resourceName(),
				},
				MinReplicas: cfg.Replicas.Dynamic.MinReplicas,
				MaxReplicas: cfg.Replicas.Dynamic.MaxReplicas,
				Metrics:     cfg.Replicas.Dynamic.Metrics,
				Behavior:    cfg.Replicas.Dynamic.Behavior,
			},
			Status: autoscalingv2beta2.HorizontalPodAutoscalerStatus{
				Conditions: []autoscalingv2beta2.HorizontalPodAutoscalerCondition{},
			},
		}
	}
}
