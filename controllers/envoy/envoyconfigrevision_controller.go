/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"hash/fnv"

	common "github.com/3scale/marin3r/pkg/common"
	"github.com/3scale/marin3r/pkg/envoy"

	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoyapi_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoyapi_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoyapi_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	xds_cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/golang/protobuf/proto"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretCertificate = "tls.crt"
	secretPrivateKey  = "tls.key"
)

// EnvoyConfigRevisionReconciler reconciles a EnvoyConfigRevision object
type EnvoyConfigRevisionReconciler struct {
	Client   client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	ADSCache *xds_cache.SnapshotCache
}

// +kubebuilder:rbac:groups=envoy.marin3r.3scale.net,resources=envoyconfigrevisions,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=envoy.marin3r.3scale.net,resources=envoyconfigrevisions/status,verbs=get;update

func (r *EnvoyConfigRevisionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("envoyconfigrevision", req.NamespacedName)
	r.Log.Info("Reconciling EnvoyConfigRevision")

	// Fetch the EnvoyConfigRevision instance
	ecr := &envoyv1alpha1.EnvoyConfigRevision{}
	err := r.Client.Get(ctx, req.NamespacedName, ecr)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// If this ecr has the RevisionPublishedCondition set to "True" pusblish the resources
	// to the xds server cache
	if ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition) {

		nodeID := ecr.Spec.NodeID
		version := ecr.Spec.Version
		snap := newNodeSnapshot(nodeID, version)

		// Deserialize envoy resources from the spec and create a new snapshot with them
		if err := r.loadResources(ctx, req.Name, req.Namespace,
			ecr.Spec.Serialization, ecr.Spec.EnvoyResources, field.NewPath("spec", "resources"), snap); err != nil {
			// Requeue with delay, as the envoy resources syntax is probably wrong
			// and that is not a transitory error (some other higher level resource
			// probaly needs fixing)
			r.Log.Error(err, "Errors occured while loading resources from CR")
			if err := r.taintSelf(ctx, ecr, "FailedLoadingResources", err.Error()); err != nil {
				return ctrl.Result{}, err
			}
			// This is an unrecoverable error because resources are wrong
			// so do not reque
			return ctrl.Result{}, nil
		}

		// Push the snapshot to the xds server cache
		oldSnap, _ := (*r.ADSCache).GetSnapshot(nodeID)
		if !snapshotIsEqual(snap, &oldSnap) {
			// TODO: check snapshot consistency using "snap.Consistent()". Consider also validating
			// consistency between clusters and listeners, as this is not done by snap.Consistent()
			// because listeners and clusters are not requested by name by the envoy gateways

			// Publish the in-memory cache to the envoy control-plane
			r.Log.Info("Publishing new snapshot for nodeID", "Version", version, "NodeID", nodeID)
			if err := (*r.ADSCache).SetSnapshot(nodeID, *snap); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			r.Log.Info("Generated snapshot is equal to published one, avoiding push to xds server cache", "NodeID", nodeID)
		}
	}

	// Update status
	if err := r.updateStatus(ctx, ecr); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *EnvoyConfigRevisionReconciler) loadResources(ctx context.Context, name, namespace, serialization string,
	resources *envoyv1alpha1.EnvoyResources, resPath *field.Path, snap *xds_cache.Snapshot) error {

	var ds envoy.ResourceUnmarshaller
	switch serialization {
	case "b64json":
		ds = envoy.B64JSON{}

	case "yaml":
		ds = envoy.YAML{}
	default:
		// "json" is the default
		ds = envoy.JSON{}
	}

	for idx, endpoint := range resources.Endpoints {
		res := &envoyapi.ClusterLoadAssignment{}
		if err := ds.Unmarshal(endpoint.Value, res); err != nil {
			return resourceLoaderError(name, namespace, "Endpoints", endpoint.Value, resPath, idx)
		}
		setResource(endpoint.Name, res, snap)
	}

	for idx, cluster := range resources.Clusters {
		res := &envoyapi.Cluster{}
		if err := ds.Unmarshal(cluster.Value, res); err != nil {
			return resourceLoaderError(name, namespace, "Clusters", cluster.Value, resPath, idx)
		}
		setResource(cluster.Name, res, snap)
	}

	for idx, route := range resources.Routes {
		res := &envoyapi_route.Route{}
		if err := ds.Unmarshal(route.Value, res); err != nil {
			return resourceLoaderError(name, namespace, "Routes", route.Value, resPath, idx)
		}
		setResource(route.Name, res, snap)
	}

	for idx, listener := range resources.Listeners {
		res := &envoyapi.Listener{}
		if err := ds.Unmarshal(listener.Value, res); err != nil {
			return resourceLoaderError(name, namespace, "Listeners", listener.Value, resPath, idx)
		}
		setResource(listener.Name, res, snap)
	}

	for idx, runtime := range resources.Runtimes {
		res := &envoyapi_discovery.Runtime{}
		if err := ds.Unmarshal(runtime.Value, res); err != nil {
			return resourceLoaderError(name, namespace, "Runtimes", runtime.Value, resPath, idx)
		}
		setResource(runtime.Name, res, snap)
	}

	for idx, secret := range resources.Secrets {
		s := &corev1.Secret{}
		key := types.NamespacedName{
			Name:      secret.Ref.Name,
			Namespace: secret.Ref.Namespace,
		}
		if err := r.Client.Get(ctx, key, s); err != nil {
			if errors.IsNotFound(err) {
				return err
			}
			return err
		}

		// Validate secret holds a certificate
		if s.Type == "kubernetes.io/tls" {
			res := envoy.NewSecret(secret.Name, string(s.Data[secretPrivateKey]), string(s.Data[secretCertificate]))
			setResource(secret.Name, res, snap)
		} else {
			return errors.NewInvalid(
				schema.GroupKind{Group: "caches", Kind: "EnvoyConfig"},
				fmt.Sprintf("%s/%s", namespace, name),
				field.ErrorList{
					field.Invalid(
						resPath.Child("Secrets").Index(idx).Child("Ref"),
						secret.Ref,
						fmt.Sprint("Only 'kubernetes.io/tls' type secrets allowed"),
					),
				},
			)
		}
	}

	// Secrets are a special case of resources as its values are not defined in the spec. This
	// could cause that new values in the same set of secrets wouldn't trigger updates to the envoy
	// gateways as the version (the Spec.Resources hash) would be the same. To avoid this problem, we
	// append the hash of the values of the secrets to the version of the secret resources that will
	// only trigger secret updates in the envoy gateways when necessary
	secretsHash := calculateSecretsHash(snap.Resources[xds_cache_types.Secret].Items)
	snap.Resources[xds_cache_types.Secret].Version = fmt.Sprintf("%s-%s", snap.Resources[xds_cache_types.Secret].Version, secretsHash)

	return nil
}

