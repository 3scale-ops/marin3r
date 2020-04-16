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

	"github.com/3scale/marin3r/pkg/reconciler"
	"github.com/3scale/marin3r/pkg/util"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

const (
	// TODO: make the configmapAnnotation to look for configurable
	configmapAnnotation = "marin3r.3scale.net/node-id"
)

// ConfigMapHandler represents a kubernetes shared
// informer for ConfigMaps
type ConfigMapHandler struct {
	ctx       context.Context
	client    *util.K8s
	namespace string
	queue     chan reconciler.ReconcileJob
	logger    *zap.SugaredLogger
}

// NewConfigMapHandler creates a new ConfigMapHandler from
// the given params
func NewConfigMapHandler(
	ctx context.Context, client *util.K8s, namespace string, queue chan reconciler.ReconcileJob,
	logger *zap.SugaredLogger) *ConfigMapHandler {

	return &ConfigMapHandler{
		ctx:       ctx,
		client:    client,
		namespace: namespace,
		queue:     queue,
		logger:    logger,
	}
}

// RunConfigMapHandler runs the ConfigMapHandler in a goroutine
// and waits forever until the stopper signal is sent to the
// stopper channel
func (cmh *ConfigMapHandler) RunConfigMapHandler() error {

	factory := informers.NewSharedInformerFactoryWithOptions(cmh.client.Clientset, 0, informers.WithNamespace(cmh.namespace))
	informer := factory.Core().V1().ConfigMaps().Informer()
	defer runtime.HandleCrash()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { onConfigMapAdd(obj, cmh.queue, cmh.logger) },
		UpdateFunc: func(oldObj, newObj interface{}) { onConfigMapUpdate(newObj, cmh.queue, cmh.logger) },
		DeleteFunc: func(obj interface{}) { onConfigMapDelete(obj, cmh.queue, cmh.logger) },
	})
	stopper := make(chan struct{})
	go informer.Run(stopper)

	cmh.logger.Info("ConfigMap handler started")

	// Shutdown when ctx is canceled
	<-cmh.ctx.Done()
	close(stopper)
	return nil
}

func onConfigMapAdd(obj interface{}, queue chan reconciler.ReconcileJob, logger *zap.SugaredLogger) {
	cm := obj.(*corev1.ConfigMap)
	if nodeID, ok := cm.GetAnnotations()[configmapAnnotation]; ok {
		logger.Infof("ConfigMap '%s/%s' 'node-id=%s' added", cm.GetNamespace(), cm.GetName(), nodeID)
		reconciler.NewConfigMapReconcileJob(nodeID, reconciler.Add, *cm).Push(queue)
	}
}

func onConfigMapUpdate(obj interface{}, queue chan reconciler.ReconcileJob, logger *zap.SugaredLogger) {
	cm := obj.(*corev1.ConfigMap)
	if nodeID, ok := cm.GetAnnotations()[configmapAnnotation]; ok {
		logger.Infof("ConfigMap '%s/%s' 'node-id=%s' updated", cm.GetNamespace(), cm.GetName(), nodeID)
		reconciler.NewConfigMapReconcileJob(nodeID, reconciler.Update, *cm).Push(queue)
	}
}

func onConfigMapDelete(obj interface{}, queue chan reconciler.ReconcileJob, logger *zap.SugaredLogger) {
	cm := obj.(*corev1.ConfigMap)
	if nodeID, ok := cm.GetAnnotations()[configmapAnnotation]; ok {
		logger.Infof("ConfigMap '%s/%s' 'node-id=%s' deleted", cm.GetNamespace(), cm.GetName(), nodeID)
		reconciler.NewConfigMapReconcileJob(nodeID, reconciler.Delete, *cm).Push(queue)
	}
}
