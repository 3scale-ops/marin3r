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

	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
)

func TestEnvoyConfigRevisionStatus_IsPublished(t *testing.T) {
	cases := []struct {
		testName                   string
		envoyConfigRevisionFactory func() *EnvoyConfigRevision
		expectedResult             bool
	}{
		{"With default",
			func() *EnvoyConfigRevision {
				return &EnvoyConfigRevision{}
			},
			false,
		},
		{"With explicitly set value",
			func() *EnvoyConfigRevision {
				return &EnvoyConfigRevision{
					Status: EnvoyConfigRevisionStatus{
						Published: pointer.New(true),
					},
				}
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyConfigRevisionFactory().Status.IsPublished()
			if receivedResult != tc.expectedResult {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoyConfigRevisionStatus_IsTainted(t *testing.T) {
	cases := []struct {
		testName                   string
		envoyConfigRevisionFactory func() *EnvoyConfigRevision
		expectedResult             bool
	}{
		{"With default",
			func() *EnvoyConfigRevision {
				return &EnvoyConfigRevision{}
			},
			false,
		},
		{"With explicitly set value",
			func() *EnvoyConfigRevision {
				return &EnvoyConfigRevision{
					Status: EnvoyConfigRevisionStatus{
						Tainted: pointer.New(true),
					},
				}
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.envoyConfigRevisionFactory().Status.IsTainted()
			if receivedResult != tc.expectedResult {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestEnvoyConfigRevision_GetEnvoyAPIVersion(t *testing.T) {
	cases := []struct {
		testName                   string
		envoyConfigRevisionFactory func() *EnvoyConfigRevision
		expectedResult             envoy.APIVersion
	}{
		{"With default",
			func() *EnvoyConfigRevision {
				return &EnvoyConfigRevision{}
			},
			envoy.APIv3,
		},
		{"With explicitly set value",
			func() *EnvoyConfigRevision {
				return &EnvoyConfigRevision{
					Spec: EnvoyConfigRevisionSpec{
						EnvoyAPI: pointer.New(envoy.APIv3),
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

func TestEnvoyConfigRevision_GetSerialization(t *testing.T) {
	cases := []struct {
		testName                   string
		envoyConfigRevisionFactory func() *EnvoyConfigRevision
		expectedResult             envoy_serializer.Serialization
	}{
		{"With default",
			func() *EnvoyConfigRevision {
				return &EnvoyConfigRevision{}
			},
			envoy_serializer.JSON,
		},
		{"With explicitly set value",
			func() *EnvoyConfigRevision {
				return &EnvoyConfigRevision{
					Spec: EnvoyConfigRevisionSpec{
						Serialization: pointer.New(envoy_serializer.YAML),
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
