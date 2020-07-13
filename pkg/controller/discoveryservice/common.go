package discoveryservice

import (
	"fmt"

	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
)

func OwnedObjectName(ds *operatorv1alpha1.DiscoveryService) string {
	return fmt.Sprintf("%s-%s", "marin3r", ds.GetName())
}

func OwnedObjectNamespace(ds *operatorv1alpha1.DiscoveryService) string {
	return ds.Spec.DiscoveryServiceNamespace
}

func OwnedObjectAppLabel(ds *operatorv1alpha1.DiscoveryService) string {
	return fmt.Sprintf("%s-%s", "marin3r", ds.GetName())
}

func Labels(ds *operatorv1alpha1.DiscoveryService) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "marin3r",
		"app.kubernetes.io/managed-by": "marin3r-operator",
		"app.kubernetes.io/component":  "discovery-service",
		"app.kubernetes.io/instance":   ds.GetName(),
	}
}
