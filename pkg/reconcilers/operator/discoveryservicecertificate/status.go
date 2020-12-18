package reconcilers

import (
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/operator-framework/operator-lib/status"
	"k8s.io/utils/pointer"
)

// IsStatusReconciled calculates the status of the resource
func IsStatusReconciled(dsc *operatorv1alpha1.DiscoveryServiceCertificate, certificateHash string, ready bool) bool {

	ok := true

	if dsc.Status.GetCertificateHash() != certificateHash {
		dsc.Status.CertificateHash = pointer.StringPtr(certificateHash)
		ok = false
	}

	if dsc.Status.IsReady() != ready {
		dsc.Status.Ready = pointer.BoolPtr(ready)
		ok = false
	}

	if ready && dsc.Status.Conditions.IsTrueFor(operatorv1alpha1.CertificateNeedsRenewalCondition) {
		dsc.Status.Conditions.RemoveCondition(operatorv1alpha1.CertificateNeedsRenewalCondition)
		ok = false
	}

	if dsc.Status.Conditions == nil {
		dsc.Status.Conditions = status.NewConditions()
		ok = false
	}

	return ok
}
