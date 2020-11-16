package controllers

import (
	"context"
	"fmt"
	"time"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/webhooks/podv1mutator"
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
func (r *DiscoveryServiceReconciler) reconcileEnabledNamespaces(ctx context.Context) (reconcile.Result, error) {
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

func (r *DiscoveryServiceReconciler) reconcileEnabledNamespace(ctx context.Context, namespace string) error {

	r.Log.V(1).Info("Reconciling enabled Namespace", "Namespace", namespace)

	ns := &corev1.Namespace{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: namespace}, ns)

	if err != nil {
		// Namespace should exist
		return err
	}

	owner, err := isSidecarEnabled(r.ds, ns)
	if err != nil {
		return err
	}

	if !owner {

		patch := client.MergeFrom(ns.DeepCopy())

		// Init label's map
		if ns.GetLabels() == nil {
			ns.SetLabels(map[string]string{})
		}

		// Set namespace labels
		ns.ObjectMeta.Labels[operatorv1alpha1.DiscoveryServiceEnabledKey] = operatorv1alpha1.DiscoveryServiceEnabledValue
		ns.ObjectMeta.Labels[operatorv1alpha1.DiscoveryServiceLabelKey] = r.ds.GetName()

		if err := r.Client.Patch(ctx, ns, patch); err != nil {
			return err
		}
		r.Log.Info("Patched Namespace", "Namespace", namespace)
	}

	eb := &envoyv1alpha1.EnvoyBootstrap{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: r.ds.GetName(), Namespace: namespace}, eb); err != nil {

		if errors.IsNotFound(err) {
			eb, err := genEnvoyBootstrapObject(namespace, r.ds)
			if err != nil {
				return err
			}
			if err := controllerutil.SetControllerReference(r.ds, eb, r.Scheme); err != nil {
				return err
			}
			if err := r.Client.Create(ctx, eb); err != nil {
				return err
			}
			r.Log.Info("Created EnvoyBootstrap", "Name", r.ds.GetName(), "Namespace", namespace)
			return nil
		}
		return err
	}

	return nil
}

func isSidecarEnabled(owner metav1.Object, object metav1.Object) (bool, error) {

	value, ok := object.GetLabels()[operatorv1alpha1.DiscoveryServiceLabelKey]
	if ok {
		if value == owner.GetName() {
			return true, nil
		}
		return false, fmt.Errorf("Namespace already onwed by %s", value)
	}

	return false, nil
}

func genEnvoyBootstrapObject(namespace string, ds *operatorv1alpha1.DiscoveryService) (*envoyv1alpha1.EnvoyBootstrap, error) {

	duration, err := time.ParseDuration("48h")
	if err != nil {
		return nil, err
	}

	return &envoyv1alpha1.EnvoyBootstrap{
		ObjectMeta: metav1.ObjectMeta{Name: ds.GetName(), Namespace: namespace},
		Spec: envoyv1alpha1.EnvoyBootstrapSpec{
			DiscoveryService: ds.GetName(),
			ClientCertificate: &envoyv1alpha1.ClientCertificate{
				Directory:  podv1mutator.DefaultEnvoyTLSBasePath,
				SecretName: podv1mutator.DefaultClientCertificate,
				Duration: metav1.Duration{
					Duration: duration,
				},
			},
			EnvoyStaticConfig: &envoyv1alpha1.EnvoyStaticConfig{
				ConfigMapNameV2:       podv1mutator.DefaultBootstrapConfigMap,
				ConfigMapNameV3:       fmt.Sprintf("%s-v3", podv1mutator.DefaultBootstrapConfigMap),
				ConfigFile:            fmt.Sprintf("%s/%s", podv1mutator.DefaultEnvoyConfigBasePath, podv1mutator.DefaultEnvoyConfigFileName),
				ResourcesDir:          podv1mutator.DefaultEnvoyConfigBasePath,
				RtdsLayerResourceName: "runtime",
				AdminBindAddress:      "0.0.0.0:9901",
				AdminAccessLogPath:    "/dev/null",
			},
		},
	}, nil
}
