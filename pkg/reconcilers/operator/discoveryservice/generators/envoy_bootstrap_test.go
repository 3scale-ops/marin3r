package generators

import (
	"fmt"
	"testing"
	"time"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	defaults "github.com/3scale-ops/marin3r/pkg/envoy/container/defaults"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGeneratorOptions_EnvoyBootstrap(t *testing.T) {
	type args struct {
		hash string
	}
	tests := []struct {
		name string
		opts GeneratorOptions
		args args
		want client.Object
	}{
		{"Generates an ENvoyBootstrap",
			GeneratorOptions{
				InstanceName:                      "test",
				Namespace:                         "default",
				RootCertificateNamePrefix:         "ca-cert",
				RootCertificateCommonNamePrefix:   "test",
				RootCertificateDuration:           time.Duration(10 * time.Second), // 3 years
				ServerCertificateNamePrefix:       "server-cert",
				ServerCertificateCommonNamePrefix: "test",
				ServerCertificateDuration:         time.Duration(10 * time.Second), // 90 days,
				ClientCertificateDuration:         time.Duration(10 * time.Second),
				XdsServerPort:                     1000,
				MetricsServerPort:                 1001,
				ServiceType:                       operatorv1alpha1.ClusterIPType,
				DeploymentImage:                   "test:latest",
				DeploymentResources:               corev1.ResourceRequirements{},
				Debug:                             true,
			},
			args{hash: "hash"},
			&marin3rv1alpha1.EnvoyBootstrap{
				TypeMeta: metav1.TypeMeta{
					Kind:       marin3rv1alpha1.EnvoyBootstrapKind,
					APIVersion: marin3rv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "marin3r-test",
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "marin3r",
						"app.kubernetes.io/managed-by": "marin3r-operator",
						"app.kubernetes.io/component":  "discovery-service",
						"app.kubernetes.io/instance":   "test",
					},
				},
				Spec: marin3rv1alpha1.EnvoyBootstrapSpec{
					DiscoveryService: "test",
					ClientCertificate: marin3rv1alpha1.ClientCertificate{
						Directory:  defaults.EnvoyTLSBasePath,
						SecretName: defaults.SidecarClientCertificate,
						Duration: metav1.Duration{
							Duration: time.Duration(10 * time.Second),
						},
					},
					EnvoyStaticConfig: marin3rv1alpha1.EnvoyStaticConfig{
						ConfigMapNameV2:       defaults.SidecarBootstrapConfigMapV2,
						ConfigMapNameV3:       defaults.SidecarBootstrapConfigMapV3,
						ConfigFile:            fmt.Sprintf("%s/%s", defaults.EnvoyConfigBasePath, defaults.EnvoyConfigFileName),
						ResourcesDir:          defaults.EnvoyConfigBasePath,
						RtdsLayerResourceName: "runtime",
						AdminBindAddress:      "0.0.0.0:9901",
						AdminAccessLogPath:    "/dev/null",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.opts
			if got := cfg.EnvoyBootstrap()(); !equality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("GeneratorOptions.EnvoyBootstrap() = %v, want %v", got, tt.want)
			}
		})
	}
}
