package v1alpha1

import (
	"fmt"
	"testing"
	"time"

	"github.com/3scale/marin3r/pkg/version"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestDiscoveryService_Resources(t *testing.T) {
	explicitlySetResources := &v1.ResourceRequirements{
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
		{"With explicitly set Resources",
			func() *DiscoveryService {
				return &DiscoveryService{
					Spec: DiscoveryServiceSpec{
						Resources: explicitlySetResources,
					},
				}
			},
			*explicitlySetResources,
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

func TestDiscoveryService_GetRootCertificateAuthorityOptions(t *testing.T) {
	explicitlySet := &CertificateOptions{
		SecretName: "test",
		Duration: metav1.Duration{
			Duration: func() time.Duration {
				d, _ := time.ParseDuration("1h")
				return d
			}(),
		},
	}

	cases := []struct {
		testName                string
		discoveryServiceFactory func() *DiscoveryService
		expectedResult          *CertificateOptions
	}{
		{"With default options",
			func() *DiscoveryService {
				return &DiscoveryService{}
			},
			(&DiscoveryService{}).defaultRootCertificateAuthorityOptions(),
		},
		{"With explicitly set options",
			func() *DiscoveryService {
				return &DiscoveryService{
					Spec: DiscoveryServiceSpec{
						PKIConfig: &PKIConfig{
							RootCertificateAuthority: explicitlySet,
						},
					},
				}
			},
			explicitlySet,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.discoveryServiceFactory().GetRootCertificateAuthorityOptions()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestDiscoveryService_GetServerCertificateOptions(t *testing.T) {
	explicitlySet := &CertificateOptions{
		SecretName: "test",
		Duration: metav1.Duration{
			Duration: func() time.Duration {
				d, _ := time.ParseDuration("1h")
				return d
			}(),
		},
	}

	cases := []struct {
		testName                string
		discoveryServiceFactory func() *DiscoveryService
		expectedResult          *CertificateOptions
	}{
		{"With default options",
			func() *DiscoveryService {
				return &DiscoveryService{}
			},
			(&DiscoveryService{}).defaultServerCertificateOptions(),
		},
		{"With explicitly set options",
			func() *DiscoveryService {
				return &DiscoveryService{
					Spec: DiscoveryServiceSpec{
						PKIConfig: &PKIConfig{
							ServerCertificate: explicitlySet,
						},
					},
				}
			},
			explicitlySet,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.discoveryServiceFactory().GetServerCertificateOptions()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestDiscoveryService_GetXdsServerPort(t *testing.T) {
	cases := []struct {
		testName                string
		discoveryServiceFactory func() *DiscoveryService
		expectedResult          uint32
	}{
		{"With default",
			func() *DiscoveryService {
				return &DiscoveryService{}
			},
			DefaultXdsServerPort,
		},
		{"With explicitly set value",
			func() *DiscoveryService {
				return &DiscoveryService{
					Spec: DiscoveryServiceSpec{
						XdsServerPort: func() *uint32 { var u uint32 = 1000; return &u }(),
					},
				}
			},
			1000,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.discoveryServiceFactory().GetXdsServerPort()
			if tc.expectedResult != receivedResult {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestDiscoveryService_GetMetricsPort(t *testing.T) {
	cases := []struct {
		testName                string
		discoveryServiceFactory func() *DiscoveryService
		expectedResult          uint32
	}{
		{"With default",
			func() *DiscoveryService {
				return &DiscoveryService{}
			},
			DefaultMetricsPort,
		},
		{"With explicitly set value",
			func() *DiscoveryService {
				return &DiscoveryService{
					Spec: DiscoveryServiceSpec{
						MetricsPort: func() *uint32 { var u uint32 = 1000; return &u }(),
					},
				}
			},
			1000,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.discoveryServiceFactory().GetMetricsPort()
			if tc.expectedResult != receivedResult {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestDiscoveryService_GetServiceConfig(t *testing.T) {
	explicitlySet := &ServiceConfig{
		Name: "my-service",
		Type: HeadlessType,
	}

	cases := []struct {
		testName                string
		discoveryServiceFactory func() *DiscoveryService
		expectedResult          *ServiceConfig
	}{
		{"With default options",
			func() *DiscoveryService {
				return &DiscoveryService{}
			},
			(&DiscoveryService{}).defaultServiceConfig(),
		},
		{"With explicitly set options",
			func() *DiscoveryService {
				return &DiscoveryService{
					Spec: DiscoveryServiceSpec{
						ServiceConfig: explicitlySet,
					}}
			},
			explicitlySet,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.discoveryServiceFactory().GetServiceConfig()
			if !equality.Semantic.DeepEqual(tc.expectedResult, receivedResult) {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestDiscoveryService_GetImage(t *testing.T) {
	cases := []struct {
		testName                string
		discoveryServiceFactory func() *DiscoveryService
		expectedResult          string
	}{
		{"With default",
			func() *DiscoveryService {
				return &DiscoveryService{}
			},
			fmt.Sprintf("%s:%s", DefaultImageRegistry, version.Current()),
		},
		{"With explicitly set value",
			func() *DiscoveryService {
				return &DiscoveryService{
					Spec: DiscoveryServiceSpec{
						Image: pointer.StringPtr("image:test"),
					},
				}
			},
			"image:test",
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.discoveryServiceFactory().GetImage()
			if tc.expectedResult != receivedResult {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}

func TestDiscoveryService_Debug(t *testing.T) {
	cases := []struct {
		testName                string
		discoveryServiceFactory func() *DiscoveryService
		expectedResult          bool
	}{
		{"With default",
			func() *DiscoveryService {
				return &DiscoveryService{}
			},
			false,
		},
		{"With explicitly set value",
			func() *DiscoveryService {
				return &DiscoveryService{
					Spec: DiscoveryServiceSpec{
						Debug: pointer.BoolPtr(true),
					},
				}
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(subT *testing.T) {
			receivedResult := tc.discoveryServiceFactory().Debug()
			if tc.expectedResult != receivedResult {
				subT.Errorf("Expected result differs: Expected: %v, Received: %v", tc.expectedResult, receivedResult)
			}
		})
	}
}
