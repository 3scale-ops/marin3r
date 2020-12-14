package reconcilers

import (
	"context"
	"reflect"
	"testing"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/common"
	xdss "github.com/3scale/marin3r/pkg/discoveryservice/xdss"
	xdss_v2 "github.com/3scale/marin3r/pkg/discoveryservice/xdss/v2"
	xdss_v3 "github.com/3scale/marin3r/pkg/discoveryservice/xdss/v3"
	envoy "github.com/3scale/marin3r/pkg/envoy"
	envoy_resources "github.com/3scale/marin3r/pkg/envoy/resources"
	envoy_resources_v2 "github.com/3scale/marin3r/pkg/envoy/resources/v2"
	envoy_resources_v3 "github.com/3scale/marin3r/pkg/envoy/resources/v3"
	envoy_serializer "github.com/3scale/marin3r/pkg/envoy/serializer"
	testutil "github.com/3scale/marin3r/pkg/util/test"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	envoy_service_runtime_v3 "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func fakeCacheV2() xdss.Cache {
	cache := xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil))
	cache.SetSnapshot("node1", xdss_v2.NewSnapshot(&cache_v2.Snapshot{
		Resources: [6]cache_v2.Resources{
			{Version: "xxxx", Items: map[string]cache_types.Resource{
				"endpoint1": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint1"},
			}},
			{Version: "xxxx", Items: map[string]cache_types.Resource{
				"cluster1": &envoy_api_v2.Cluster{Name: "cluster1"},
			}},
			{Version: "xxxx", Items: map[string]cache_types.Resource{}},
			{Version: "xxxx", Items: map[string]cache_types.Resource{}},
			{Version: "xxxx-557db659d4", Items: map[string]cache_types.Resource{}},
			{Version: "xxxx", Items: map[string]cache_types.Resource{}},
		}}),
	)
	return cache
}

func fakeCacheV3() xdss.Cache {
	cache := xdss_v3.NewCache(cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil))
	cache.SetSnapshot("node1", xdss_v3.NewSnapshot(&cache_v3.Snapshot{
		Resources: [6]cache_v3.Resources{
			{Version: "xxxx", Items: map[string]cache_types.Resource{
				"endpoint1": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint1"},
			}},
			{Version: "xxxx", Items: map[string]cache_types.Resource{
				"cluster1": &envoy_config_cluster_v3.Cluster{Name: "cluster1"},
			}},
			{Version: "xxxx", Items: map[string]cache_types.Resource{}},
			{Version: "xxxx", Items: map[string]cache_types.Resource{}},
			{Version: "xxxx-557db659d4", Items: map[string]cache_types.Resource{}},
			{Version: "xxxx", Items: map[string]cache_types.Resource{}},
		}}),
	)
	return cache
}

