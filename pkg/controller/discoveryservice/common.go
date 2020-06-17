package discoveryservice

import "fmt"

func (r *ReconcileDiscoveryService) getName() string {
	return fmt.Sprintf("%s-%s", "marin3r", r.ds.GetName())
}

func (r *ReconcileDiscoveryService) getNamespace() string {
	return r.ds.Spec.DiscoveryServiceNamespace
}

func (r *ReconcileDiscoveryService) getAppLabel() string {
	return fmt.Sprintf("%s-%s", "marin3r", r.ds.GetName())
}
