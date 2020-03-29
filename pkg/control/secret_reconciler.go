package control

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/roivaz/marin3r/pkg/envoy"
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
	ctx            context.Context
	envoyXdsServer *envoy.XdsServer
	logger         *zap.SugaredLogger
	stopper        chan struct{}
}

func NewSecretReconciler(ctx context.Context, envoyXdsServer *envoy.XdsServer, logger *zap.SugaredLogger, stopper chan struct{}) *SecretReconciler {

	return &SecretReconciler{
		ctx:            ctx,
		envoyXdsServer: envoyXdsServer,
		logger:         logger,
		stopper:        stopper,
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
		AddFunc: onAdd,
	})
	go informer.Run(sr.stopper)
	if !cache.WaitForCacheSync(sr.stopper, informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}
	<-sr.stopper
}

func onAdd(obj interface{}) {
	secret := obj.(*corev1.Secret)
	name := secret.GetName()
	namespace := secret.GetNamespace()
	log.Printf("Secret '%s/%s' added", namespace, name)
}
