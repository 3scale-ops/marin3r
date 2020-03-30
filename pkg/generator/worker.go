package generator

import (
	"fmt"

	auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/roivaz/marin3r/pkg/envoy"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

type WorkerJob struct {
	jobType string
	name    string
	payload interface{}
}

type CacheWorker struct {
	version       int
	caches        *Caches
	snapshotCache *cache.SnapshotCache
	Queue         chan *WorkerJob
	logger        *zap.SugaredLogger
	stopper       chan struct{}
}

func NewCacheWorker(snapshotCache *cache.SnapshotCache, stopper chan struct{}, logger *zap.SugaredLogger) *CacheWorker {
	return &CacheWorker{
		caches:        NewCaches(),
		snapshotCache: snapshotCache,
		Queue:         make(chan *WorkerJob),
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
			switch job.jobType {
			case "secret":
				cw.caches.AddSecret(job.name, (job.payload).(*corev1.Secret))
			default:
				cw.logger.Warn("Received an unknown type of job, discarding ...")
			}
		} else {
			cw.logger.Info("Received channel close, shutting down worker")
			return
		}

		// This would create an snapshot per event... we might want
		// to buffer events and push them all at the same time
		cw.makeSnapshot()
	}
}

func SendSecretJob(name string, payload *corev1.Secret, queue chan *WorkerJob) {
	j := &WorkerJob{
		jobType: "secret",
		name:    name,
		payload: payload,
	}
	queue <- j
}

func (cw *CacheWorker) makeSnapshot() {
	cw.version++
	snapshotCache := *(cw.snapshotCache)
	// spew.Dump(snapshotCache.GetSnapshot("test-id"))
	secrets := make([]cache.Resource, len(cw.caches.secrets))
	i := 0
	for _, secret := range cw.caches.secrets {
		secrets[i] = secret
		i++
	}

	cw.logger.Infof(">>>>>>>>>>>>>>>>>>> creating snapshot Version " + fmt.Sprint(cw.version))
	snap := cache.NewSnapshot(fmt.Sprint(cw.version), nil, nil, nil, nil, nil)
	snap.Resources[cache.Secret] = cache.NewResources(fmt.Sprintf("%v", cw.version), secrets)
	// ID should not be hardcoded, probably a worker per configured ID would be cool
	// snapshotCache.ClearSnapshot("test-id")
	snapshotCache.SetSnapshot("test-id", snap)
}

type Caches struct {
	secrets map[string]*auth.Secret
}

func NewCaches() *Caches {
	return &Caches{
		secrets: map[string]*auth.Secret{},
	}
}
func (c *Caches) AddSecret(key string, s *corev1.Secret) {
	c.secrets[key] = envoy.NewSecret(
		key,
		string(s.Data["tls.key"]),
		string(s.Data["tls.crt"]),
	)
}
