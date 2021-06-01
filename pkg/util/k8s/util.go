package k8sutil

import "github.com/operator-framework/operator-lib/status"

func ConditionsEqual(a, b *status.Condition) bool {
	if a == nil && b == nil {
		return true
	}

	if a != nil && b != nil && a.Type == b.Type && a.Reason == b.Reason && a.Message == b.Message {
		return true
	}
	return false
}
