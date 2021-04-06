package generators

import (
	"fmt"
	"testing"
	"time"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	defaults "github.com/3scale-ops/marin3r/pkg/envoy/bootstrap/defaults"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGeneratorOptions_EnvoyBootstrap(t *testing.T) {
	type args struct {
		discoveryServiceName string
	}
	tests := []struct {
		name string
		opts GeneratorOptions
		args args
		want client.Object
	}{
		{
			name: "Generates an EnvoyBootstrap",
			opts: GeneratorOptions{
				InstanceName:              "instance",
				Namespace:                 "default",
				EnvoyAPIVersion:           "v3",
				EnvoyNodeID:               "test",
				EnvoyClusterID:            "test",
				ClientCertificateDuration: time.Duration(20 * time.Second),
				DeploymentImage:           "test:latest",
				DeploymentResources:       corev1.ResourceRequirements{},
				ExposedPorts:              []operatorv1alpha1.ContainerPort{},
			},
			args: args{discoveryServiceName: "discoveryservice"},
			want: &marin3rv1alpha1.EnvoyBootstrap{
				TypeMeta: metav1.TypeMeta{
					Kind:       marin3rv1alpha1.EnvoyBootstrapKind,
					APIVersion: marin3rv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "marin3r-envoy-deployment-instance",
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "marin3r",
						"app.kubernetes.io/managed-by": "marin3r-operator",
						"app.kubernetes.io/component":  "envoy-deployment",
						"app.kubernetes.io/instance":   "instance",
					},
				},
				Spec: marin3rv1alpha1.EnvoyBootstrapSpec{
					DiscoveryService: "discoveryservice",
					ClientCertificate: marin3rv1alpha1.ClientCertificate{
						Directory:  defaults.EnvoyTLSBasePath,
						SecretName: defaults.DeploymentClientCertificate + "-instance",
						Duration: metav1.Duration{
							Duration: time.Duration(20 * time.Second),
						},
					},
					EnvoyStaticConfig: marin3rv1alpha1.EnvoyStaticConfig{
						ConfigMapNameV2:       defaults.DeploymentBootstrapConfigMapV2 + "-instance",
						ConfigMapNameV3:       defaults.DeploymentBootstrapConfigMapV3 + "-instance",
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
			if got := cfg.EnvoyBootstrap(tt.args.discoveryServiceName)(); !equality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("GeneratorOptions.EnvoyBootstrap() = %v, want %v", got, tt.want)
			}
		})
	}
}
