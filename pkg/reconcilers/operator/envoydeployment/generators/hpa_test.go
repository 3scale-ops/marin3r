package generators

import (
	"testing"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGeneratorOptions_HPA(t *testing.T) {
	tests := []struct {
		name string
		opts GeneratorOptions
		want client.Object
	}{
		{
			name: "Generate an HPA",
			opts: GeneratorOptions{
				InstanceName: "instance",
				Namespace:    "default",
				Replicas: operatorv1alpha1.ReplicasSpec{
					Dynamic: &operatorv1alpha1.DynamicReplicasSpec{
						MinReplicas: pointer.Int32Ptr(2),
						MaxReplicas: 4,
						Metrics: []autoscalingv2beta2.MetricSpec{
							{
								Type: autoscalingv2beta2.ResourceMetricSourceType,
								Resource: &autoscalingv2beta2.ResourceMetricSource{
									Name: corev1.ResourceCPU,
									Target: autoscalingv2beta2.MetricTarget{
										Type:               autoscalingv2beta2.UtilizationMetricType,
										AverageUtilization: pointer.Int32Ptr(50),
									},
								},
							},
						},
					},
				},
			},
			want: &autoscalingv2beta2.HorizontalPodAutoscaler{
				TypeMeta: metav1.TypeMeta{
					Kind:       "HorizontalPodAutoscaler",
					APIVersion: autoscalingv2beta2.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "marin3r-envoydeployment-instance",
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "marin3r",
						"app.kubernetes.io/managed-by": "marin3r-operator",
						"app.kubernetes.io/component":  "envoy-deployment",
						"app.kubernetes.io/instance":   "instance",
					},
				},
				Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
						APIVersion: appsv1.SchemeGroupVersion.String(),
						Kind:       "Deployment",
						Name:       "marin3r-envoydeployment-instance",
					},
					MinReplicas: pointer.Int32Ptr(2),
					MaxReplicas: 4,
					Metrics: []autoscalingv2beta2.MetricSpec{
						{
							Type: autoscalingv2beta2.ResourceMetricSourceType,
							Resource: &autoscalingv2beta2.ResourceMetricSource{
								Name: corev1.ResourceCPU,
								Target: autoscalingv2beta2.MetricTarget{
									Type:               autoscalingv2beta2.UtilizationMetricType,
									AverageUtilization: pointer.Int32Ptr(50),
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.opts
			if got := cfg.HPA()(); !equality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("GeneratorOptions.HPA() = %v, want %v", got, tt.want)
			}
		})
	}
}
