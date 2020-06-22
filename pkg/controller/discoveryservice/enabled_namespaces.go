package discoveryservice

import (
	"context"
	"fmt"

	controlplanev1alpha1 "github.com/3scale/marin3r/pkg/apis/controlplane/v1alpha1"
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

	if !owner || !hasEnabledLabel(ns) {
		patch := client.MergeFrom(ns.DeepCopy())
		controllerutil.SetOwnerReference(r.ds, ns, r.scheme)
		if ns.GetLabels() == nil {
			ns.SetLabels(map[string]string{
				controlplanev1alpha1.EnabledNamespaceLabelKey: controlplanev1alpha1.EnabledNamespaceLabelValue,
			})
		} else {
			ns.ObjectMeta.Labels[controlplanev1alpha1.EnabledNamespaceLabelKey] = controlplanev1alpha1.EnabledNamespaceLabelValue
		}
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
	flag := false
	for _, or := range object.GetOwnerReferences() {
		if or.Kind == controlplanev1alpha1.DiscoveryServiceKind {
			// Only a single DiscoveryService can own a namespace at a given point in time
			if or.Name == owner.GetName() {
				flag = true
			} else {
				err := fmt.Errorf("marin3r instance '%s' already enabled in namespace '%s'", or.Name, object)
				return false, err
			}
		}
	}
	return flag, nil
}

func hasEnabledLabel(object metav1.Object) bool {

	value, ok := object.GetLabels()[controlplanev1alpha1.EnabledNamespaceLabelKey]
	if ok && value == controlplanev1alpha1.EnabledNamespaceLabelValue {
		return true
	}

	return false
}

func (r *ReconcileDiscoveryService) reconcileClientCertificate(ctx context.Context, namespace string) error {
	r.logger.V(1).Info("Reconciling client certificate", "Namespace", namespace)
	existent := &controlplanev1alpha1.DiscoveryServiceCertificate{}
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

func (r *ReconcileDiscoveryService) getClientCertObject(namespace string) *controlplanev1alpha1.DiscoveryServiceCertificate {
	return &controlplanev1alpha1.DiscoveryServiceCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhook.DefaultClientCertificate,
			Namespace: namespace,
		},
		Spec: controlplanev1alpha1.DiscoveryServiceCertificateSpec{
			CommonName: r.getName(),
			ValidFor:   clientValidFor,
			Signer: controlplanev1alpha1.DiscoveryServiceCertificateSigner{
				CertManager: &controlplanev1alpha1.CertManagerConfig{
					ClusterIssuer: r.getClusterIssuerName(),
				},
			},
			SecretRef: corev1.SecretReference{
				Name:      webhook.DefaultClientCertificate,
				Namespace: namespace,
			},
		},
	}
}

func (r *ReconcileDiscoveryService) getBootstrapConfigMapObject(namespace string) (*corev1.ConfigMap, error) {

	config, err := getEnvoyBootstrapConfig()
	if err != nil {
		return nil, err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhook.DefaultBootstrapConfigMap,
			Namespace: namespace,
		},
		Data: map[string]string{
			"config.yaml": config,
		},
	}

	return cm, nil
}

func getEnvoyBootstrapConfig() (string, error) {

	return "test", nil
}
