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

func (srj SecretReconcileJob) Push(queue chan ReconcileJob) {
	queue <- srj
}

func (job SecretReconcileJob) process(c *nodeCaches, logger *zap.SugaredLogger) {

	// TODO: do not always update the envoy secret. Not all object
	// updates in kubernetes mean an update of the envoy secret object
	switch job.eventType {

	case Add, Update:
		c.secrets[job.cn] = envoy.NewSecret(
			job.cn,
			string(job.secret.Data["tls.key"]),
			string(job.secret.Data["tls.crt"]),
		)
	case Delete:
		logger.Warnf("The certificate with CN '%s' is about to be deleted", job.cn)
		delete(c.secrets, job.cn)
	}
}
