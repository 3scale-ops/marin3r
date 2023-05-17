package reconcilers

import (
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// IsInitialized checks whether the EnvoyConfigRevision object is initialized
// or not. Returns true if it has modified the EnvoyConfigRevision. Returns false if
// it has not.
func IsInitialized(ec *marin3rv1alpha1.EnvoyConfig) bool {
	ok := true

	if ec.Spec.EnvoyAPI == nil {
		ec.Spec.EnvoyAPI = pointer.New(ec.GetEnvoyAPIVersion())
		ok = false
	}

	if controllerutil.ContainsFinalizer(ec, marin3rv1alpha1.EnvoyConfigRevisionFinalizer) {
		controllerutil.RemoveFinalizer(ec, marin3rv1alpha1.EnvoyConfigRevisionFinalizer)
		ok = false
	}

	return ok
}
