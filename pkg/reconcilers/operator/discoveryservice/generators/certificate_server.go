package generators

import (
	"fmt"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	// cert-manager
)

func (cfg *GeneratorOptions) ServerCertificate() client.Object {

	return &operatorv1alpha1.DiscoveryServiceCertificate{
		TypeMeta: metav1.TypeMeta{
			Kind:       operatorv1alpha1.DiscoveryServiceCertificateKind,
			APIVersion: operatorv1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.serverCertName(),
			Namespace: cfg.Namespace,
			Labels:    cfg.labels(),
		},
		Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
			CommonName:          fmt.Sprintf("%s-%s", cfg.ServerCertificateCommonNamePrefix, cfg.InstanceName),
			IsServerCertificate: pointer.BoolPtr(true),
			ValidFor:            int64(cfg.ServerCertificateDuration.Seconds()),
			Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
				CASigned: &operatorv1alpha1.CASignedConfig{
					SecretRef: corev1.SecretReference{
						Name:      cfg.rootCertName(),
						Namespace: cfg.Namespace,
					}},
			},
			SecretRef: corev1.SecretReference{
				Name:      cfg.serverCertName(),
				Namespace: cfg.Namespace,
			},
		},
	}
}