func TestNewCacheReconciler(t *testing.T) {
	type args struct {
		ctx       context.Context
		logger    logr.Logger
		client    client.Client
		xdsCache  xdss.Cache
		decoder   envoy_serializer.ResourceUnmarshaller
		generator envoy_resources.Generator
	}
	tests := []struct {
		name string
		args args
		want CacheReconciler
	}{
		{
			name: "Returns a CacheReconciler (v2)",
			args: args{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv2),
				generator: envoy_resources_v2.Generator{},
			},
			want: CacheReconciler{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv2),
				generator: envoy_resources_v2.Generator{},
			},
		},
		{
			name: "Returns a CacheReconciler (v3)",
			args: args{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  xdss_v3.NewCache(cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv3),
				generator: envoy_resources_v3.Generator{},
			},
			want: CacheReconciler{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  xdss_v3.NewCache(cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv3),
				generator: envoy_resources_v3.Generator{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCacheReconciler(tt.args.ctx, tt.args.logger, tt.args.client, tt.args.xdsCache, tt.args.decoder, tt.args.generator); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCacheReconciler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCacheReconciler_Reconcile(t *testing.T) {
	type fields struct {
		ctx       context.Context
		logger    logr.Logger
		client    client.Client
		xdsCache  xdss.Cache
		decoder   envoy_serializer.ResourceUnmarshaller
		generator envoy_resources.Generator
	}
	type args struct {
		req       types.NamespacedName
		resources *marin3rv1alpha1.EnvoyResources
		nodeID    string
		version   string
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		want        ctrl.Result
		wantErr     bool
		wantSnap    xdss.Snapshot
		wantVersion string
	}{
		{
			name: "Reconciles cache (v2)",
			fields: fields{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv2),
				generator: envoy_resources_v2.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Endpoints: []marin3rv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
					}},
				version: "xxxx",
				nodeID:  "node2",
			},

			want:    reconcile.Result{},
			wantErr: false,
			wantSnap: xdss_v2.NewSnapshot(&cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"endpoint": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint"}}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx-557db659d4", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
				}}),
		},
		{
			name: "Reconciles cache (v3)",
			fields: fields{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  xdss_v3.NewCache(cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv3),
				generator: envoy_resources_v3.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Endpoints: []marin3rv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
					}},
				version: "xxxx",
				nodeID:  "node2",
			},

			want:    reconcile.Result{},
			wantErr: false,
			wantSnap: xdss_v3.NewSnapshot(&cache_v3.Snapshot{
				Resources: [6]cache_v3.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx-557db659d4", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
				}}),
		},
		{
			name: "Does not write to cache if secret versions are equal",
			fields: fields{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  fakeCacheV3(),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv3),
				generator: envoy_resources_v3.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Listeners: []marin3rv1alpha1.EnvoyResource{
						{Name: "listener", Value: "{\"name\": \"endpoint\"}"},
					}},
				version: "xxxx",
				nodeID:  "node1",
			},

			want:    reconcile.Result{},
			wantErr: false,
			wantSnap: xdss_v3.NewSnapshot(&cache_v3.Snapshot{
				Resources: [6]cache_v3.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"endpoint1": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint1"},
					}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"cluster1": &envoy_config_cluster_v3.Cluster{Name: "cluster1"},
					}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx-557db659d4", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
				}}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CacheReconciler{
				ctx:       tt.fields.ctx,
				logger:    tt.fields.logger,
				client:    tt.fields.client,
				xdsCache:  tt.fields.xdsCache,
				decoder:   tt.fields.decoder,
				generator: tt.fields.generator,
			}
			got, err := r.Reconcile(tt.args.req, tt.args.resources, tt.args.nodeID, tt.args.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("CacheReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CacheReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
			gotSnap, _ := r.xdsCache.GetSnapshot(tt.args.nodeID)
			if !testutil.SnapshotsAreEqual(gotSnap, tt.wantSnap) {
				t.Errorf("CacheReconciler.GenerateSnapshot() Snapshot = %v, want %v", gotSnap, tt.wantSnap)
			}
		})
	}
}

