package control

import (
	"context"
	"fmt"
	"os"

	"github.com/roivaz/marin3r/pkg/generator"
	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/util/runtime"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	namespace = "3scale"
)

type SecretReconciler struct {
	queue   chan *generator.WorkerJob
	ctx     context.Context
	logger  *zap.SugaredLogger
	stopper chan struct{}
}

var version int32

func NewSecretReconciler(queue chan *generator.WorkerJob, ctx context.Context, logger *zap.SugaredLogger, stopper chan struct{}) *SecretReconciler {

	return &SecretReconciler{
		queue:   queue,
		ctx:     ctx,
		logger:  logger,
		stopper: stopper,
	}
}

func (sr *SecretReconciler) RunSecretReconciler() {
	sr.logger.Info("Shared Informer app started")
	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		sr.logger.Panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		sr.logger.Panic(err.Error())
	}

	factory := informers.NewSharedInformerFactoryWithOptions(clientset, 0, informers.WithNamespace(namespace))
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
func onAdd(obj interface{}, queue chan *generator.WorkerJob, logger *zap.SugaredLogger) {
	secret := obj.(*corev1.Secret)
	// TODO: make the annotation to look for configurable so not just
	// cert-manager provided certs are supported
	if cn, ok := secret.GetAnnotations()["cert-manager.io/common-name"]; ok {
		logger.Infof("Certificate '%s/%s' added", secret.GetNamespace(), secret.GetName())
		generator.SendSecretJob(cn, secret, queue)
	}
}
