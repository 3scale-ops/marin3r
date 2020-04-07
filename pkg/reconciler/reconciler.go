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
	"k8s.io/client-go/kubernetes"
)

const (
	// TODO: make the annotation to look for configurable so not just
	// cert-manager provided certs are supported
	certificateAnnotation = "cert-manager.io/common-name"
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

// ReconcileJob is the interface that all types
// of jobs must adhere to. It has a "process" function
// which holds the logic of how the item is to be processed
// and a "Push" function that pushes new jobs of the same
// type to the job queue
type ReconcileJob interface {
	process(caches, *kubernetes.Clientset, string, *zap.SugaredLogger) ([]string, error)
	Push(chan ReconcileJob)
}

//------------------
//----- Worker -----
//------------------

// Reconciler is a loop that reads jobs
// from a queue and processes them to reconcile
// input information (usually in the form of events)
// with output envoy configuration
type Reconciler struct {
	clientset     *kubernetes.Clientset
	namespace     string
	version       int
	caches        caches
	snapshotCache *cache.SnapshotCache
	// TODO: do not go passing the channel around so freely,
	// create a queue object with a channel inside, not public,
	// and a set of public functions to access the channel
	Queue   chan ReconcileJob
	logger  *zap.SugaredLogger
	stopper chan struct{}
}

// NewReconciler returns a new Reconciler object built using the
// passed parameters
func NewReconciler(clientset *kubernetes.Clientset, namespace string,
	snapshotCache *cache.SnapshotCache, stopper chan struct{},
	logger *zap.SugaredLogger) *Reconciler {
	return &Reconciler{
		clientset:     clientset,
		namespace:     namespace,
		caches:        make(caches),
		snapshotCache: snapshotCache,
		Queue:         make(chan ReconcileJob),
		logger:        logger,
		stopper:       stopper,
	}
}

// RunReconciler runs the reconciler loop
// until the stop signal is sent to it
func (r *Reconciler) RunReconciler() {

	// Watch for the call to shutdown the worker
	r.runStopWatcher()

	for {
		job, more := <-r.Queue
		if more {
			{
				// Recover from panics in job processing
				defer func() {
					if recov := recover(); r != nil {
						r.logger.Warnf("Recovered from panicked job", recov)
					}
				}()
				nodeIDs, err := job.process(r.caches, r.clientset, r.namespace, r.logger)
				if err != nil {
					break
				}
				for _, nodeID := range nodeIDs {
					r.makeSnapshot(nodeID)
				}
			}
		} else {
			r.logger.Info("Received channel close, shutting down worker")
			return
		}
	}
}

func (r *Reconciler) runStopWatcher() {
	go func() {
		<-r.stopper
		close(r.Queue)
	}()
}

func (r *Reconciler) makeSnapshot(nodeID string) {
	r.version++
	snapshotCache := *(r.snapshotCache)

	r.logger.Infof(">>> creating snapshot version '%s' for node-id '%s'", fmt.Sprint(r.version), nodeID)
	snap := cache.NewSnapshot(fmt.Sprint(r.version),
		nil,
		r.caches[nodeID].makeClusterResources(),
		nil,
		r.caches[nodeID].makeListenerResources(),
		nil,
	)
	snap.Resources[cache.Secret] = cache.NewResources(fmt.Sprintf("%v", r.version), r.caches[nodeID].makeSecretResources())
	// Push snapshot to the server for the given node-id
	snapshotCache.SetSnapshot(nodeID, snap)
}
