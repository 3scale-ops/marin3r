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

	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"go.uber.org/zap"
)

//------------------
//----- Caches -----
//------------------

type Caches struct {
	secrets   map[string]*auth.Secret
	listeners map[string]*envoyapi.Listener
	clusters  map[string]*envoyapi.Cluster
	endpoint  map[string]*envoyapi.ClusterLoadAssignment
}

func NewCaches() *Caches {
	return &Caches{
		secrets:   map[string]*auth.Secret{},
		listeners: map[string]*envoyapi.Listener{},
		clusters:  map[string]*envoyapi.Cluster{},
		endpoint:  map[string]*envoyapi.ClusterLoadAssignment{},
	}
}

func (c *Caches) makeSecretResources() []cache.Resource {
	secrets := make([]cache.Resource, len(c.secrets))
	i := 0
	for _, secret := range c.secrets {
		secrets[i] = secret
		i++
	}
	return secrets
}

func (c *Caches) makeClusterResources() []cache.Resource {
	clusters := make([]cache.Resource, len(c.clusters))
	i := 0
	for _, cluster := range c.clusters {
		clusters[i] = cluster
		i++
	}
	return clusters
}

func (c *Caches) makeListenerResources() []cache.Resource {
	listeners := make([]cache.Resource, len(c.listeners))
	i := 0
	for _, listener := range c.listeners {
		listeners[i] = listener
		i++
	}
	return listeners
}

// func (c *Caches) makeEndpointResources() []cache.Resource {
// 	endpoints := make([]cache.Resource, len(c.endpoints))
// 	i := 0
// 	for _, endpoint := range c.endpoints {
// 		endpoints[i] = endpoint
// 		i++
// 	}
// 	return endpoints
// }

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

func (cw *CacheWorker) makeSnapshot() {
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
