/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"reflect"
	"testing"
	"time"

	defaults "github.com/3scale-ops/marin3r/pkg/envoy/container/defaults"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func TestEnvoyDeployment_Image(t *testing.T) {
	cases := []struct {
		testName               string
		envoyDeploymentFactory func() *EnvoyDeployment
		expectedResult         string
	}{
		{"With default options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{}
			},
			defaults.Image,
		},
		{"With explicitly set options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{
					Spec: EnvoyDeploymentSpec{Image: pointer.StringPtr("image:test")},
				}
			},
			"image:test",
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyDeploymentFactory().Image()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoyDeployment_Resources(t *testing.T) {
	cases := []struct {
		testName               string
		envoyDeploymentFactory func() *EnvoyDeployment
		expectedResult         corev1.ResourceRequirements
	}{
		{"With default options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{}
			},
			corev1.ResourceRequirements{},
		},
		{"With explicitly set options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{
					Spec: EnvoyDeploymentSpec{
						Resources: &corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("200m"),
								corev1.ResourceMemory: resource.MustParse("200Mi"),
							}},
					},
				}
			},
			corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyDeploymentFactory().Resources()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoyDeployment_ClientCertificateDuration(t *testing.T) {
	cases := []struct {
		testName               string
		envoyDeploymentFactory func() *EnvoyDeployment
		expectedResult         time.Duration
	}{
		{"With default options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{}
			},
			func() time.Duration { d, _ := time.ParseDuration(ClientCertificateDefaultDuration); return d }(),
		},
		{"With explicitly set options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{
					Spec: EnvoyDeploymentSpec{
						ClientCertificateDuration: &metav1.Duration{
							Duration: func() time.Duration { d, _ := time.ParseDuration("20m"); return d }(),
						},
					},
				}
			},
			func() time.Duration { d, _ := time.ParseDuration("20m"); return d }(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyDeploymentFactory().ClientCertificateDuration()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoyDeployment_AdminPort(t *testing.T) {
	cases := []struct {
		testName               string
		envoyDeploymentFactory func() *EnvoyDeployment
		expectedResult         uint32
	}{
		{"With default options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{}
			},
			defaults.EnvoyAdminPort,
		},
		{"With explicitly set options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{
					Spec: EnvoyDeploymentSpec{AdminPort: func() *uint32 { var d uint32 = 1000; return &d }()},
				}
			},
			1000,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyDeploymentFactory().AdminPort()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoyDeployment_AdminAccessLogPath(t *testing.T) {
	cases := []struct {
		testName               string
		envoyDeploymentFactory func() *EnvoyDeployment
		expectedResult         string
	}{
		{"With default options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{}
			},
			defaults.EnvoyAdminAccessLogPath,
		},
		{"With explicitly set options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{
					Spec: EnvoyDeploymentSpec{AdminAccessLogPath: pointer.StringPtr("/my/log/file")},
				}
			},
			"/my/log/file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyDeploymentFactory().AdminAccessLogPath()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoyDeployment_Replicas(t *testing.T) {
	cases := []struct {
		testName               string
		envoyDeploymentFactory func() *EnvoyDeployment
		expectedResult         ReplicasSpec
	}{
		{"With default options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{}
			},
			ReplicasSpec{Static: pointer.Int32Ptr(DefaultReplicas)},
		},
		{"With explicitly set options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{
					Spec: EnvoyDeploymentSpec{Replicas: &ReplicasSpec{
						Dynamic: &DynamicReplicasSpec{},
					}},
				}
			},
			ReplicasSpec{
				Dynamic: &DynamicReplicasSpec{},
			},
		},
		{"Static replicas takes precedence",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{
					Spec: EnvoyDeploymentSpec{Replicas: &ReplicasSpec{
						Static:  pointer.Int32Ptr(3),
						Dynamic: &DynamicReplicasSpec{},
					}},
				}
			},
			ReplicasSpec{
				Static: pointer.Int32Ptr(3),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyDeploymentFactory().Replicas()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoyDeployment_LivenessProbe(t *testing.T) {
	cases := []struct {
		testName               string
		envoyDeploymentFactory func() *EnvoyDeployment
		expectedResult         ProbeSpec
	}{
		{"With default options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{}
			},
			defaultLivenessProbe,
		},
		{"With explicitly set options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{
					Spec: EnvoyDeploymentSpec{LivenessProbe: &ProbeSpec{
						InitialDelaySeconds: 1,
						TimeoutSeconds:      1,
						PeriodSeconds:       1,
						SuccessThreshold:    1,
						FailureThreshold:    1,
					}},
				}
			},
			ProbeSpec{
				InitialDelaySeconds: 1,
				TimeoutSeconds:      1,
				PeriodSeconds:       1,
				SuccessThreshold:    1,
				FailureThreshold:    1,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyDeploymentFactory().LivenessProbe()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoyDeployment_ReadinessProbe(t *testing.T) {
	cases := []struct {
		testName               string
		envoyDeploymentFactory func() *EnvoyDeployment
		expectedResult         ProbeSpec
	}{
		{"With default options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{}
			},
			defaultReadinessProbe,
		},
		{"With explicitly set options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{
					Spec: EnvoyDeploymentSpec{ReadinessProbe: &ProbeSpec{
						InitialDelaySeconds: 1,
						TimeoutSeconds:      1,
						PeriodSeconds:       1,
						SuccessThreshold:    1,
						FailureThreshold:    1,
					}},
				}
			},
			ProbeSpec{
				InitialDelaySeconds: 1,
				TimeoutSeconds:      1,
				PeriodSeconds:       1,
				SuccessThreshold:    1,
				FailureThreshold:    1,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyDeploymentFactory().ReadinessProbe()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoyDeployment_PodAffinity(t *testing.T) {
	cases := []struct {
		testName               string
		envoyDeploymentFactory func() *EnvoyDeployment
		expectedResult         *corev1.Affinity
	}{
		{"Returns value",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{Spec: EnvoyDeploymentSpec{Affinity: &corev1.Affinity{}}}
			},
			&corev1.Affinity{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyDeploymentFactory().Affinity()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoyDeployment_PodDisruptionBudget(t *testing.T) {
	cases := []struct {
		testName               string
		envoyDeploymentFactory func() *EnvoyDeployment
		expectedResult         PodDisruptionBudgetSpec
	}{
		{"With default options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{}
			},
			defaultPodDisruptionBudget,
		},
		{"With explicitly set options",
			func() *EnvoyDeployment {
				return &EnvoyDeployment{
					Spec: EnvoyDeploymentSpec{
						PodDisruptionBudget: &PodDisruptionBudgetSpec{
							MinAvailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 3},
						},
					},
				}
			},
			PodDisruptionBudgetSpec{
				MinAvailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 3},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyDeploymentFactory().PodDisruptionBudget()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestInitManager_GetImage(t *testing.T) {
	type fields struct {
		Image *string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "returns default",
			fields: fields{Image: nil},
			want:   defaults.InitMgrImage(),
		},
		{
			name:   "returns value",
			fields: fields{Image: pointer.StringPtr("test")},
			want:   "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := &InitManager{
				Image: tt.fields.Image,
			}
			if got := im.GetImage(); got != tt.want {
				t.Errorf("InitManager.GetImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShutdownManager_GetDrainTime(t *testing.T) {
	type fields struct {
		Image         *string
		ServerPort    *uint32
		DrainTime     *int64
		DrainStrategy *defaults.DrainStrategy
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name: "Returns value",
			fields: fields{
				DrainTime: pointer.Int64(100),
			},
			want: 100,
		},
		{
			name:   "Returns default",
			fields: fields{},
			want:   300,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &ShutdownManager{
				Image:         tt.fields.Image,
				ServerPort:    tt.fields.ServerPort,
				DrainTime:     tt.fields.DrainTime,
				DrainStrategy: tt.fields.DrainStrategy,
			}
			if got := sm.GetDrainTime(); got != tt.want {
				t.Errorf("ShutdownManager.GetDrainTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShutdownManager_GetDrainStrategy(t *testing.T) {
	type fields struct {
		Image         *string
		ServerPort    *uint32
		DrainTime     *int64
		DrainStrategy *defaults.DrainStrategy
	}
	tests := []struct {
		name   string
		fields fields
		want   defaults.DrainStrategy
	}{
		{
			name: "Returns value",
			fields: fields{
				DrainStrategy: func() *defaults.DrainStrategy { s := defaults.DrainStrategyImmediate; return &s }(),
			},
			want: defaults.DrainStrategyImmediate,
		},
		{
			name:   "Returns default",
			fields: fields{},
			want:   defaults.DrainStrategyGradual,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &ShutdownManager{
				Image:         tt.fields.Image,
				ServerPort:    tt.fields.ServerPort,
				DrainTime:     tt.fields.DrainTime,
				DrainStrategy: tt.fields.DrainStrategy,
			}
			if got := sm.GetDrainStrategy(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ShutdownManager.GetDrainStrategy() = %v, want %v", got, tt.want)
			}
		})
	}
}