func TestCacheReconciler_GenerateSnapshot(t *testing.T) {
	type fields struct {
		ctx       context.Context
		logger    logr.Logger
		client    client.Client
		xdsCache  xdss.Cache
		decoder   envoy_serializer.ResourceUnmarshaller
		generator envoy_resources.Generator
	}
	type args struct {
		req       types.NamespacedName
		resources *marin3rv1alpha1.EnvoyResources
		version   string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    xdss.Snapshot
		wantErr bool
	}{
		{
			name: "Loads v2 resources into the snapshot",
			fields: fields{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv2),
				generator: envoy_resources_v2.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Endpoints: []marin3rv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
					},
					Clusters: []marin3rv1alpha1.EnvoyResource{
						{Name: "cluster", Value: "{\"name\": \"cluster\"}"},
					},
					Routes: []marin3rv1alpha1.EnvoyResource{
						{Name: "route", Value: "{\"name\": \"route\"}"},
					},
					Listeners: []marin3rv1alpha1.EnvoyResource{
						{Name: "listener", Value: "{\"name\": \"listener\"}"},
					},
					Runtimes: []marin3rv1alpha1.EnvoyResource{
						{Name: "runtime", Value: "{\"name\": \"runtime\"}"},
					}},
				version: "xxxx",
			},
			want: xdss_v2.NewSnapshot(&cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"endpoint": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"cluster": &envoy_api_v2.Cluster{Name: "cluster"},
					}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"route": &envoy_api_v2.RouteConfiguration{Name: "route"},
					}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"listener": &envoy_api_v2.Listener{Name: "listener"},
					}},
					{Version: "xxxx-557db659d4", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"runtime": &envoy_service_discovery_v2.Runtime{Name: "runtime"},
					}},
				},
			}),
			wantErr: false,
		},
		{
			name: "Loads v3 resources into the snapshot",
			fields: fields{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  xdss_v3.NewCache(cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv3),
				generator: envoy_resources_v3.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Endpoints: []marin3rv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
					},
					Clusters: []marin3rv1alpha1.EnvoyResource{
						{Name: "cluster", Value: "{\"name\": \"cluster\"}"},
					},
					Routes: []marin3rv1alpha1.EnvoyResource{
						{Name: "route", Value: "{\"name\": \"route\"}"},
					},
					Listeners: []marin3rv1alpha1.EnvoyResource{
						{Name: "listener", Value: "{\"name\": \"listener\"}"},
					},
					Runtimes: []marin3rv1alpha1.EnvoyResource{
						{Name: "runtime", Value: "{\"name\": \"runtime\"}"},
					}},
				version: "xxxx",
			},
			want: xdss_v3.NewSnapshot(&cache_v3.Snapshot{
				Resources: [6]cache_v3.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"cluster": &envoy_config_cluster_v3.Cluster{Name: "cluster"},
					}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"route": &envoy_config_route_v3.RouteConfiguration{Name: "route"},
					}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"listener": &envoy_config_listener_v3.Listener{Name: "listener"},
					}},
					{Version: "xxxx-557db659d4", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"runtime": &envoy_service_runtime_v3.Runtime{Name: "runtime"},
					}},
				},
			}),
			wantErr: false,
		},
		{
			name: "Error, bad endpoint value",
			fields: fields{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv2),
				generator: envoy_resources_v2.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Endpoints: []marin3rv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "giberish"},
					}},
				version: "xxxx",
			},
			wantErr: true,
			want:    xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
		},
		{
			name: "Error, bad cluster value",
			fields: fields{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv2),
				generator: envoy_resources_v2.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Clusters: []marin3rv1alpha1.EnvoyResource{
						{Name: "cluster", Value: "giberish"},
					}},
				version: "xxxx",
			},
			wantErr: true,
			want:    xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
		},
		{
			name: "Error, bad route value",
			fields: fields{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv2),
				generator: envoy_resources_v2.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Routes: []marin3rv1alpha1.EnvoyResource{
						{Name: "route", Value: "giberish"},
					}},
				version: "xxxx",
			},
			wantErr: true,
			want:    xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
		},
		{
			name: "Error, bad listener value",
			fields: fields{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv2),
				generator: envoy_resources_v2.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Listeners: []marin3rv1alpha1.EnvoyResource{
						{Name: "listener", Value: "giberish"},
					}},
				version: "xxxx",
			},
			wantErr: true,
			want:    xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
		},
		{
			name: "Error, bad runtime value",
			fields: fields{
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				client:    fake.NewFakeClient(),
				xdsCache:  xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv2),
				generator: envoy_resources_v2.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Runtimes: []marin3rv1alpha1.EnvoyResource{
						{Name: "runtime", Value: "giberish"},
					}},
				version: "xxxx",
			},
			wantErr: true,
			want:    xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
		},
		{
			name: "Loads secret resources into the snapshot (v2)",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClient(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "default"},
					Type:       corev1.SecretTypeTLS,
					Data:       map[string][]byte{"tls.crt": []byte("cert"), "tls.key": []byte("key")},
				}),
				xdsCache:  xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv2),
				generator: envoy_resources_v2.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Secrets: []marin3rv1alpha1.EnvoySecretResource{
						{Name: "secret", Ref: corev1.SecretReference{
							Name:      "secret",
							Namespace: "default",
						}},
					}},
				version: "xxxx",
			},
			wantErr: false,
			want: xdss_v2.NewSnapshot(&cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx-6c68d58f5f", Items: map[string]cache_types.Resource{
						"secret": &envoy_api_v2_auth.Secret{
							Name: "secret",
							Type: &envoy_api_v2_auth.Secret_TlsCertificate{
								TlsCertificate: &envoy_api_v2_auth.TlsCertificate{
									PrivateKey: &envoy_api_v2_core.DataSource{
										Specifier: &envoy_api_v2_core.DataSource_InlineBytes{InlineBytes: []byte("key")},
									},
									CertificateChain: &envoy_api_v2_core.DataSource{
										Specifier: &envoy_api_v2_core.DataSource_InlineBytes{InlineBytes: []byte("cert")},
									}}}}}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
				},
			}),
		},
		{
			name: "Loads secret resources into the snapshot (v3)",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClient(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "default"},
					Type:       corev1.SecretTypeTLS,
					Data:       map[string][]byte{"tls.crt": []byte("cert"), "tls.key": []byte("key")},
				}),
				xdsCache:  xdss_v3.NewCache(cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv3),
				generator: envoy_resources_v3.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Secrets: []marin3rv1alpha1.EnvoySecretResource{
						{Name: "secret", Ref: corev1.SecretReference{
							Name:      "secret",
							Namespace: "default",
						}},
					}},
				version: "xxxx",
			},
			wantErr: false,
			want: xdss_v3.NewSnapshot(&cache_v3.Snapshot{
				Resources: [6]cache_v3.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx-77c9875d7b", Items: map[string]cache_types.Resource{
						"secret": &envoy_extensions_transport_sockets_tls_v3.Secret{
							Name: "secret",
							Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
								TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
									PrivateKey: &envoy_config_core_v3.DataSource{
										Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("key")},
									},
									CertificateChain: &envoy_config_core_v3.DataSource{
										Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("cert")},
									}}}}}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
				},
			}),
		},
		{
			name: "Fails with wrong secret type",
			fields: fields{
				client: fake.NewFakeClient(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "default"},
					Type:       corev1.SecretTypeBasicAuth,
					Data:       map[string][]byte{"tls.crt": []byte("cert"), "tls.key": []byte("key")},
				}),
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				xdsCache:  xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv2),
				generator: envoy_resources_v2.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Secrets: []marin3rv1alpha1.EnvoySecretResource{
						{Name: "secret", Ref: corev1.SecretReference{
							Name:      "secret",
							Namespace: "default",
						}},
					}},
				version: "xxxx",
			},
			wantErr: true,
			want:    xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
		},
		{
			name: "Fails when secret does not exist",
			fields: fields{
				client:    fake.NewFakeClient(),
				ctx:       context.TODO(),
				logger:    ctrl.Log.WithName("test"),
				xdsCache:  xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)),
				decoder:   envoy_serializer.NewResourceUnmarshaller(envoy_serializer.JSON, envoy.APIv2),
				generator: envoy_resources_v2.Generator{},
			},
			args: args{
				req: types.NamespacedName{Name: "xx", Namespace: "xx"},
				resources: &marin3rv1alpha1.EnvoyResources{
					Secrets: []marin3rv1alpha1.EnvoySecretResource{
						{Name: "secret", Ref: corev1.SecretReference{
							Name:      "secret",
							Namespace: "default",
						}},
					}},
				version: "xxxx",
			},
			wantErr: true,
			want:    xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CacheReconciler{
				ctx:       tt.fields.ctx,
				logger:    tt.fields.logger,
				client:    tt.fields.client,
				xdsCache:  tt.fields.xdsCache,
				decoder:   tt.fields.decoder,
				generator: tt.fields.generator,
			}
			got, err := r.GenerateSnapshot(tt.args.req, tt.args.resources, tt.args.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("CacheReconciler.GenerateSnapshot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !testutil.SnapshotsAreEqual(got, tt.want) {
				t.Errorf("CacheReconciler.GenerateSnapshot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_resourceLoaderError(t *testing.T) {
	type args struct {
		req     types.NamespacedName
		value   interface{}
		resPath *field.Path
		msg     string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := resourceLoaderError(tt.args.req, tt.args.value, tt.args.resPath, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("resourceLoaderError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_calculateSecretsHash(t *testing.T) {
	type args struct {
		resources map[string]envoy.Resource
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Calculates the hash of secret resources",
			args: args{
				resources: map[string]envoy.Resource{
					"secret": &envoy_api_v2_auth.Secret{
						Name: "secret",
						Type: &envoy_api_v2_auth.Secret_TlsCertificate{
							TlsCertificate: &envoy_api_v2_auth.TlsCertificate{
								PrivateKey: &envoy_api_v2_core.DataSource{
									Specifier: &envoy_api_v2_core.DataSource_InlineBytes{InlineBytes: []byte("key")},
								},
								CertificateChain: &envoy_api_v2_core.DataSource{
									Specifier: &envoy_api_v2_core.DataSource_InlineBytes{InlineBytes: []byte("cert")},
								}}}}}},
			want: "6c68d58f5f",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := common.Hash(tt.args.resources); got != tt.want {
				t.Errorf("calculateSecretsHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
