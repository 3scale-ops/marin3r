package controllers

import (
	"github.com/3scale-ops/basereconciler/config"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	config.SetAnnotationsDomain("marin3r.3scale.net")
	config.DisableResourcePruner()
	config.DisableDynamicWatches()
	config.SetDefaultReconcileConfigForGVK(
		schema.FromAPIVersionAndKind("v1", "Service"),
		config.ReconcileConfigForGVK{
			EnsureProperties: []string{
				"metadata.annotations",
				"metadata.labels",
				"spec.type",
				"spec.ports",
				"spec.selector",
				"spec.clusterIP",
				"spec.clusterIPs",
			},
		})
	config.SetDefaultReconcileConfigForGVK(
		schema.FromAPIVersionAndKind("apps/v1", "Deployment"),
		config.ReconcileConfigForGVK{
			EnsureProperties: []string{
				"metadata.annotations",
				"metadata.labels",
				"spec.minReadySeconds",
				"spec.replicas",
				"spec.selector",
				"spec.strategy",
				"spec.template.metadata.labels",
				"spec.template.metadata.annotations",
				"spec.template.spec",
			},
			IgnoreProperties: []string{
				"metadata.annotations['deployment.kubernetes.io/revision']",
				"spec.template.spec.dnsPolicy",
				"spec.template.spec.schedulerName",
				"spec.template.spec.restartPolicy",
				"spec.template.spec.securityContext",
				"spec.template.spec.containers[*].terminationMessagePath",
				"spec.template.spec.containers[*].terminationMessagePolicy",
				"spec.template.spec.initContainers[*].terminationMessagePath",
				"spec.template.spec.initContainers[*].terminationMessagePolicy",
			},
		})
	config.SetDefaultReconcileConfigForGVK(
		schema.FromAPIVersionAndKind("autoscaling/v2", "HorizontalPodAutoscaler"),
		config.ReconcileConfigForGVK{
			EnsureProperties: []string{
				"metadata.annotations",
				"metadata.labels",
				"spec.scaleTargetRef",
				"spec.minReplicas",
				"spec.maxReplicas",
				"spec.metrics",
			},
		})
	config.SetDefaultReconcileConfigForGVK(
		schema.FromAPIVersionAndKind("policy/v1", "PodDisruptionBudget"),
		config.ReconcileConfigForGVK{
			EnsureProperties: []string{
				"metadata.annotations",
				"metadata.labels",
				"spec.maxUnavailable",
				"spec.minAvailable",
				"spec.selector",
			},
		})
	config.SetDefaultReconcileConfigForGVK(
		schema.FromAPIVersionAndKind("rbac.authorization.k8s.io/v1", "Role"),
		config.ReconcileConfigForGVK{
			EnsureProperties: []string{
				"metadata.annotations",
				"metadata.labels",
				"rules",
			},
		})
	config.SetDefaultReconcileConfigForGVK(
		schema.FromAPIVersionAndKind("rbac.authorization.k8s.io/v1", "RoleBinding"),
		config.ReconcileConfigForGVK{
			EnsureProperties: []string{
				"metadata.annotations",
				"metadata.labels",
				"roleRef",
				"subjects",
			},
		})
}
