package e2e

import (
	"context"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func waitForSecret(t *testing.T, kubeclient kubernetes.Interface, namespace, name string,
	retryInterval, timeout time.Duration) error {

	// t.Logf("Waiting for availability of Secret: %s in Namespace: %s \n", name, namespace)
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		_, err = kubeclient.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of Secret: %s in Namespace: %s \n", name, namespace)
				return false, nil
			}
			return false, err
		}

		return true, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Secret %s/%s available\n", namespace, name)
	return nil
}

func waitForReplicaSetDrain(t *testing.T, kubeclient kubernetes.Interface, namespace, name string,
	retryInterval, timeout time.Duration) error {

	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		replicaset, err := kubeclient.AppsV1().ReplicaSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for drain of ReplicaSet: %s in Namespace: %s \n", name, namespace)
				return false, nil
			}
			return false, err
		}

		if int(replicaset.Status.Replicas) == 0 {
			return true, nil
		}
		t.Logf("Waiting for drain of %s replicaset (%d/0)\n", name,
			replicaset.Status.Replicas)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("ReplicaSet %s/%s drained\n", namespace, name)
	return nil
}
