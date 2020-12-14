package reconcilers

import (
	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// IsInitialized checks whether the EnvoyConfigRevision object is initialized
// or not. Returns true if it has modified the EnvoyConfigRevision. Returns false if
// it has not.
func IsInitialized(ecr *marin3rv1alpha1.EnvoyConfigRevision) bool {
	ok := true

	if ecr.Spec.EnvoyAPI == nil {
		ecr.Spec.EnvoyAPI = pointer.StringPtr(string(ecr.GetEnvoyAPIVersion()))
		ok = false
	}

	if ecr.Spec.Serialization == nil {
		ecr.Spec.Serialization = pointer.StringPtr(string(ecr.GetSerialization()))
		ok = false
	}

	if !controllerutil.ContainsFinalizer(ecr, marin3rv1alpha1.EnvoyConfigRevisionFinalizer) {
		controllerutil.AddFinalizer(ecr, marin3rv1alpha1.EnvoyConfigRevisionFinalizer)
		ok = false
	}

	return ok
}
