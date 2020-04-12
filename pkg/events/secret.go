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
	// TODO: make the annotation to look for configurable so not just
	// cert-manager provided certs are supported
	certificateAnnotation = "cert-manager.io/common-name"
)

// SecretHandler represents a kubernetes shared
// informer for Secrets
type SecretHandler struct {
	client    *util.K8s
	namespace string
	queue     chan reconciler.ReconcileJob
	ctx       context.Context
	logger    *zap.SugaredLogger
	stopper   chan struct{}
}

var version int32

// NewSecretHandler creates a new SecretHandler from
// the given params
func NewSecretHandler(
	ctx context.Context, client *util.K8s, namespace string, queue chan reconciler.ReconcileJob,
	logger *zap.SugaredLogger, stopper chan struct{}) *SecretHandler {

	return &SecretHandler{
		client:    client,
		namespace: namespace,
		queue:     queue,
		ctx:       ctx,
		logger:    logger,
		stopper:   stopper,
	}
}

// RunSecretHandler runs the SecretHandler in a goroutine
// and waits forever until the stopper signal is sent to the
// stopper channel
func (sr *SecretHandler) RunSecretHandler() {

	factory := informers.NewSharedInformerFactoryWithOptions(sr.client.Clientset, 0, informers.WithNamespace(sr.namespace))
	informer := factory.Core().V1().Secrets().Informer()
	defer runtime.HandleCrash()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { onSecretAdd(obj, sr.queue, sr.logger) },
		UpdateFunc: func(oldObj, newObj interface{}) { onSecretUpdate(newObj, sr.queue, sr.logger) },
		DeleteFunc: func(obj interface{}) { onSecretDelete(obj, sr.queue, sr.logger) },
	})
	go informer.Run(sr.stopper)
	if !cache.WaitForCacheSync(sr.stopper, informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	sr.logger.Info("Secret handler started")
	<-sr.stopper
}

func onSecretAdd(obj interface{}, queue chan reconciler.ReconcileJob, logger *zap.SugaredLogger) {
	secret := obj.(*corev1.Secret)
	if cn, ok := secret.GetAnnotations()[certificateAnnotation]; ok {
		logger.Infof("Certificate '%s/%s' 'CN=%s' added", secret.GetNamespace(), secret.GetName(), cn)
		reconciler.NewSecretReconcileJob(cn, reconciler.Add, secret).Push(queue)
	}
}

// WARNING: onUpdate can be the first time we see a given certificate. Example:
//	- The secret is created in the cluster
//  - The secret is then annotated with the proper annotation that marks it as relevan for marin3r
//  - The informer event will be a "onUpdate" because it is an update from the point of view of k8s
//  - We need to watch for this casuistic
func onSecretUpdate(obj interface{}, queue chan reconciler.ReconcileJob, logger *zap.SugaredLogger) {
	secret := obj.(*corev1.Secret)
	if cn, ok := secret.GetAnnotations()[certificateAnnotation]; ok {
		logger.Infof("Certificate '%s/%s' 'CN=%s' updated", secret.GetNamespace(), secret.GetName(), cn)
		reconciler.NewSecretReconcileJob(cn, reconciler.Update, secret).Push(queue)
	}
}

func onSecretDelete(obj interface{}, queue chan reconciler.ReconcileJob, logger *zap.SugaredLogger) {
	secret := obj.(*corev1.Secret)
	if cn, ok := secret.GetAnnotations()[certificateAnnotation]; ok {
		logger.Infof("Certificate '%s/%s' 'CN=%s' deleted", secret.GetNamespace(), secret.GetName(), cn)
		reconciler.NewSecretReconcileJob(cn, reconciler.Delete, secret).Push(queue)
	}
}
