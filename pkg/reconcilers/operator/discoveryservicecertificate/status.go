package reconcilers

import (
	"fmt"
	"reflect"
	"time"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
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

	// Calculate if we are inside the renewal window (20% of the total certificate's duration)
	renewalWindow := float64(dsc.Spec.ValidFor) * 0.20
	if notAfter.Sub(time.Now()).Seconds() < renewalWindow &&
		!dsc.Status.Conditions.IsTrueFor(operatorv1alpha1.CertificateNeedsRenewalCondition) {
		dsc.Status.Conditions.SetCondition(status.Condition{
			Type:    operatorv1alpha1.CertificateNeedsRenewalCondition,
			Status:  corev1.ConditionTrue,
			Reason:  status.ConditionReason("CertificateAboutToExpire"),
			Message: fmt.Sprintf("Certificate wil expire in less than %v seconds", renewalWindow),
		})
		ok = false

	} else if notAfter.Sub(time.Now()).Seconds() > renewalWindow &&
		dsc.Status.Conditions.IsTrueFor(operatorv1alpha1.CertificateNeedsRenewalCondition) {
		dsc.Status.Conditions.RemoveCondition(operatorv1alpha1.CertificateNeedsRenewalCondition)
		ok = false
	}

	if dsc.Status.Conditions == nil {
		dsc.Status.Conditions = status.NewConditions()
		ok = false
	}

	return ok
}
