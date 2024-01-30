package generators

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cfg *GeneratorOptions) RoleBinding() *rbacv1.RoleBinding {

	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.ResourceName(),
			Namespace: cfg.Namespace,
			Labels:    cfg.labels(),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     cfg.ResourceName(),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      cfg.ResourceName(),
				Namespace: cfg.Namespace,
			},
		},
	}
}
