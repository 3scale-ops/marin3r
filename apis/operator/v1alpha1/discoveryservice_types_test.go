package v1alpha1

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestDiscoveryServiceResources(t *testing.T) {
	explicitelySetResources := &v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("100m"),
			v1.ResourceMemory: resource.MustParse("200Mi"),
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("200m"),
			v1.ResourceMemory: resource.MustParse("400Mi"),
		},
	}

	cases := []struct {
		testName                string
		discoveryServiceFactory func() *DiscoveryService
		expectedResult          v1.ResourceRequirements
	}{
		{"With default Resources",
			func() *DiscoveryService {
				return &DiscoveryService{}
			},
			v1.ResourceRequirements{},
		},
		{"With explicitely set Resources",
			func() *DiscoveryService {
				return &DiscoveryService{
					Spec: DiscoveryServiceSpec{
						Resources: explicitelySetResources,
					},
				}
			},
			*explicitelySetResources,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.discoveryServiceFactory().Resources()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}