func resourceLoaderError(name, namespace, rtype, rvalue string, resPath *field.Path, idx int) error {
	return errors.NewInvalid(
		schema.GroupKind{Group: "caches", Kind: "EnvoyConfig"},
		fmt.Sprintf("%s/%s", namespace, name),
		field.ErrorList{
			field.Invalid(
				resPath.Child(rtype).Index(idx).Child("Value"),
				rvalue,
				fmt.Sprint("Invalid envoy resource value"),
			),
		},
	)
}

func (r *EnvoyConfigRevisionReconciler) taintSelf(ctx context.Context, ecr *envoyv1alpha1.EnvoyConfigRevision, reason, msg string) error {
	if !ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionTaintedCondition) {
		patch := client.MergeFrom(ecr.DeepCopy())
		ecr.Status.Conditions.SetCondition(status.Condition{
			Type:    envoyv1alpha1.RevisionTaintedCondition,
			Status:  corev1.ConditionTrue,
			Reason:  status.ConditionReason(reason),
			Message: msg,
		})
		ecr.Status.Tainted = true

		if err := r.Client.Status().Patch(ctx, ecr, patch); err != nil {
			return err
		}
	}
	return nil
}

func (r *EnvoyConfigRevisionReconciler) updateStatus(ctx context.Context, ecr *envoyv1alpha1.EnvoyConfigRevision) error {

	changed := false
	patch := client.MergeFrom(ecr.DeepCopy())

	// Clear ResourcesOutOfSyncCondition
	if ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.ResourcesOutOfSyncCondition) {
		ecr.Status.Conditions.SetCondition(status.Condition{
			Type:    envoyv1alpha1.ResourcesOutOfSyncCondition,
			Reason:  "NodeConficRevisionSynced",
			Status:  corev1.ConditionFalse,
			Message: "EnvoyConfigRevision successfully synced",
		})
		changed = true

	}

	// Set status.published and status.lastPublishedAt fields
	if ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition) && !ecr.Status.Published {
		ecr.Status.Published = true
		ecr.Status.LastPublishedAt = metav1.Now()
		// We also initialise the "tainted" status property to false
		ecr.Status.Tainted = false
		changed = true
	} else if !ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition) && ecr.Status.Published {
		ecr.Status.Published = false
		changed = true
	}

	// Set status.failed field
	if ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionTaintedCondition) && !ecr.Status.Tainted {
		ecr.Status.Tainted = true
		changed = true
	} else if !ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionTaintedCondition) && ecr.Status.Tainted {
		ecr.Status.Tainted = false
		changed = true
	}

	if changed {
		if err := r.Client.Status().Patch(ctx, ecr, patch); err != nil {
			return err
		}
	}

	return nil
}

