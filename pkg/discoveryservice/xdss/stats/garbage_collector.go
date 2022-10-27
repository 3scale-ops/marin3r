package stats

import (
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func (s *Stats) RunGC(client kubernetes.Interface, namespace string, stopCh <-chan struct{}) error {

	factory := informers.NewSharedInformerFactoryWithOptions(client, time.Hour*24, informers.WithNamespace(namespace))
	podInformer := factory.Core().V1().Pods()
	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: deleteStatsForDeletedPodsFn(s),
		},
	)

	factory.Start(stopCh)
	if !cache.WaitForNamedCacheSync("podInformer", stopCh, podInformer.Informer().HasSynced) {
		return errors.New("failed to sync")
	}

	return nil

}

func deleteStatsForDeletedPodsFn(s *Stats) func(obj interface{}) {

	return func(obj interface{}) {
		switch o := obj.(type) {
		case *corev1.Pod:
			s.DeleteKeysByFilter(o.GetName())
		}
	}

}
