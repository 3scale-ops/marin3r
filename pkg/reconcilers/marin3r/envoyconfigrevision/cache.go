package reconcilers

import (
	"context"
	"fmt"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_resources "github.com/3scale-ops/marin3r/pkg/envoy/resources"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	"github.com/3scale-ops/marin3r/pkg/util"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretCertificate = "tls.crt"
	secretPrivateKey  = "tls.key"
)

type CacheReconciler struct {
	ctx       context.Context
	logger    logr.Logger
	client    client.Client
	xdsCache  xdss.Cache
	decoder   envoy_serializer.ResourceUnmarshaller
	generator envoy_resources.Generator
}

func NewCacheReconciler(ctx context.Context, logger logr.Logger, client client.Client, xdsCache xdss.Cache,
	decoder envoy_serializer.ResourceUnmarshaller, generator envoy_resources.Generator) CacheReconciler {

	return CacheReconciler{ctx, logger, client, xdsCache, decoder, generator}
}

func (r *CacheReconciler) Reconcile(req types.NamespacedName, resources *marin3rv1alpha1.EnvoyResources, nodeID, version string) (ctrl.Result, error) {

	snap, err := r.GenerateSnapshot(req, resources, version)

	if err != nil {
		return ctrl.Result{}, err
	}

	oldSnap, err := r.xdsCache.GetSnapshot(nodeID)
	// Publish the generated snapshot when the version is different from the published one. We look specifically
	// for the version of the "Secret" resources because secrets can change even when the spec hasn't changed.
	// Publish the snapshot when an error retrieving the published one occurs as it means that no snpshot has already
	// been written to the cache for that specific nodeID.
	if snap.GetVersion(envoy.Secret) != oldSnap.GetVersion(envoy.Secret) || err != nil {

		r.logger.Info("Writing new snapshot to xDS cache", "Version", version, "NodeID", nodeID, "Secrets Hash", snap.GetVersion(envoy.Secret))

		if err := r.xdsCache.SetSnapshot(nodeID, snap); err != nil {
			return ctrl.Result{}, err
		}

	} else {
		r.logger.V(1).Info("Snapshot has not changed, skipping writing to xDS cache", "NodeID", nodeID)
	}

	return ctrl.Result{}, nil
}

func (r *CacheReconciler) GenerateSnapshot(req types.NamespacedName, resources *marin3rv1alpha1.EnvoyResources, version string) (xdss.Snapshot, error) {
	snap := r.xdsCache.NewSnapshot(version)

	for idx, endpoint := range resources.Endpoints {
		res := r.generator.New(envoy.Endpoint)
		if err := r.decoder.Unmarshal(endpoint.Value, res); err != nil {
			return nil,
				resourceLoaderError(
					req, endpoint.Value, field.NewPath("spec", "resources").Child("endpoint").Index(idx).Child("value"),
					fmt.Sprintf("Invalid envoy resource value: '%s'", err),
				)
		}
		snap.SetResource(endpoint.Name, res)
	}

	for idx, cluster := range resources.Clusters {
		res := r.generator.New(envoy.Cluster)
		if err := r.decoder.Unmarshal(cluster.Value, res); err != nil {
			return nil,
				resourceLoaderError(
					req, cluster.Value, field.NewPath("spec", "resources").Child("clusters").Index(idx).Child("value"),
					fmt.Sprintf("Invalid envoy resource value: '%s'", err),
				)
		}
		snap.SetResource(cluster.Name, res)
	}

	for idx, route := range resources.Routes {
		res := r.generator.New(envoy.Route)
		if err := r.decoder.Unmarshal(route.Value, res); err != nil {
			return nil,
				resourceLoaderError(
					req, route.Value, field.NewPath("spec", "resources").Child("routes").Index(idx).Child("value"),
					fmt.Sprintf("Invalid envoy resource value: '%s'", err),
				)
		}
		snap.SetResource(route.Name, res)
	}

	for idx, listener := range resources.Listeners {
		res := r.generator.New(envoy.Listener)
		if err := r.decoder.Unmarshal(listener.Value, res); err != nil {
			return nil,
				resourceLoaderError(
					req, listener.Value, field.NewPath("spec", "resources").Child("listener").Index(idx).Child("value"),
					fmt.Sprintf("Invalid envoy resource value: '%s'", err),
				)
		}
		snap.SetResource(listener.Name, res)
	}

	for idx, runtime := range resources.Runtimes {
		res := r.generator.New(envoy.Runtime)
		if err := r.decoder.Unmarshal(runtime.Value, res); err != nil {
			return nil,
				resourceLoaderError(
					req, runtime.Value, field.NewPath("spec", "resources").Child("runtime").Index(idx).Child("value"),
					fmt.Sprintf("Invalid envoy resource value: '%s'", err),
				)
		}
		snap.SetResource(runtime.Name, res)
	}

	for idx, secret := range resources.Secrets {
		s := &corev1.Secret{}
		key := types.NamespacedName{
			Name:      secret.Ref.Name,
			Namespace: secret.Ref.Namespace,
		}
		if err := r.client.Get(r.ctx, key, s); err != nil {
			return nil, fmt.Errorf("%s", err.Error())
		}

		// Validate secret holds a certificate
		if s.Type == "kubernetes.io/tls" {
			res := r.generator.NewSecret(secret.Name, string(s.Data[secretPrivateKey]), string(s.Data[secretCertificate]))
			snap.SetResource(secret.Name, res)
		} else {
			err := resourceLoaderError(
				req, secret.Ref, field.NewPath("spec", "resources").Child("secrets").Index(idx).Child("ref"),
				"Only 'kubernetes.io/tls' type secrets allowed",
			)
			return nil, fmt.Errorf("%s", err.Error())

		}
	}

	// Secrets are runtime calculated resourcesso its contents are not included in the spec. This means
	// that changes in the content of secret resources wont be reflected in the hash of spec.envoyResources.
	// To reflect changes to the content of secrets we append the hash of the runtime calculated secrets to
	// the hash of sepc.envoyResources in the version of secret resources in the snapshot.
	secretsHash := util.Hash(snap.GetResources(envoy.Secret))
	snap.SetVersion(envoy.Secret, fmt.Sprintf("%s-%s", version, secretsHash))

	return snap, nil

}

func resourceLoaderError(req types.NamespacedName, value interface{}, resPath *field.Path, msg string) error {
	return errors.NewInvalid(
		schema.GroupKind{Group: "envoy", Kind: "EnvoyConfig"},
		fmt.Sprintf("%s/%s", req.Namespace, req.Name),
		field.ErrorList{field.Invalid(resPath, value, fmt.Sprint(msg))},
	)
}
