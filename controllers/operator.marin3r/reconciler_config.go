package controllers

import (
	"github.com/3scale-ops/basereconciler/reconciler"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func init() {
	reconciler.Config.AnnotationsDomain = "marin3r.3scale.net"
	reconciler.Config.ResourcePruner = false
	reconciler.Config.ManagedTypes = reconciler.NewManagedTypes().
		Register(&appsv1.DeploymentList{}).
		Register(&corev1.ServiceList{}).
		Register(&rbacv1.RoleList{}).
		Register(&rbacv1.RoleBindingList{}).
		Register(&corev1.ServiceAccountList{}).
		Register(&operatorv1alpha1.DiscoveryServiceCertificateList{}).
		Register(&policyv1.PodDisruptionBudgetList{}).
		Register(&autoscalingv2.HorizontalPodAutoscalerList{})
}
