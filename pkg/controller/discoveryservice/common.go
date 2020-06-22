package discoveryservice

import (
	"fmt"
)

func (r *ReconcileDiscoveryService) getName() string {
	return fmt.Sprintf("%s-%s", "marin3r", r.ds.GetName())
}

func (r *ReconcileDiscoveryService) getNamespace() string {
	return r.ds.Spec.DiscoveryServiceNamespace
}

func (r *ReconcileDiscoveryService) getAppLabel() string {
	return fmt.Sprintf("%s-%s", "marin3r", r.ds.GetName())
}

func (r *ReconcileDiscoveryService) getDiscoveryServiceHost() string {
	return fmt.Sprintf("%s.%s.%s", r.getName(), r.getNamespace(), "svc.cluster.local")
}

func (r *ReconcileDiscoveryService) getDiscoveryServicePort() uint32 {
	return uint32(18000)
}