func newNodeSnapshot(nodeID string, version string) *xds_cache.Snapshot {

	snap := xds_cache.Snapshot{Resources: [6]xds_cache.Resources{}}
	snap.Resources[xds_cache_types.Listener] = xds_cache.NewResources(version, []xds_cache_types.Resource{})
	snap.Resources[xds_cache_types.Endpoint] = xds_cache.NewResources(version, []xds_cache_types.Resource{})
	snap.Resources[xds_cache_types.Cluster] = xds_cache.NewResources(version, []xds_cache_types.Resource{})
	snap.Resources[xds_cache_types.Route] = xds_cache.NewResources(version, []xds_cache_types.Resource{})
	snap.Resources[xds_cache_types.Secret] = xds_cache.NewResources(version, []xds_cache_types.Resource{})
	snap.Resources[xds_cache_types.Runtime] = xds_cache.NewResources(version, []xds_cache_types.Resource{})

	return &snap
}

func setResource(name string, res xds_cache_types.Resource, snap *xds_cache.Snapshot) {

	switch o := res.(type) {

	case *envoyapi.ClusterLoadAssignment:
		snap.Resources[xds_cache_types.Endpoint].Items[name] = o

	case *envoyapi.Cluster:
		snap.Resources[xds_cache_types.Cluster].Items[name] = o

	case *envoyapi_route.Route:
		snap.Resources[xds_cache_types.Route].Items[name] = o

	case *envoyapi.Listener:
		snap.Resources[xds_cache_types.Listener].Items[name] = o

	case *envoyapi_auth.Secret:
		snap.Resources[xds_cache_types.Secret].Items[name] = o

	case *envoyapi_discovery.Runtime:
		snap.Resources[xds_cache_types.Runtime].Items[name] = o

	}
}

func snapshotIsEqual(newSnap, oldSnap *xds_cache.Snapshot) bool {

	// Check resources are equal for each resource type
	for rtype, newResources := range newSnap.Resources {
		oldResources := oldSnap.Resources[rtype]

		// If lenght is not equal, resources are not equal
		if len(oldResources.Items) != len(newResources.Items) {
			return false
		}

		for name, oldValue := range oldResources.Items {

			newValue, ok := newResources.Items[name]

			// If some key does not exist, resources are not equal
			if !ok {
				return false
			}

			// If value has changed, resources are not equal
			if !proto.Equal(oldValue, newValue) {
				return false
			}
		}
	}
	return true
}

func calculateSecretsHash(resources map[string]xds_cache_types.Resource) string {
	resourcesHasher := fnv.New32a()
	common.DeepHashObject(resourcesHasher, resources)
	return rand.SafeEncodeString(fmt.Sprint(resourcesHasher.Sum32()))
}

func (r *EnvoyConfigRevisionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&envoyv1alpha1.EnvoyConfigRevision{}).
		Complete(r)
}
