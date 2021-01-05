package reconcilers

import (
	"reflect"
	"time"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/operator-framework/operator-lib/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// IsStatusReconciled calculates the status of the resource
func IsStatusReconciled(dsc *operatorv1alpha1.DiscoveryServiceCertificate, certificateHash string,
	ready bool, notBefore, notAfter time.Time) bool {

	ok := true

	if dsc.Status.GetCertificateHash() != certificateHash {
		dsc.Status.CertificateHash = pointer.StringPtr(certificateHash)
		ok = false
	}

	if dsc.Status.IsReady() != ready {
		dsc.Status.Ready = pointer.BoolPtr(ready)
		ok = false
	}

	if !reflect.DeepEqual(dsc.Status.NotBefore, &metav1.Time{Time: notBefore}) {
		dsc.Status.NotBefore = &metav1.Time{Time: notBefore}
		ok = false
	}

	if !reflect.DeepEqual(dsc.Status.NotAfter, &metav1.Time{Time: notAfter}) {
		dsc.Status.NotAfter = &metav1.Time{Time: notAfter}
		ok = false
	}

	if dsc.Status.Conditions == nil {
		dsc.Status.Conditions = status.NewConditions()
		ok = false
	}

	return ok
}
