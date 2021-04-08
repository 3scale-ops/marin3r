package reconcilers

import (
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	"k8s.io/utils/pointer"
)

// IsStatusReconciled calculates the status of the resource
func IsStatusReconciled(eb *marin3rv1alpha1.EnvoyBootstrap, configHashV2, configHashV3 string) bool {

	ok := true

	if eb.Status.GetConfigHashV2() != configHashV2 {
		eb.Status.ConfigHashV2 = pointer.StringPtr(configHashV2)
		ok = false
	}

	if eb.Status.GetConfigHashV3() != configHashV3 {
		eb.Status.ConfigHashV3 = pointer.StringPtr(configHashV3)
		ok = false
	}

	return ok
}
