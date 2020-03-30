package control

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/roivaz/marin3r/pkg/envoy"
	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/util/runtime"

	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache"
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

var version int32

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
		AddFunc: func(obj interface{}) { onAdd(obj, sr) },
	})
	go informer.Run(sr.stopper)
	if !cache.WaitForCacheSync(sr.stopper, informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}
	<-sr.stopper
}

func onAdd(obj interface{}, sr *SecretReconciler) {
	secret := obj.(*corev1.Secret)
	name := secret.GetName()
	namespace := secret.GetNamespace()

	if cn, ok := secret.GetAnnotations()["cert-manager.io/common-name"]; ok {
		sr.logger.Infof("Certificate '%s/%s' added", namespace, name)
		// This is a small thing to test serving a certificate through sds api
		privateKey := string(secret.Data["tls.key"])
		certificate := string(secret.Data["tls.crt"])
		secrets := []envoycache.Resource{
			envoy.NewSecret(cn, privateKey, certificate),
		}

		// This shouldn't be used, use the sync package or channels instead
		// This is just a dirty test
		atomic.AddInt32(&version, 1)
		sr.logger.Infof(">>>>>>>>>>>>>>>>>>> creating snapshot Version " + fmt.Sprint(version))
		snap := envoycache.NewSnapshot(fmt.Sprint(version), nil, nil, nil, nil, nil)
		snap.Resources[envoycache.Secret] = envoycache.NewResources(fmt.Sprintf("%v", version), secrets)
		sr.envoyXdsServer.SetSnapshot(&snap, "test-id") // config.SetSnapshot(nodeId, snap)
	}
}
