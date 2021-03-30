package reconcilers

import (
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"k8s.io/utils/pointer"
)

// IsInitialized checks whether the EnvoyConfigRevision object is initialized
// or not. Returns true if it has modified the EnvoyConfigRevision. Returns false if
// it has not.
func IsInitialized(dsc *operatorv1alpha1.DiscoveryServiceCertificate) bool {
	ok := true

	if dsc.Spec.IsServerCertificate == nil {
		dsc.Spec.IsServerCertificate = pointer.BoolPtr(dsc.IsServerCertificate())
		ok = false
	}
	if dsc.Spec.IsCA == nil {
		dsc.Spec.IsCA = pointer.BoolPtr(dsc.IsCA())
		ok = false
	}
	if dsc.Spec.Hosts == nil {
		dsc.Spec.Hosts = dsc.GetHosts()
		ok = false
	}
	if dsc.Spec.CertificateRenewalConfig == nil {
		crc := dsc.GetCertificateRenewalConfig()
		dsc.Spec.CertificateRenewalConfig = &crc
		ok = false
	}

	return ok
}
