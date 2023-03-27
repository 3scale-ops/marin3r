package generators

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cfg *GeneratorOptions) ServiceAccount() func() *corev1.ServiceAccount {

	return func() *corev1.ServiceAccount {

		return &corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceAccount",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      cfg.ResourceName(),
				Namespace: cfg.Namespace,
				Labels:    cfg.labels(),
			},
		}
	}
}
