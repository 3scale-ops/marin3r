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
	"context"
	"fmt"

	"github.com/3scale/marin3r/pkg/cache"
	"github.com/3scale/marin3r/pkg/util"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"go.uber.org/zap"
)

const (
	// TODO: make the annotation to look for configurable so not just
	// cert-manager provided certs are supported
	certificateAnnotation = "cert-manager.io/common-name"
)

// ------------------------------
// ---- Reconciler interface ----
// ------------------------------

// EventType is a type that holds information
// on the type of event received in the queue
type EventType int

const (
	// Add is an event triggered by a "create"
	// operation in the k8s api
	Add EventType = iota
	// Update is an event triggered by a "update"
	// operation in the k8s api
	Update
	// Delete is an event triggered by a "delete"
	// operation in the k8s api
	Delete
)

// ReconcileJob is the interface that all types
// of jobs must adhere to. It has a "process" function
// which holds the logic of how the item is to be processed
// and a "Push" function that pushes new jobs of the same
// type to the job queue
type ReconcileJob interface {
	process(cache.Cache, *util.K8s, string, *zap.SugaredLogger) ([]string, error)
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
	ctx           context.Context
	client        *util.K8s
	namespace     string
	cache         cache.Cache
	snapshotCache *xds_cache.SnapshotCache
	// TODO: do not go passing the channel around so freely,
	// create a queue object with a channel inside, not public,
	// and a set of public functions to access the channel
	Queue  chan ReconcileJob
	logger *zap.SugaredLogger
}

// NewReconciler returns a new Reconciler object built using the
// passed parameters
func NewReconciler(ctx context.Context, client *util.K8s, namespace string,
	snapshotCache *xds_cache.SnapshotCache, logger *zap.SugaredLogger) *Reconciler {
	return &Reconciler{
		ctx:           ctx,
		client:        client,
		namespace:     namespace,
		cache:         cache.NewCache(),
		snapshotCache: snapshotCache,
		Queue:         make(chan ReconcileJob),
		logger:        logger,
	}
}

// RunReconciler runs the reconciler loop
// until the stop signal is sent to it
func (r *Reconciler) RunReconciler() {

	// Watch for the call to shutdown the worker
	r.runStopWatcher()
	r.logger.Info("Reconcile worker started")
	for {
		job, more := <-r.Queue
		if more {
			{
				nodeIDs, err := safeJobExecutor(job, r.cache, r.client, r.namespace, r.logger)
				if err != nil {
					continue
				}
				for _, nodeID := range nodeIDs {
					r.cache.BumpCacheVersion(nodeID)
					r.cache.SetSnapshot(nodeID, *r.snapshotCache)
				}
			}
		} else {
			r.logger.Info("Shutting down reconcile worker")
			return
		}
	}
}

func (r *Reconciler) runStopWatcher() {
	go func() {
		<-r.ctx.Done()
		close(r.Queue)
	}()
}

// sfaeJobExecutor wraps the process() function and handles
// panics so the whole worker does not crush on a job panic
func safeJobExecutor(job ReconcileJob, cache cache.Cache, client *util.K8s, namespace string, logger *zap.SugaredLogger) (nodeIDs []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Warnf("Recovered from panicked job: '%s'", r)
			err = fmt.Errorf("Recovered from panicked job: '%s'", r)
			nodeIDs = []string{}
		}
	}()

	nodeIDs, err = job.process(cache, client, namespace, logger)
	return nodeIDs, err
}
