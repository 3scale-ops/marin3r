// Copyright 2020 rvazquez@redhat.com
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package reconciler

import (
	"github.com/roivaz/marin3r/pkg/envoy"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type SecretReconcileJob struct {
	eventType EventType
	cn        string
	secret    *corev1.Secret
}

func NewSecretReconcileJob(cn string, eventType EventType, secret *corev1.Secret) *SecretReconcileJob {
	return &SecretReconcileJob{
		eventType: eventType,
		cn:        cn,
		secret:    secret,
	}
}

func (job SecretReconcileJob) Push(queue chan ReconcileJob) {
	queue <- job
}

func (job SecretReconcileJob) process(c caches, clientset *kubernetes.Clientset, namespace string, logger *zap.SugaredLogger) ([]string, error) {

	// SecretReconcileJob jobs don't have nodeID information because the secrets holding
	// the certificates in k8s/ocp can be created by other tools (eg cert-manager)
	// We need inspect the node-ids registered in the cache and publish the secrets to
	// all of them
	// TODO: improve this and publish secrets only to those node-ids actually interested
	// in them

	nodeIDs := make([]string, len(c))
	i := 0
	for k := range c {
		nodeIDs[i] = k
		i++
	}

	switch job.eventType {

	case Add, Update:
		// Copy the secret to all existent node caches
		for _, nodeID := range nodeIDs {
			c[nodeID].secrets[job.cn] = envoy.NewSecret(
				job.cn,
				string(job.secret.Data["tls.key"]),
				string(job.secret.Data["tls.crt"]),
			)
		}
	case Delete:
		logger.Warnf("The certificate with CN '%s' is about to be deleted", job.cn)
		for _, nodeID := range nodeIDs {
			delete(c[nodeID].secrets, job.cn)
		}
	}

	return nodeIDs, nil
}
