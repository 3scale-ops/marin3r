package generators

import (
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cfg *GeneratorOptions) PDB() func() *policyv1.PodDisruptionBudget {

	return func() *policyv1.PodDisruptionBudget {

		return &policyv1.PodDisruptionBudget{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PodDisruptionBudget",
				APIVersion: policyv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      cfg.resourceName(),
				Namespace: cfg.Namespace,
				Labels:    cfg.labels(),
			},
			Spec: func() policyv1.PodDisruptionBudgetSpec {
				spec := policyv1.PodDisruptionBudgetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: cfg.labels(),
					},
				}
				if cfg.PodDisruptionBudget.MinAvailable != nil {
					spec.MinAvailable = cfg.PodDisruptionBudget.MinAvailable
				} else {
					spec.MaxUnavailable = cfg.PodDisruptionBudget.MaxUnavailable
				}
				return spec
			}(),
		}
	}
}
