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
	"fmt"

	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"go.uber.org/zap"
)

const (
	nodeID = "envoy-tls-sidecar"
)

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
	process(*Caches, *zap.SugaredLogger)
	Push(chan ReconcileJob)
}

func NewReconciler(snapshotCache *cache.SnapshotCache, stopper chan struct{}, logger *zap.SugaredLogger) *Reconciler {
	return &Reconciler{
		caches:        NewCaches(),
		snapshotCache: snapshotCache,
		Queue:         make(chan ReconcileJob),
		logger:        logger,
		stopper:       stopper,
	}
}

func (cw *Reconciler) RunReconciler() {

	// Watch for the call to shutdown the worker
	go func() {
		<-cw.stopper
		close(cw.Queue)
	}()

	for {
		job, more := <-cw.Queue
		if more {
			job.process(cw.caches, cw.logger)
		} else {
			cw.logger.Info("Received channel close, shutting down worker")
			return
		}

		// This would create an snapshot per event... we might want
		// to buffer events and push them all at the same time
		cw.makeSnapshot()
	}
}

func (cw *Reconciler) makeSnapshot() {
	cw.version++
	snapshotCache := *(cw.snapshotCache)

	cw.logger.Infof(">>>>>>>>>>>>>>>>>>> creating snapshot Version " + fmt.Sprint(cw.version))
	snap := cache.NewSnapshot(fmt.Sprint(cw.version),
		nil,
		cw.caches.makeClusterResources(),
		nil,
		cw.caches.makeListenerResources(),
		nil,
	)
	snap.Resources[cache.Secret] = cache.NewResources(fmt.Sprintf("%v", cw.version), cw.caches.makeSecretResources())
	// ID should not be hardcoded, probably a worker per configured ID would be nice
	snapshotCache.SetSnapshot(nodeID, snap)
}
