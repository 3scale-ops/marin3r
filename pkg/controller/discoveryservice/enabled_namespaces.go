package discoveryservice

import (
	"context"
	"fmt"

	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/envoy"
	"github.com/3scale/marin3r/pkg/webhook"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileEnabledNamespaces is in charge of keep the resources that envoy sidecars require available in all
// the active namespaces:
//     - a Secret holding a client certificate for mTLS with the DiscoveryService
//     - a ConfigMap with the envoy bootstrap configuration to allow envoy sidecars to talk to the DiscoveryService
//     - keeps the Namespace object marked as owned by the marin3r instance
func (r *ReconcileDiscoveryService) reconcileEnabledNamespaces(ctx context.Context) (reconcile.Result, error) {
	var err error
	// Reconcile each namespace in the list of enabled namespaces
	for _, ns := range r.ds.Spec.EnabledNamespaces {
		err = r.reconcileEnabledNamespace(ctx, ns)
		// Keep going even if an error is returned
	}

	if err != nil {
		// TODO: this will surface just the last error, change it so if several errors
		// occur in different namespaces all of them are reported to the caller
		return reconcile.Result{}, fmt.Errorf("Failed reconciling enabled namespaces: %s", err)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileDiscoveryService) reconcileEnabledNamespace(ctx context.Context, namespace string) error {

	r.logger.V(1).Info("Reconciling enabled Namespace", "Namespace", namespace)

	ns := &corev1.Namespace{}
	err := r.client.Get(ctx, types.NamespacedName{Name: namespace}, ns)

	if err != nil {
		// Namespace should exist
		return err
	}

	owner, err := isOwner(r.ds, ns)
	if err != nil {
		return err
	}

	if !owner || !hasEnabledLabel(ns) {

		patch := client.MergeFrom(ns.DeepCopy())

		// Init label's map
		if ns.GetLabels() == nil {
			ns.SetLabels(map[string]string{})
		}

		// Set namespace labels
		ns.ObjectMeta.Labels[operatorv1alpha1.DiscoveryServiceEnabledKey] = operatorv1alpha1.DiscoveryServiceEnabledValue
		ns.ObjectMeta.Labels[operatorv1alpha1.DiscoveryServiceLabelKey] = r.ds.GetName()

		if err := r.client.Patch(ctx, ns, patch); err != nil {
			return err
		}
		r.logger.Info("Patched Namespace", "Namespace", namespace)
	}

	if err := r.reconcileClientCertificate(ctx, namespace); err != nil {
		return err
	}

	if err := r.reconcileBootstrapConfigMap(ctx, namespace); err != nil {
		return err
	}

	return nil
}

func isOwner(owner metav1.Object, object metav1.Object) (bool, error) {

	value, ok := object.GetLabels()[operatorv1alpha1.DiscoveryServiceLabelKey]
	if ok {
		if value == owner.GetName() {
			return true, nil
		}
		return false, fmt.Errorf("Namespace already onwed by %s", value)
	}

	return false, nil
}

func hasEnabledLabel(object metav1.Object) bool {

	value, ok := object.GetLabels()[operatorv1alpha1.DiscoveryServiceEnabledKey]
	if ok && value == operatorv1alpha1.DiscoveryServiceEnabledValue {
		return true
	}

	return false
}

func (r *ReconcileDiscoveryService) reconcileClientCertificate(ctx context.Context, namespace string) error {
	r.logger.V(1).Info("Reconciling client certificate", "Namespace", namespace)
	existent := &operatorv1alpha1.DiscoveryServiceCertificate{}
	err := r.client.Get(ctx, types.NamespacedName{Name: webhook.DefaultClientCertificate, Namespace: namespace}, existent)

	if err != nil {
		if errors.IsNotFound(err) {
			existent = r.getClientCertObject(namespace)
			if err := controllerutil.SetControllerReference(r.ds, existent, r.scheme); err != nil {
				return err
			}
			if err := r.client.Create(ctx, existent); err != nil {
				return err
			}
			r.logger.Info("Created client certificate", "Namespace", namespace)
			return nil
		}
		return err
	}

	// Certificates are not currently reconciled

	return nil
}

func (r *ReconcileDiscoveryService) reconcileBootstrapConfigMap(ctx context.Context, namespace string) error {
	r.logger.V(1).Info("Reconciling bootstrap ConfigMap", "Namespace", namespace)
	existent := &corev1.ConfigMap{}
	err := r.client.Get(ctx, types.NamespacedName{Name: webhook.DefaultBootstrapConfigMap, Namespace: namespace}, existent)

	if err != nil {
		if errors.IsNotFound(err) {
			existent, err := r.getBootstrapConfigMapObject(namespace)
			if err != nil {
				return err
			}
			if err := controllerutil.SetControllerReference(r.ds, existent, r.scheme); err != nil {
				return err
			}
			if err := r.client.Create(ctx, existent); err != nil {
				return err
			}
			r.logger.Info("Created bootstrap ConfigMap", "Namespace", namespace)
			return nil
		}
		return err
	}

	// Bootstrap ConfigMap are not currently reconciled

	return nil
}

func (r *ReconcileDiscoveryService) getClientCertObject(namespace string) *operatorv1alpha1.DiscoveryServiceCertificate {
	return &operatorv1alpha1.DiscoveryServiceCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhook.DefaultClientCertificate,
			Namespace: namespace,
		},
		Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
			CommonName: fmt.Sprintf("%s-client", OwnedObjectName(r.ds)),
			ValidFor:   clientCertValidFor,
			Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
				CASigned: &operatorv1alpha1.CASignedConfig{
					SecretRef: corev1.SecretReference{
						Name:      getCACertName(r.ds),
						Namespace: OwnedObjectNamespace(r.ds),
					}},
			},
			SecretRef: corev1.SecretReference{
				Name:      webhook.DefaultClientCertificate,
				Namespace: namespace,
			},
		},
	}
}

func (r *ReconcileDiscoveryService) getBootstrapConfigMapObject(namespace string) (*corev1.ConfigMap, error) {

	config, err := envoy.GenerateBootstrapConfig(getDiscoveryServiceHost(r.ds), getDiscoveryServicePort())
	if err != nil {
		return nil, err
	}

	tlsConfig, err := envoy.GenerateTlsCertificateSdsConfig(getDiscoveryServiceHost(r.ds), getDiscoveryServicePort())
	if err != nil {
		return nil, err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhook.DefaultBootstrapConfigMap,
			Namespace: namespace,
		},
		Data: map[string]string{
			webhook.DefaultEnvoyConfigFileName:      config,
			webhook.TlsCertificateSdsSecretFileName: tlsConfig,
		},
	}

	return cm, nil
}

func getDiscoveryServiceHost(ds *operatorv1alpha1.DiscoveryService) string {
	return fmt.Sprintf("%s.%s.%s", OwnedObjectName(ds), OwnedObjectNamespace(ds), "svc")
}

func getDiscoveryServicePort() uint32 {
	return uint32(18000)
}
