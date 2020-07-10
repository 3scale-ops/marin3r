package discoveryservice

import (
	"context"
	"fmt"
	"time"

	"github.com/3scale/marin3r/pkg/apis/external"
	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/reconcilers"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	// cert-manager
	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
)

const (
	// as there is currently no renewal mechanism for the CA
	// set a validity sufficiently high. This might be configurable
	// in the future when renewal is managed by the operator
	caValidFor                 int64         = 94610000 // 3 years
	serverValidFor             int64         = 31536000 // 1 year
	clientValidFor             int64         = 7776000  // 90 days
	caCommonName               string        = "marin3r"
	caCertSecretNamePrefix     string        = "marin3r-ca-cert"
	serverCertSecretNamePrefix string        = "marin3r-server-cert"
	pollingPeriod              time.Duration = 10
)

var log = logf.Log.WithName("controller_dicoveryservice")

// Add creates a new DiscoveryService Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	dc, _ := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	return &ReconcileDiscoveryService{
		client:             mgr.GetClient(),
		scheme:             mgr.GetScheme(),
		discoveryClient:    dc,
		clusterIssuerWatch: false,
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

	// Watch for Namespace resources owned by the DiscoveryService resource
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.DiscoveryService{},
	})
	if err != nil {
		return err
	}

	// Set up a goroutine to autodetect if required 3rd party apis are available
	go func() {

		discoverFn := func() {
			rec := r.(*ReconcileDiscoveryService)
			resourceExists, _ := external.HasCertManagerClusterIssuer(rec.discoveryClient)

			if resourceExists && !rec.clusterIssuerWatch {
				err := c.Watch(&source.Kind{Type: &certmanagerv1alpha2.ClusterIssuer{}}, &handler.EnqueueRequestForOwner{
					IsController: true,
					OwnerType:    &operatorv1alpha1.DiscoveryService{},
				})
				if err != nil {
					log.Error(err, "Failed setting a watch on certmanagerv1alpha2.ClusterIssuer type")
				} else {
					// Mark the watch was correctly set
					log.Info("Discovered certmanagerv1alpha2 api, watching type 'ClusterIssuer'")
					// WARNING: this is not thread safe
					rec.clusterIssuerWatch = true
				}
			}
		}

		ticker := time.NewTicker(pollingPeriod * time.Second)

		discoverFn()
		for range ticker.C {
			discoverFn()
		}
	}()

	return nil
}

// blank assignment to verify that ReconcileDiscoveryService implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDiscoveryService{}

// ReconcileDiscoveryService reconciles a DiscoveryService object
// This is not currently thread safe, so use just one worker for
// the controller
type ReconcileDiscoveryService struct {
	client             client.Client
	scheme             *runtime.Scheme
	ds                 *operatorv1alpha1.DiscoveryService
	logger             logr.Logger
	discoveryClient    discovery.DiscoveryInterface
	clusterIssuerWatch bool
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

	result, err = r.reconcileSigner(ctx)
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

	// result, err = r.reconcileDeployment(ctx)
	// if result.Requeue || err != nil {
	// 	return result, err
	// }
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

	return reconcile.Result{}, nil
}
