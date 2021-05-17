package generators

import (
	"testing"
	"time"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGeneratorOptions_ClientCertificate(t *testing.T) {
	tests := []struct {
		name string
		opts GeneratorOptions
		want client.Object
	}{
		{
			name: "Generates DSC resource",
			opts: GeneratorOptions{
				InstanceName:              "instance",
				Namespace:                 "default",
				ClientCertificateName:     "cert",
				ClientCertificateDuration: time.Duration(20 * time.Second),
				SigningCertificateName:    "signing-cert",
			},
			want: &operatorv1alpha1.DiscoveryServiceCertificate{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DiscoveryServiceCertificate",
					APIVersion: operatorv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cert",
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "marin3r",
						"app.kubernetes.io/managed-by": "marin3r-operator",
						"app.kubernetes.io/component":  "envoy-deployment",
						"app.kubernetes.io/instance":   "instance",
					},
				},
				Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
					CommonName: "cert",
					ValidFor:   int64(time.Duration(20 * time.Second).Seconds()),
					Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
						CASigned: &operatorv1alpha1.CASignedConfig{
							SecretRef: corev1.SecretReference{
								Name:      "signing-cert",
								Namespace: "default",
							}},
					},
					SecretRef: corev1.SecretReference{
						Name: "cert",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.ClientCertificate()(); !equality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("GeneratorOptions.Deployment() = %v, want %v", got, tt.want)
			}
		})
	}
}
