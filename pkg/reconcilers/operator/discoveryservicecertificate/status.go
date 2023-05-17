package reconcilers

import (
	"reflect"
	"time"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsStatusReconciled calculates the status of the resource
func IsStatusReconciled(dsc *operatorv1alpha1.DiscoveryServiceCertificate, certificateHash string,
	ready bool, notBefore, notAfter time.Time) bool {

	ok := true

	if dsc.Status.GetCertificateHash() != certificateHash {
		dsc.Status.CertificateHash = pointer.New(certificateHash)
		ok = false
	}

	if dsc.Status.IsReady() != ready {
		dsc.Status.Ready = pointer.New(ready)
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
		dsc.Status.Conditions = []metav1.Condition{}
		ok = false
	}

	return ok
}
