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

package events

import (
	"context"
	"fmt"

	"github.com/roivaz/marin3r/pkg/reconciler"
	"github.com/roivaz/marin3r/pkg/util"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

const (
	// TODO: make the annotation to look for configurable
	annotation = "marin3r.3scale.net/node-id"
)

// ConfigMapHandler represents a kubernetes shared
// informer for ConfigMaps
type ConfigMapHandler struct {
	client    *util.K8s
	namespace string
	queue     chan reconciler.ReconcileJob
	ctx       context.Context
	logger    *zap.SugaredLogger
	stopper   chan struct{}
}

// NewConfigMapHandler creates a new ConfigMapHandler from
// the given params
func NewConfigMapHandler(
	ctx context.Context, client *util.K8s, namespace string, queue chan reconciler.ReconcileJob,
	logger *zap.SugaredLogger, stopper chan struct{}) *ConfigMapHandler {

	return &ConfigMapHandler{
		client:    client,
		namespace: namespace,
		queue:     queue,
		ctx:       ctx,
		logger:    logger,
		stopper:   stopper,
	}
}

// RunConfigMapHandler runs the ConfigMapHandler in a goroutine
// and waits forever until the stopper signal is sent to the
// stopper channel
func (cmh *ConfigMapHandler) RunConfigMapHandler() {

	factory := informers.NewSharedInformerFactoryWithOptions(cmh.client.Clientset, 0, informers.WithNamespace(cmh.namespace))
	informer := factory.Core().V1().ConfigMaps().Informer()
	defer runtime.HandleCrash()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { onConfigMapAdd(obj, cmh.queue, cmh.logger) },
		UpdateFunc: func(oldObj, newObj interface{}) { onConfigMapUpdate(newObj, cmh.queue, cmh.logger) },
		DeleteFunc: func(obj interface{}) { onConfigMapDelete(obj, cmh.queue, cmh.logger) },
	})
	go informer.Run(cmh.stopper)
	if !cache.WaitForCacheSync(cmh.stopper, informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	cmh.logger.Info("ConfigMap handler started")
	<-cmh.stopper
}

func onConfigMapAdd(obj interface{}, queue chan reconciler.ReconcileJob, logger *zap.SugaredLogger) {
	cm := obj.(*corev1.ConfigMap)
	if nodeID, ok := cm.GetAnnotations()[annotation]; ok {
		logger.Infof("ConfigMap '%s/%s' 'node-id=%s' added", cm.GetNamespace(), cm.GetName(), nodeID)
		reconciler.NewConfigMapReconcileJob(nodeID, reconciler.Add, *cm).Push(queue)
	}
}

func onConfigMapUpdate(obj interface{}, queue chan reconciler.ReconcileJob, logger *zap.SugaredLogger) {
	cm := obj.(*corev1.ConfigMap)
	if nodeID, ok := cm.GetAnnotations()[annotation]; ok {
		logger.Infof("ConfigMap '%s/%s' 'node-id=%s' updated", cm.GetNamespace(), cm.GetName(), nodeID)
		reconciler.NewConfigMapReconcileJob(nodeID, reconciler.Update, *cm).Push(queue)
	}
}

func onConfigMapDelete(obj interface{}, queue chan reconciler.ReconcileJob, logger *zap.SugaredLogger) {
	cm := obj.(*corev1.ConfigMap)
	if nodeID, ok := cm.GetAnnotations()[annotation]; ok {
		logger.Infof("ConfigMap '%s/%s' 'node-id=%s' deleted", cm.GetNamespace(), cm.GetName(), nodeID)
		reconciler.NewConfigMapReconcileJob(nodeID, reconciler.Delete, *cm).Push(queue)
	}
}
