package generators

import (
	"github.com/3scale-ops/marin3r/pkg/reconcilers/lockedresources"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (cfg *GeneratorOptions) PDB() lockedresources.GeneratorFunction {

	return func() client.Object {

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
