package generators

import (
	"testing"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGeneratorOptions_HPA(t *testing.T) {
	tests := []struct {
		name string
		opts GeneratorOptions
		want *autoscalingv2.HorizontalPodAutoscaler
	}{
		{
			name: "Generate an HPA",
			opts: GeneratorOptions{
				InstanceName: "instance",
				Namespace:    "default",
				Replicas: operatorv1alpha1.ReplicasSpec{
					Dynamic: &operatorv1alpha1.DynamicReplicasSpec{
						MinReplicas: pointer.New(int32(2)),
						MaxReplicas: 4,
						Metrics: []autoscalingv2.MetricSpec{
							{
								Type: autoscalingv2.ResourceMetricSourceType,
								Resource: &autoscalingv2.ResourceMetricSource{
									Name: corev1.ResourceCPU,
									Target: autoscalingv2.MetricTarget{
										Type:               autoscalingv2.UtilizationMetricType,
										AverageUtilization: pointer.New(int32(50)),
									},
								},
							},
						},
					},
				},
			},
			want: &autoscalingv2.HorizontalPodAutoscaler{
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
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						APIVersion: appsv1.SchemeGroupVersion.String(),
						Kind:       "Deployment",
						Name:       "marin3r-envoydeployment-instance",
					},
					MinReplicas: pointer.New(int32(2)),
					MaxReplicas: 4,
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: autoscalingv2.ResourceMetricSourceType,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: corev1.ResourceCPU,
								Target: autoscalingv2.MetricTarget{
									Type:               autoscalingv2.UtilizationMetricType,
									AverageUtilization: pointer.New(int32(50)),
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
			if diff := cmp.Diff(tt.opts.HPA(), tt.want); len(diff) > 0 {
				t.Errorf("GeneratorOptions.HPA() DIFF:\n %v", diff)
			}
		})
	}
}
