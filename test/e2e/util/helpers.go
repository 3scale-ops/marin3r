package e2e

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ReadyReplicas(k8sClient client.Client, namespace string, matchingLabels client.MatchingLabels) int {
	podList := &corev1.PodList{}
	_ = k8sClient.List(
		context.Background(),
		podList,
		[]client.ListOption{matchingLabels, client.InNamespace(namespace)}...,
	)

	readyCount := 0
	for _, pod := range podList.Items {
		if cond := GetPodCondition(pod.Status.Conditions, corev1.PodReady); cond != nil {
			if cond.Status == corev1.ConditionTrue {
				readyCount++
			}
		}
	}

	return readyCount
}

func GetPodCondition(conditions []corev1.PodCondition, conditionType corev1.PodConditionType) *corev1.PodCondition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}
