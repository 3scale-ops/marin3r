package discoveryservice

import (
	"context"
	"fmt"
	"time"

	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/reconcilers"
	"github.com/go-logr/logr"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// as there is currently no renewal mechanism for the CA
	// set a validity sufficiently high. This might be configurable
	// in the future when renewal is managed by the operator
	caCertValidFor             int64  = 3600 * 24 * 365 * 3 // 3 years
	serverCertValidFor         int64  = 3600 * 24 * 90      // 90 days
	clientCertValidFor         int64  = 3600 * 48           // 48 hours
	caCommonName               string = "marin3r-ca"
	caCertSecretNamePrefix     string = "marin3r-ca-cert"
	serverCommonName           string = "marin3r-server"
	serverCertSecretNamePrefix string = "marin3r-server-cert"
)

var (
	log = logf.Log.WithName("controller_dicoveryservice")
)

// Add creates a new DiscoveryService Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDiscoveryService{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("discoveryservice-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource DiscoveryService
	err = c.Watch(&source.Kind{Type: &operatorv1alpha1.DiscoveryService{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for DiscoveryServiceCertificate resources owned by the DiscoveryService resource
	err = c.Watch(&source.Kind{Type: &operatorv1alpha1.DiscoveryServiceCertificate{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.DiscoveryService{},
	})
	if err != nil {
		return err
	}

	// Watch for Deployment resources owned by the DiscoveryService resource
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.DiscoveryService{},
	})
	if err != nil {
		return err
	}

	// Watch for Service resources owned by the DiscoveryService resource
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.DiscoveryService{},
	})
	if err != nil {
		return err
	}

	// Watch for ClusterRole resources owned by the DiscoveryService resource
	err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRole{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.DiscoveryService{},
	})
	if err != nil {
		return err
	}

	// Watch for ClusterRoleBinding resources owned by the DiscoveryService resource
	err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRoleBinding{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.DiscoveryService{},
	})
	if err != nil {
		return err
	}

	// Watch for ServiceAccount resources owned by the DiscoveryService resource
	err = c.Watch(&source.Kind{Type: &corev1.ServiceAccount{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.DiscoveryService{},
	})
	if err != nil {
		return err
	}

	// Watch for MutatingAdmissonConfiguration resources owned by the DiscoveryService resource
	err = c.Watch(&source.Kind{Type: &admissionregistrationv1.MutatingWebhookConfiguration{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.DiscoveryService{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileDiscoveryService implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDiscoveryService{}

// ReconcileDiscoveryService reconciles a DiscoveryService object
// This is not currently thread safe, so use just one worker for
// the controller
type ReconcileDiscoveryService struct {
	client client.Client
	scheme *runtime.Scheme
	ds     *operatorv1alpha1.DiscoveryService
	logger logr.Logger
}

// Reconcile reads that state of the cluster for a DiscoveryService object and makes changes based on the state read
// and what is in the DiscoveryService.Spec
func (r *ReconcileDiscoveryService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.logger = log.WithValues("Request.Name", request.Name)
	r.logger.Info("Reconciling DiscoveryService")
	ctx := context.Background()

	// Fetch the DiscoveryService instance
	dsList := &operatorv1alpha1.DiscoveryServiceList{}
	err := r.client.List(ctx, dsList)
	if err != nil {
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if len(dsList.Items) == 0 {
		// Request object not found, could have been deleted after reconcile request.
		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
		// Return and don't requeue
		return reconcile.Result{}, nil
	}

	if len(dsList.Items) > 1 {
		err := fmt.Errorf("More than one DiscoveryService object in the cluster, refusing to reconcile")
		r.logger.Error(err, "Only one marin3r installation per cluster is supported")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, err
	}

	// Call reconcilers in the proper installation order
	var result reconcile.Result
	r.ds = &dsList.Items[0]

	result, err = r.reconcileCA(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileServerCertificate(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileServiceAccount(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileClusterRole(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileClusterRoleBinding(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	dr := reconcilers.NewDeploymentReconciler(ctx, r.logger, r.client, r.scheme, r.ds)
	result, err = dr.Reconcile(
		types.NamespacedName{Name: OwnedObjectName(r.ds), Namespace: OwnedObjectNamespace(r.ds)},
		deploymentGeneratorFn(r.ds),
	)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileService(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	// TODO: mechanism to cleanup resorces from namespaces
	// TODO: finalizer to cleanup labels in namespaces
	// This is necessary because namespaces are not
	// resources owned by this controller so the usual
	// garbage collection mechanisms won't
	result, err = r.reconcileEnabledNamespaces(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileMutatingWebhook(ctx)
	if result.Requeue || err != nil {
		return result, err
	}

	// Manage server restart when condition is active
	if r.ds.Status.Conditions.IsTrueFor(operatorv1alpha1.ServerRestartRequiredCondition) {

		// TODO: add an admin port to the DiscoveryService server so an http call can be done
		// to indicate that the server must shutdown or reload. It is a much safer approach.
		podList := &corev1.PodList{}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{appLabelKey: OwnedObjectAppLabel(r.ds)},
		})
		if err := r.client.List(ctx, podList, &client.ListOptions{LabelSelector: selector}); err != nil {
			return reconcile.Result{}, err
		}
		for _, pod := range podList.Items {
			err = r.client.Delete(ctx, &pod, client.GracePeriodSeconds(5))
		}
		if err != nil {
			return reconcile.Result{}, err
		}

		// Clear condition
		patch := client.MergeFrom(r.ds.DeepCopy())
		r.ds.Status.Conditions.RemoveCondition(operatorv1alpha1.ServerRestartRequiredCondition)
		if err := r.client.Status().Patch(ctx, r.ds, patch); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}
