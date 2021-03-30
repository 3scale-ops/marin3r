package reconcilers

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_bootstrap "github.com/3scale-ops/marin3r/pkg/envoy/bootstrap"
	envoy_bootstrap_options "github.com/3scale-ops/marin3r/pkg/envoy/bootstrap/options"
	"github.com/3scale-ops/marin3r/pkg/webhooks/podv1mutator"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// BootstrapConfigReconciler has methods to reconcile discovery service
// client certificates
type BootstrapConfigReconciler struct {
	ctx    context.Context
	logger logr.Logger
	client client.Client
	scheme *runtime.Scheme
	eb     *marin3rv1alpha1.EnvoyBootstrap
}

// NewBootstrapConfigReconciler returns a BootstrapConfigReconciler struct
func NewBootstrapConfigReconciler(ctx context.Context, logger logr.Logger, client client.Client, scheme *runtime.Scheme,
	eb *marin3rv1alpha1.EnvoyBootstrap) BootstrapConfigReconciler {

	return BootstrapConfigReconciler{ctx, logger, client, scheme, eb}
}

// Reconcile keeps a discovery service client certificates in sync with the desired state
func (r *BootstrapConfigReconciler) Reconcile(envoyAPI envoy.APIVersion) (ctrl.Result, error) {

	// Get the DiscoveryService instance this client want to connect to
	ds := &operatorv1alpha1.DiscoveryService{}
	key := types.NamespacedName{Name: r.eb.Spec.DiscoveryService, Namespace: r.eb.GetNamespace()}
	if err := r.client.Get(r.ctx, key, ds); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Error(err, "DiscoveryService does not exist", "DiscoveryService", r.eb.Spec.DiscoveryService)
		}
		return ctrl.Result{}, err
	}

	cmName := r.ConfigMapName(envoyAPI)
	cmNamespace := r.eb.GetNamespace()

	// Get this client's bootstrap ConfigMap
	cm := &corev1.ConfigMap{}
	err := r.client.Get(r.ctx, types.NamespacedName{Name: cmName, Namespace: cmNamespace}, cm)

	if err != nil {
		if errors.IsNotFound(err) {
			cm, err := r.getBootstrapConfigMapObject(ds, envoyAPI)
			if err != nil {
				return ctrl.Result{}, err
			}
			if err := controllerutil.SetControllerReference(r.eb, cm, r.scheme); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.client.Create(r.ctx, cm); err != nil {
				return ctrl.Result{}, err
			}
			r.logger.Info("Created bootstrap ConfigMap", "Name", cmName, "Namespace", cmNamespace)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// This is code only required for upgrading from v0.5.x
	if err := controllerutil.SetControllerReference(r.eb, cm, r.scheme); err != nil {
		switch err.(type) {
		case *controllerutil.AlreadyOwnedError:
			// Create a new controller ref. so the EnvoyBootstrap controller adopts this
			// resource that was in previous versions owned directly by the DiscoveryService
			// controller
			gvk, err := apiutil.GVKForObject(r.eb, r.scheme)
			if err != nil {
				return ctrl.Result{}, err
			}
			ref := metav1.OwnerReference{
				APIVersion:         gvk.GroupVersion().String(),
				Kind:               gvk.Kind,
				Name:               r.eb.GetName(),
				UID:                r.eb.GetUID(),
				BlockOwnerDeletion: pointer.BoolPtr(true),
				Controller:         pointer.BoolPtr(true),
			}
			cm.ObjectMeta.OwnerReferences = []metav1.OwnerReference{ref}
			if err := r.client.Update(r.ctx, cm); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Reconcile the configs in the ConfigMap
	desired, err := r.getBootstrapConfigMapObject(ds, envoyAPI)
	if err != nil {
		return ctrl.Result{}, err
	}

	if equality.Semantic.DeepEqual(desired.Data, cm.Data) {
		patch := client.MergeFrom(cm.DeepCopy())
		cm.Data = desired.Data
		if err := r.client.Patch(r.ctx, cm, patch); err != nil {
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil

}

func (r *BootstrapConfigReconciler) getBootstrapConfigMapObject(ds *operatorv1alpha1.DiscoveryService, envoyAPI envoy.APIVersion) (*corev1.ConfigMap, error) {

	host, port, err := parseBindAddress(r.eb.Spec.EnvoyStaticConfig.AdminBindAddress)
	if err != nil {
		r.logger.Error(err, "Error parsing 'spec.EnvoyStaticConfig.AdminBindAddress'")
	}

	bootstrap := envoy_bootstrap.NewConfig(envoyAPI, envoy_bootstrap_options.ConfigOptions{
		XdsHost:                     fmt.Sprintf("%s.%s.%s", ds.GetServiceConfig().Name, ds.GetNamespace(), "svc"),
		XdsPort:                     ds.GetXdsServerPort(),
		XdsClientCertificatePath:    fmt.Sprintf("%s/%s", r.eb.Spec.ClientCertificate.Directory, corev1.TLSCertKey),
		XdsClientCertificateKeyPath: fmt.Sprintf("%s/%s", r.eb.Spec.ClientCertificate.Directory, corev1.TLSPrivateKeyKey),
		SdsConfigSourcePath:         fmt.Sprintf("%s/%s", r.eb.Spec.EnvoyStaticConfig.ResourcesDir, envoy_bootstrap_options.TlsCertificateSdsSecretFileName),
		RtdsLayerResourceName:       r.eb.Spec.EnvoyStaticConfig.RtdsLayerResourceName,
		AdminAddress:                host,
		AdminPort:                   port,
		AdminAccessLogPath:          r.eb.Spec.EnvoyStaticConfig.AdminAccessLogPath,
	})

	config, err := bootstrap.GenerateStatic()
	if err != nil {
		r.logger.Error(err, "Error generating envoy config'")
		return nil, err
	}

	sdsResources, err := bootstrap.GenerateSdsResources()
	if err != nil {
		r.logger.Error(err, "Error generating envoy client certificate sds config'")
		return nil, err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.ConfigMapName(envoyAPI),
			Namespace: r.eb.GetNamespace(),
		},
		Data: map[string]string{
			podv1mutator.DefaultEnvoyConfigFileName: config,
		},
	}

	for file, content := range sdsResources {
		cm.Data[file] = content
	}

	return cm, nil
}

func parseBindAddress(address string) (string, uint32, error) {

	var err error
	var host string
	var port int

	var parts []string
	if parts = strings.Split(address, ":"); len(parts) != 2 {
		return "", 0, fmt.Errorf("wrong 'spec.envoyStaticConfig.adminBindAddress' specification, expected '<ip>:<port>'")
	}

	host = parts[0]
	if net.ParseIP(host) == nil {
		err := fmt.Errorf("ip address %s is invalid", host)
		return "", 0, err
	}

	if port, err = strconv.Atoi(parts[1]); err != nil {
		return "", 0, fmt.Errorf("unable to parse port value in 'spec.envoyStaticConfig.adminBindAddress'")
	}

	return host, uint32(port), nil
}

func (r *BootstrapConfigReconciler) ConfigMapName(envoyAPI envoy.APIVersion) string {
	if envoyAPI == envoy.APIv2 {
		return r.eb.Spec.EnvoyStaticConfig.ConfigMapNameV2
	}
	return r.eb.Spec.EnvoyStaticConfig.ConfigMapNameV3
}
