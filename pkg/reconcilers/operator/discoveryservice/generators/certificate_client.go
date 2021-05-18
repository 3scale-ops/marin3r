package generators

import (
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/envoy/container/defaults"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/lockedresources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (cfg *GeneratorOptions) ClientCertificate() lockedresources.GeneratorFunction {

	return func() client.Object {

		return &operatorv1alpha1.DiscoveryServiceCertificate{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DiscoveryServiceCertificate",
				APIVersion: operatorv1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      defaults.SidecarClientCertificate,
				Namespace: cfg.Namespace,
				Labels:    cfg.labels(),
			},
			Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
				CommonName: defaults.SidecarClientCertificate,
				ValidFor:   int64(cfg.ClientCertificateDuration.Seconds()),
				Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
					CASigned: &operatorv1alpha1.CASignedConfig{
						SecretRef: corev1.SecretReference{
							Name:      cfg.rootCertName(),
							Namespace: cfg.Namespace,
						}},
				},
				SecretRef: corev1.SecretReference{
					Name: defaults.SidecarClientCertificate,
				},
			},
		}
	}
}
