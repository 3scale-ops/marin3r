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

	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	"github.com/3scale-ops/marin3r/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

func TestEnvoyConfig_GetEnvoyAPIVersion(t *testing.T) {
	cases := []struct {
		testName                   string
		envoyConfigRevisionFactory func() *EnvoyConfig
		expectedResult             envoy.APIVersion
	}{
		{"With default",
			func() *EnvoyConfig {
				return &EnvoyConfig{}
			},
			envoy.APIv3,
		},
		{"With explicitly set value",
			func() *EnvoyConfig {
				return &EnvoyConfig{
					Spec: EnvoyConfigSpec{
						EnvoyAPI: pointer.StringPtr("v3"),
					},
				}
			},
			envoy.APIv3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyConfigRevisionFactory().GetEnvoyAPIVersion()
			if receivedResult.String() != tc.expectedResult.String() {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoyConfig_GetSerialization(t *testing.T) {
	cases := []struct {
		testName                   string
		envoyConfigRevisionFactory func() *EnvoyConfig
		expectedResult             envoy_serializer.Serialization
	}{
		{"With default",
			func() *EnvoyConfig {
				return &EnvoyConfig{}
			},
			envoy_serializer.JSON,
		},
		{"With explicitly set value",
			func() *EnvoyConfig {
				return &EnvoyConfig{
					Spec: EnvoyConfigSpec{
						Serialization: pointer.StringPtr("yaml"),
					},
				}
			},
			envoy_serializer.YAML,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyConfigRevisionFactory().GetSerialization()
			if string(receivedResult) != string(tc.expectedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoyConfig_GetEnvoyResourcesVersion(t *testing.T) {
	cases := []struct {
		testName                   string
		envoyConfigRevisionFactory func() *EnvoyConfig
		expectedResult             string
	}{
		{"With default",
			func() *EnvoyConfig {
				return &EnvoyConfig{
					Spec: EnvoyConfigSpec{
						EnvoyResources: &EnvoyResources{},
					},
				}
			},
			util.Hash(&EnvoyResources{}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyConfigRevisionFactory().GetEnvoyResourcesVersion()
			if receivedResult != tc.expectedResult {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoySecretResource_Validate(t *testing.T) {
	type fields struct {
		Name string
		Ref  *corev1.SecretReference
	}
	type args struct {
		namespace string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "Ref.Name cannot be empty in Ref",
			fields: fields{Name: "aaa", Ref: &corev1.SecretReference{Name: ""}},
			args: args{
				namespace: "test",
			},
			wantErr: true,
		},
		{
			name:   "Ref.Namespace can be empty in Ref",
			fields: fields{Name: "aaa", Ref: &corev1.SecretReference{Name: "bbb"}},
			args: args{
				namespace: "test",
			},
			wantErr: false,
		},
		{
			name:   "Ref.Namespace can be set to the resource namespace",
			fields: fields{Name: "aaa", Ref: &corev1.SecretReference{Name: "bbb", Namespace: "test"}},
			args: args{
				namespace: "test",
			},
			wantErr: false,
		},
		{
			name:   "Fail for any other value of Ref.Namespace",
			fields: fields{Name: "aaa", Ref: &corev1.SecretReference{Name: "bbb", Namespace: "other"}},
			args: args{
				namespace: "test",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			esr := &EnvoySecretResource{
				Name: tt.fields.Name,
				Ref:  tt.fields.Ref,
			}
			if err := esr.Validate(tt.args.namespace); (err != nil) != tt.wantErr {
				t.Errorf("EnvoySecretResource.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvoySecretResource_GetSecretKey(t *testing.T) {
	type fields struct {
		Name string
		Ref  *corev1.SecretReference
	}
	type args struct {
		namespace string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   types.NamespacedName
	}{
		{
			name: "Returns a key with same Name when Ref unset",
			fields: fields{
				Name: "test",
				Ref:  nil,
			},
			args: args{namespace: "ns"},
			want: types.NamespacedName{Name: "test", Namespace: "ns"},
		},
		{
			name: "Returns a key matchin Ref if Ref is set",
			fields: fields{
				Name: "test",
				Ref: &corev1.SecretReference{
					Name:      "aaa",
					Namespace: "bbb",
				},
			},
			args: args{namespace: "bbb"},
			want: types.NamespacedName{Name: "aaa", Namespace: "bbb"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			esr := &EnvoySecretResource{
				Name: tt.fields.Name,
				Ref:  tt.fields.Ref,
			}
			if got := esr.GetSecretKey(tt.args.namespace); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EnvoySecretResource.GetSecretKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
