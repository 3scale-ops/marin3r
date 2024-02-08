package generators

import (
	"testing"
	"time"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGeneratorOptions_RootCertificationAuthority(t *testing.T) {
	type args struct {
		hash string
	}
	tests := []struct {
		name string
		opts GeneratorOptions
		args args
		want *operatorv1alpha1.DiscoveryServiceCertificate
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
					IsCA:       pointer.New(true),
					ValidFor:   int64(10),
					Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
						SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
					},
					SecretRef: corev1.SecretReference{
						Name:      "ca-cert-test",
						Namespace: "default",
					},
					CertificateRenewalConfig: &operatorv1alpha1.CertificateRenewalConfig{
						Enabled: true,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.opts.RootCertificationAuthority(), tt.want); len(diff) > 0 {
				t.Errorf("GeneratorOptions.RootCertificationAuthority() DIFF:\n %v", diff)
			}
		})
	}
}
