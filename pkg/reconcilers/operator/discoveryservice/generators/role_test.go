package generators

import (
	"testing"
	"time"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGeneratorOptions_Role(t *testing.T) {
	type args struct {
		hash string
	}
	tests := []struct {
		name string
		opts GeneratorOptions
		args args
		want *rbacv1.Role
	}{
		{"Generates a Role",
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
			&rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "marin3r-test",
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "marin3r",
						"app.kubernetes.io/managed-by": "marin3r-operator",
						"app.kubernetes.io/component":  "discovery-service",
						"app.kubernetes.io/instance":   "test",
					},
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.opts.Role(), tt.want); len(diff) > 0 {
				t.Errorf("GeneratorOptions.Role() DIFF:\n %v", diff)
			}
		})
	}
}
