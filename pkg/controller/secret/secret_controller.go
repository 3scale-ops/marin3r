package secret

import (
	"context"

	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	secretCertificate = "tls.crt"
	secretPrivateKey  = "tls.key"
)

var log = logf.Log.WithName("controller_secret")

// Add creates a new Secret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSecret{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("secret-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	filter := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			if e.Object.(*corev1.Secret).Type == "kubernetes.io/tls" {
				return true
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectNew.(*corev1.Secret).Type == "kubernetes.io/tls" {
				return true
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			if e.Object.(*corev1.Secret).Type == "kubernetes.io/tls" {
				return true
			}
			return false
		},
	}

	// Watch for changes to primary resource Secret
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{}, filter)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileSecret implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSecret{}

// ReconcileSecret reconciles a Secret object
type ReconcileSecret struct {
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Secret object and makes changes based on the state read
// and what is in the Secret.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSecret) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.TODO()
	// Fetch the Secret instance
	secret := &corev1.Secret{}
	err := r.client.Get(ctx, request.NamespacedName, secret)
	if err != nil {
		// Error reading the object - requeue the request.
		// NOTE: We skip the IsNotFound error because we want to trigger NodeConfigCache
		// reconciles when referred secrets are deleted so the envoy control-plane
		// stops publishing them. This might cause errors if the reference hasn't been
		// removed from the NodeCacheConfig, but that's ok as we do want to surface this
		// inconsistency instead of silently failing
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
	}

	logger := log.WithValues("Namespace", request.Namespace, "Name", request.Name)
	logger.Info("Reconciling from 'kubernetes.io/tls' Secret")

	// Get the list of NoceConfigRevisions published and
	// check which of them contain refs to this secret
	list := &cachesv1alpha1.NodeConfigRevisionList{}
	if err := r.client.List(ctx, list); err != nil {
		return reconcile.Result{}, err
	}

	for _, ncr := range list.Items {

		if ncr.Status.Conditions.IsTrueFor(cachesv1alpha1.RevisionPublishedCondition) {

			for _, secret := range ncr.Spec.Resources.Secrets {
				if secret.Ref.Name == request.Name && secret.Ref.Namespace == request.Namespace {
					logger.Info("Triggered NodeConfigRevision reconcile",
						"NodeConfigRevision_Name", ncr.ObjectMeta.Name, "NodeConfigRevision_Namespace", ncr.ObjectMeta.Namespace)
					if err != nil {
						return reconcile.Result{}, err
					}

					if !ncr.Status.Conditions.IsTrueFor(cachesv1alpha1.ResourcesOutOfSyncCondition) {
						// patch operation to update Spec.Version in the cache
						patch := client.MergeFrom(ncr.DeepCopy())
						ncr.Status.Conditions.SetCondition(status.Condition{
							Type:    cachesv1alpha1.ResourcesOutOfSyncCondition,
							Reason:  "SecretChanged",
							Message: "A secret relevant to this nodeconfigrevision changed",
							Status:  corev1.ConditionTrue,
						})
						if err := r.client.Status().Patch(ctx, &ncr, patch); err != nil {
							return reconcile.Result{}, err
						}
						logger.V(1).Info("Condition should have been added ...")
					}
				}
			}
		}
	}

	return reconcile.Result{}, nil
}
