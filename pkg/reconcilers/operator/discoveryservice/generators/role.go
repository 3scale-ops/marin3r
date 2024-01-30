package generators

import (
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cfg *GeneratorOptions) Role() *rbacv1.Role {

	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.ResourceName(),
			Namespace: cfg.Namespace,
			Labels:    cfg.labels(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{corev1.SchemeGroupVersion.Group},
				Resources: []string{"secrets", "pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{marin3rv1alpha1.GroupVersion.Group},
				Resources: []string{rbacv1.ResourceAll},
				Verbs:     []string{rbacv1.VerbAll},
			},
			{
				APIGroups: []string{discoveryv1.SchemeGroupVersion.Group},
				Resources: []string{"endpointslices"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}
