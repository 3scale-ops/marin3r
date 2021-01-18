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
	"testing"

	"github.com/3scale/marin3r/pkg/envoy"
	envoy_serializer "github.com/3scale/marin3r/pkg/envoy/serializer"
	"github.com/3scale/marin3r/pkg/util"
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
			envoy.APIv2,
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
