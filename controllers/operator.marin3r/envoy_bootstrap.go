package controllers

import (
	"context"
	"fmt"
	"time"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/webhooks/podv1mutator"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileEnvoyBootstrap is in charge of keep the resources that envoy sidecars require available in all
// the active namespaces:
//     - an EnvoyBootstrap resource
func (r *DiscoveryServiceReconciler) reconcileEnvoyBootstrap(ctx context.Context, log logr.Logger) (reconcile.Result, error) {
	eb := &marin3rv1alpha1.EnvoyBootstrap{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: r.ds.GetName(), Namespace: r.ds.GetNamespace()}, eb); err != nil {

		if errors.IsNotFound(err) {
			eb, err := genEnvoyBootstrapObject(r.ds)
			if err != nil {
				return ctrl.Result{}, err
			}
			if err := controllerutil.SetControllerReference(r.ds, eb, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.Client.Create(ctx, eb); err != nil {
				return ctrl.Result{}, err
			}
			log.Info("Created EnvoyBootstrap", "Name", r.ds.GetName(), "Namespace", r.ds.GetNamespace())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func genEnvoyBootstrapObject(ds *operatorv1alpha1.DiscoveryService) (*marin3rv1alpha1.EnvoyBootstrap, error) {

	duration, err := time.ParseDuration(clientCertValidFor)
	if err != nil {
		return nil, err
	}

	return &marin3rv1alpha1.EnvoyBootstrap{
		ObjectMeta: metav1.ObjectMeta{Name: ds.GetName(), Namespace: ds.GetNamespace()},
		Spec: marin3rv1alpha1.EnvoyBootstrapSpec{
			DiscoveryService: ds.GetName(),
			ClientCertificate: &marin3rv1alpha1.ClientCertificate{
				Directory:  podv1mutator.DefaultEnvoyTLSBasePath,
				SecretName: podv1mutator.DefaultClientCertificate,
				Duration: metav1.Duration{
					Duration: duration,
				},
			},
			EnvoyStaticConfig: &marin3rv1alpha1.EnvoyStaticConfig{
				ConfigMapNameV2:       podv1mutator.DefaultBootstrapConfigMapV2,
				ConfigMapNameV3:       podv1mutator.DefaultBootstrapConfigMapV3,
				ConfigFile:            fmt.Sprintf("%s/%s", podv1mutator.DefaultEnvoyConfigBasePath, podv1mutator.DefaultEnvoyConfigFileName),
				ResourcesDir:          podv1mutator.DefaultEnvoyConfigBasePath,
				RtdsLayerResourceName: "runtime",
				AdminBindAddress:      "0.0.0.0:9901",
				AdminAccessLogPath:    "/dev/null",
			},
		},
	}, nil
}
