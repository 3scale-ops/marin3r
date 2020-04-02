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
	corev1 "k8s.io/api/core/v1"
)

type SecretReconcileJob struct {
	eventType EventType
	name      string
	secret    *corev1.Secret
}

func NewSecretReconcileJob(name string, eventType EventType, secret *corev1.Secret) *SecretReconcileJob {
	return &SecretReconcileJob{
		eventType: eventType,
		name:      name,
		secret:    secret,
	}
}

func (srj SecretReconcileJob) Push(queue chan ReconcileJob) {
	queue <- srj
}

func (job SecretReconcileJob) process(c *Caches) {

	// TODO: do not always update the envoy secret. Not all object
	// updates in kubernetes mean an update of the envoy secret object
	if job.eventType == Add || job.eventType == Update {
		c.secrets[job.name] = envoy.NewSecret(
			job.name,
			string(job.secret.Data["tls.key"]),
			string(job.secret.Data["tls.crt"]),
		)
	} else {
		delete(c.secrets, job.name)
	}
}
