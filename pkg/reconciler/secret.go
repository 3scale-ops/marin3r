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

	"github.com/roivaz/marin3r/pkg/envoy"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	namespace = "3scale"
)

type SecretReconciler struct {
	clientset *kubernetes.Clientset
	queue     chan *envoy.WorkerJob
	ctx       context.Context
	logger    *zap.SugaredLogger
	stopper   chan struct{}
}

var version int32

func NewSecretReconciler(
	clientset *kubernetes.Clientset, queue chan *envoy.WorkerJob,
	ctx context.Context, logger *zap.SugaredLogger, stopper chan struct{}) *SecretReconciler {

	return &SecretReconciler{
		clientset: clientset,
		queue:     queue,
		ctx:       ctx,
		logger:    logger,
		stopper:   stopper,
	}
}

func (sr *SecretReconciler) RunSecretReconciler() {
	sr.logger.Info("Shared Informer app started")

	factory := informers.NewSharedInformerFactoryWithOptions(sr.clientset, 0, informers.WithNamespace(namespace))
	informer := factory.Core().V1().Secrets().Informer()
	defer runtime.HandleCrash()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) { onAdd(obj, sr.queue, sr.logger) },
	})
	go informer.Run(sr.stopper)
	if !cache.WaitForCacheSync(sr.stopper, informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}
	<-sr.stopper
}

// On and sends jobs to the queue when events on secrets holding certificates occur
func onAdd(obj interface{}, queue chan *envoy.WorkerJob, logger *zap.SugaredLogger) {
	secret := obj.(*corev1.Secret)
	// TODO: make the annotation to look for configurable so not just
	// cert-manager provided certs are supported
	if cn, ok := secret.GetAnnotations()["cert-manager.io/common-name"]; ok {
		logger.Infof("Certificate '%s/%s' added", secret.GetNamespace(), secret.GetName())
		envoy.SendSecretJob(cn, secret, queue)
	}
}
