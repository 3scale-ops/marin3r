package k8sutil

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func StringtoRawExtension(value string) *runtime.RawExtension {
	return &runtime.RawExtension{
		Raw: []byte(value),
	}
}

func ConditionsEqual(a, b *metav1.Condition) bool {
	if a == nil && b == nil {
		return true
	}

	if a != nil && b != nil && a.Type == b.Type && a.Reason == b.Reason && a.Message == b.Message {
		return true
	}
	return false
}
