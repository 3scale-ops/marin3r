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

package envoy

import (
	"fmt"

	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

const (
	nodeID = "envoy-tls-sidecar"
)

//------------------
//----- Caches -----
//------------------

type Caches struct {
	secrets map[string]*auth.Secret
}

func NewCaches() *Caches {
	return &Caches{
		secrets: map[string]*auth.Secret{},
	}
}

// ----------------
// ---- Worker ----
// ----------------

type CacheWorker struct {
	version       int
	caches        *Caches
	snapshotCache *cache.SnapshotCache
	// TODO: do not go passing the channel around so freely,
	// create a queue object with a channel inside, not public,
	// and a set of public functions to access the channel
	Queue   chan ReconcileJob
	logger  *zap.SugaredLogger
	stopper chan struct{}
}

func NewCacheWorker(snapshotCache *cache.SnapshotCache, stopper chan struct{}, logger *zap.SugaredLogger) *CacheWorker {
	return &CacheWorker{
		caches:        NewCaches(),
		snapshotCache: snapshotCache,
		Queue:         make(chan ReconcileJob),
		logger:        logger,
		stopper:       stopper,
	}
}

func (cw *CacheWorker) RunCacheWorker() {

	// Watch for the call to shutdown the worker
	go func() {
		<-cw.stopper
		close(cw.Queue)
	}()

	for {
		job, more := <-cw.Queue
		if more {
			job.process(cw.caches)

			// if more {
			// 	switch job.jobType {
			// 	case "secret":
			// 		cw.caches.Secret(job.name, job.operation, (job.payload).(*corev1.Secret))
			// 	default:
			// 		cw.logger.Warn("Received an unknown type of job, discarding ...")
			// 	}
		} else {
			cw.logger.Info("Received channel close, shutting down worker")
			return
		}

		// This would create an snapshot per event... we might want
		// to buffer events and push them all at the same time
		cw.makeSnapshot()
	}
}

func (cw *CacheWorker) makeSnapshot() {
	cw.version++
	snapshotCache := *(cw.snapshotCache)
	secrets := make([]cache.Resource, len(cw.caches.secrets))
	i := 0
	for _, secret := range cw.caches.secrets {
		secrets[i] = secret
		i++
	}

	cw.logger.Infof(">>>>>>>>>>>>>>>>>>> creating snapshot Version " + fmt.Sprint(cw.version))
	snap := cache.NewSnapshot(fmt.Sprint(cw.version), nil, nil, nil, nil, nil)
	snap.Resources[cache.Secret] = cache.NewResources(fmt.Sprintf("%v", cw.version), secrets)
	// ID should not be hardcoded, probably a worker per configured ID would be nice
	snapshotCache.SetSnapshot(nodeID, snap)
}

// ------------------------------
// ---- Reconciler interface ----
// ------------------------------

type EventType int

const (
	Add EventType = iota
	Update
	Delete
)

type ReconcileJob interface {
	process(*Caches)
	Push(chan ReconcileJob)
}

// ---------------------------
// ---- Secret reconciler ----
// ---------------------------

type SecretReconcileJob struct {
	eventType EventType
	name      string
	secret    *corev1.Secret
}

func (srj SecretReconcileJob) Push(queue chan ReconcileJob) {
	queue <- srj
}

func (job SecretReconcileJob) process(c *Caches) {

	// TODO: do not always update the envoy secret. Not all object
	// updates in kubernetes mean an update of the envoy secret object
	if job.eventType == Add || job.eventType == Update {
		c.secrets[job.name] = NewSecret(
			job.name,
			string(job.secret.Data["tls.key"]),
			string(job.secret.Data["tls.crt"]),
		)
	} else {
		delete(c.secrets, job.name)
	}
}

func NewSecretReconcileJob(name string, eventType EventType, secret *corev1.Secret) *SecretReconcileJob {
	return &SecretReconcileJob{
		eventType: eventType,
		name:      name,
		secret:    secret,
	}
}

func (c *Caches) Secret(key, op string, s *corev1.Secret) {

}
