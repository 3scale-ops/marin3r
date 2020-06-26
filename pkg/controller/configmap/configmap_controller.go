package configmap

import (
	"context"
	"fmt"
	"time"

	marin3rv1alpha1 "github.com/3scale/marin3r/pkg/apis/marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/envoy"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	nodeIDTag            = "marin3r.3scale.net/node-id"
	configMapKey         = "config.yaml"
	secretAnnotation     = "cert-manager.io/common-name"
	defaultSerialization = "json"
	reconcileInterval    = 30
)

var log = logf.Log.WithName("controller_configmap")

// Add creates a new ConfigMap Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileConfigMap{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("configmap-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	filter := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// ConfigMap has marin3r annotation
			_, ok := e.Meta.GetAnnotations()[nodeIDTag]
			return ok
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// ConfigMap has marin3r annotation
			if _, ok := e.MetaNew.GetAnnotations()[nodeIDTag]; ok {
				// Ignore updates to CR status in which case metadata.ResourceVersion does not change
				return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
				// return ok
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// ConfigMap has marin3r annotation
			_, ok := e.Meta.GetAnnotations()[nodeIDTag]
			return ok
		},
	}

	// Watch for changes to primary obbject ConfigMap
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{}, filter)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileConfigMap implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileConfigMap{}

// ReconcileConfigMap reconciles a ConfigMap object
type ReconcileConfigMap struct {
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ConfigMap object and makes changes based on the state read
// and what is in the ConfigMap.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileConfigMap) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.TODO()
	// Fetch the ConfigMap instance
	cm := &corev1.ConfigMap{}
	err := r.client.Get(ctx, request.NamespacedName, cm)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	nodeID := cm.GetAnnotations()[nodeIDTag]
	reqLogger := log.WithValues(
		"Namespace", request.Namespace,
		"Name", request.Name,
		"NodeID", nodeID)

	reqLogger.Info("Reconciling from ConfigMap")

	// Get corresponding EnvoyConfig
	ecList := &marin3rv1alpha1.EnvoyConfigList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{nodeIDTag: nodeID},
	})
	if err != nil {
		reqLogger.Error(err, "Could not create selector to get cachesalpha1v1.EnvoyConfig resource")
		return reconcile.Result{}, err
	}
	err = r.client.List(ctx, ecList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		// Error reading the resource - requeue the request.
		reqLogger.Error(err, "Error listing cachesalpha1v1.envoyconfigs")
		return reconcile.Result{}, err
	}

	switch count := len(ecList.Items); count {

	case 0:
		// EnvoyConfig resource not found, create it
		reqLogger.Info("Creating new EnvoyConfig")
		ec, err := createEnvoyConfig(ctx, r.client, *cm, nodeID, request.Name, request.Namespace)
		if err != nil {
			reqLogger.Error(err, "Error building new cachesalpha1v1.envoyconfig")
			return reconcile.Result{}, err
		}

		// Set ConfigMap cm as the owner and controller
		if err := controllerutil.SetControllerReference(cm, ec, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Create the object
		err = r.client.Create(ctx, ec)
		if err != nil {
			reqLogger.Error(err, "Error creating new cachesalpha1v1.envoyconfig")
			return reconcile.Result{}, err
		}

	case 1:
		// EnvoyConfig exists, updating it
		ec := &ecList.Items[0]

		// patch operation to update Spec.Version in the cache

		// Populate resources, loaded from ConfigMap data
		resources, err := populateResources(cm.Data[configMapKey])
		if err != nil {
			reqLogger.Error(err, "Error populating resources in the config cache")
			return reconcile.Result{}, err
		}

		// Populate secret resources, referencing the cert-manager created
		// secrets in the current namespace
		secrets, err := populateSecrets(ctx, r.client, request.Namespace)
		if err != nil {
			reqLogger.Error(err, "Error populating secret resources in the config cache")
			return reconcile.Result{}, err
		}
		resources.Secrets = secrets

		// If the set of resources has changed, update the EnvoyConfig
		if fmt.Sprintf("%v", resources) != fmt.Sprintf("%v", ec.Spec.EnvoyResources) {
			reqLogger.Info("Set of resources has changed, updating EnvoyConfig")
			reqLogger.Info(fmt.Sprintf("%v ##### %v", resources, ec.Spec.EnvoyResources))
			patch := client.MergeFrom(ec.DeepCopy())
			ec.Spec.EnvoyResources = resources
			if err := r.client.Patch(ctx, ec, patch); err != nil {
				return reconcile.Result{}, err
			}
		}

	default:
		// There should always be just one cachesalpha1v1.EnvoyConfig per envoy node-id
		if len(ecList.Items) > 1 {
			err := fmt.Errorf("More than 1 EnvoyConfig object found for NodeID '%s', cannot reconcile", nodeID)
			reqLogger.Error(err, "")
			// Don't flood the controlle with reconciles that are likely to fail
			return reconcile.Result{RequeueAfter: 30 * time.Second}, err
		}
	}

	// Trigger a reconcile each 60 seconds to keep secrets in sync
	return reconcile.Result{RequeueAfter: reconcileInterval * time.Second}, nil
}

func createEnvoyConfig(ctx context.Context, c client.Client, cm corev1.ConfigMap, nodeID, name, namespace string) (*marin3rv1alpha1.EnvoyConfig, error) {

	ec := marin3rv1alpha1.EnvoyConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				nodeIDTag: nodeID,
			},
		},
		Spec: marin3rv1alpha1.EnvoyConfigSpec{
			NodeID:        nodeID,
			Serialization: defaultSerialization,
		},
	}

	er, err := populateResources(cm.Data[configMapKey])
	if err != nil {
		return nil, err
	}
	ec.Spec.EnvoyResources = er

	secrets, err := populateSecrets(ctx, c, namespace)
	if err != nil {
		return nil, err
	}
	ec.Spec.EnvoyResources.Secrets = secrets

	return &ec, nil
}

func populateResources(data string) (*marin3rv1alpha1.EnvoyResources, error) {
	// Get envoy resources
	resources, err := envoy.YAMLtoResources([]byte(data))
	if err != nil {
		return nil, err
	}

	er := &marin3rv1alpha1.EnvoyResources{}
	s := envoy.JSON{}

	for _, cluster := range resources.Clusters {
		sr, _ := s.Marshal(cluster)
		er.Clusters = append(er.Clusters,
			marin3rv1alpha1.EnvoyResource{
				Name:  cluster.Name,
				Value: sr,
			})
	}

	for _, listener := range resources.Listeners {
		sr, _ := s.Marshal(listener)
		er.Listeners = append(er.Listeners,
			marin3rv1alpha1.EnvoyResource{
				Name:  listener.Name,
				Value: sr,
			})
	}

	return er, nil
}

func populateSecrets(ctx context.Context, c client.Client, namespace string) ([]marin3rv1alpha1.EnvoySecretResource, error) {
	esrl := []marin3rv1alpha1.EnvoySecretResource{}

	sl := &corev1.SecretList{}
	err := c.List(ctx, sl, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}

	for _, secret := range sl.Items {
		if cn, ok := secret.GetAnnotations()[secretAnnotation]; ok {
			esrl = append(esrl, marin3rv1alpha1.EnvoySecretResource{
				Name: cn,
				Ref: corev1.SecretReference{
					Name:      secret.ObjectMeta.Name,
					Namespace: namespace,
				},
			})
		}

	}

	return esrl, nil

}
