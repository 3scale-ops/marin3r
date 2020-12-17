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

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/utils/pointer"
)

func TestDiscoveryServiceCertificate_IsServerCertificate(t *testing.T) {
	cases := []struct {
		testName                           string
		discoveryServiceCertificateFactory func() *DiscoveryServiceCertificate
		expectedResult                     bool
	}{
		{"With default options",
			func() *DiscoveryServiceCertificate {
				return &DiscoveryServiceCertificate{}
			},
			false,
		},
		{"With explicitly set options",
			func() *DiscoveryServiceCertificate {
				return &DiscoveryServiceCertificate{
					Spec: DiscoveryServiceCertificateSpec{
						IsServerCertificate: pointer.BoolPtr(true),
					},
				}
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.discoveryServiceCertificateFactory().IsServerCertificate()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestDiscoveryServiceCertificate_IsCA(t *testing.T) {
	cases := []struct {
		testName                           string
		discoveryServiceCertificateFactory func() *DiscoveryServiceCertificate
		expectedResult                     bool
	}{
		{"With default options",
			func() *DiscoveryServiceCertificate {
				return &DiscoveryServiceCertificate{}
			},
			false,
		},
		{"With explicitly set options",
			func() *DiscoveryServiceCertificate {
				return &DiscoveryServiceCertificate{
					Spec: DiscoveryServiceCertificateSpec{
						IsCA: pointer.BoolPtr(true),
					},
				}
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.discoveryServiceCertificateFactory().IsCA()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestDiscoveryServiceCertificate_GetHosts(t *testing.T) {
	cases := []struct {
		testName                           string
		discoveryServiceCertificateFactory func() *DiscoveryServiceCertificate
		expectedResult                     []string
	}{
		{"With default options",
			func() *DiscoveryServiceCertificate {
				return &DiscoveryServiceCertificate{Spec: DiscoveryServiceCertificateSpec{CommonName: "test"}}
			},
			[]string{"test"},
		},
		{"With explicitly set options",
			func() *DiscoveryServiceCertificate {
				return &DiscoveryServiceCertificate{
					Spec: DiscoveryServiceCertificateSpec{
						Hosts: []string{"host"},
					},
				}
			},
			[]string{"host"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.discoveryServiceCertificateFactory().GetHosts()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestDiscoveryServiceCertificate_GetCertificateRenewalConfig(t *testing.T) {
	cases := []struct {
		testName                           string
		discoveryServiceCertificateFactory func() *DiscoveryServiceCertificate
		expectedResult                     CertificateRenewalConfig
	}{
		{"With default options",
			func() *DiscoveryServiceCertificate {
				return &DiscoveryServiceCertificate{}
			},
			(&DiscoveryServiceCertificate{}).defaultCertificateRenewalConfig(),
		},
		{"With explicitly set options",
			func() *DiscoveryServiceCertificate {
				return &DiscoveryServiceCertificate{
					Spec: DiscoveryServiceCertificateSpec{
						CertificateRenewalConfig: &CertificateRenewalConfig{Enabled: false},
					},
				}
			},
			CertificateRenewalConfig{Enabled: false},
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.discoveryServiceCertificateFactory().GetCertificateRenewalConfig()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestDiscoveryServiceCertificateStatus_IsReady(t *testing.T) {
	cases := []struct {
		testName                           string
		discoveryServiceCertificateFactory func() *DiscoveryServiceCertificate
		expectedResult                     bool
	}{
		{"Returns false if unset",
			func() *DiscoveryServiceCertificate {
				return &DiscoveryServiceCertificate{}
			},
			false,
		},
		{"Returns value in status if set",
			func() *DiscoveryServiceCertificate {
				return &DiscoveryServiceCertificate{
					Status: DiscoveryServiceCertificateStatus{
						Ready: pointer.BoolPtr(true),
					},
				}
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.discoveryServiceCertificateFactory().Status.IsReady()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}
