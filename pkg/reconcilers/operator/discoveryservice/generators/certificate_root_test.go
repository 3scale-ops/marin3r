package generators

import (
	"testing"
	"time"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator.marin3r/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGeneratorOptions_RootCertificationAuthority(t *testing.T) {
	type args struct {
		hash string
	}
	tests := []struct {
		name string
		opts GeneratorOptions
		args args
		want client.Object
	}{
		{"Generates DiscoveryServiceCertificate for the root CA",
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
			&operatorv1alpha1.DiscoveryServiceCertificate{
				TypeMeta: metav1.TypeMeta{
					Kind:       operatorv1alpha1.DiscoveryServiceCertificateKind,
					APIVersion: operatorv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ca-cert-test",
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "marin3r",
						"app.kubernetes.io/managed-by": "marin3r-operator",
						"app.kubernetes.io/component":  "discovery-service",
						"app.kubernetes.io/instance":   "test",
					},
				},
				Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
					CommonName: "test-test",
					IsCA:       pointer.BoolPtr(true),
					ValidFor:   int64(10),
					Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
						SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
					},
					SecretRef: corev1.SecretReference{
						Name:      "ca-cert-test",
						Namespace: "default",
					},
					CertificateRenewalConfig: &operatorv1alpha1.CertificateRenewalConfig{
						Enabled: false,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.opts
			if got := cfg.RootCertificationAuthority()(); !equality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("GeneratorOptions.RootCertificationAuthority() = %v, want %v", got, tt.want)
			}
		})
	}
}
