package reconcilers

import (
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// IsInitialized checks whether the EnvoyConfigRevision object is initialized
// or not. Returns true if it has modified the EnvoyConfigRevision. Returns false if
// it has not.
func IsInitialized(ec *marin3rv1alpha1.EnvoyConfig) bool {
	ok := true

	if ec.Spec.EnvoyAPI == nil {
		ec.Spec.EnvoyAPI = pointer.StringPtr(string(ec.GetEnvoyAPIVersion()))
		ok = false
	}

	if ec.Spec.Serialization == nil {
		ec.Spec.Serialization = pointer.StringPtr(string(ec.GetSerialization()))
		ok = false
	}

	if controllerutil.ContainsFinalizer(ec, marin3rv1alpha1.EnvoyConfigRevisionFinalizer) {
		controllerutil.RemoveFinalizer(ec, marin3rv1alpha1.EnvoyConfigRevisionFinalizer)
		ok = false
	}

	return ok
}
