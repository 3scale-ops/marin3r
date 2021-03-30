package generators

import (
	"fmt"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/lockedresources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (cfg *GeneratorOptions) RootCertificationAuthority() lockedresources.GeneratorFunction {

	return func() client.Object {

		return &operatorv1alpha1.DiscoveryServiceCertificate{
			TypeMeta: metav1.TypeMeta{
				Kind:       operatorv1alpha1.DiscoveryServiceCertificateKind,
				APIVersion: operatorv1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      cfg.rootCertName(),
				Namespace: cfg.Namespace,
				Labels:    cfg.labels(),
			},
			Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
				CommonName: fmt.Sprintf("%s-%s", cfg.RootCertificateCommonNamePrefix, cfg.InstanceName),
				IsCA:       pointer.BoolPtr(true),
				ValidFor:   int64(cfg.RootCertificateDuration.Seconds()),
				Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
					SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
				},
				SecretRef: corev1.SecretReference{
					Name:      cfg.rootCertName(),
					Namespace: cfg.Namespace,
				},
				CertificateRenewalConfig: &operatorv1alpha1.CertificateRenewalConfig{
					Enabled: false,
				},
			},
		}
	}
}
