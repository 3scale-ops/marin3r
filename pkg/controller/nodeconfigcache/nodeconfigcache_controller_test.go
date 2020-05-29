package nodeconfigcache

import (
	"context"
	"reflect"
	"testing"
	"time"

	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
	"github.com/3scale/marin3r/pkg/envoy"
	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoyapi_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoyapi_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoyapi_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoyapi_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	xds_cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func fakeTestCache() *xds_cache.SnapshotCache {

	snapshotCache := xds_cache.NewSnapshotCache(true, xds_cache.IDHash{}, nil)

	snapshotCache.SetSnapshot("node1", xds_cache.Snapshot{
		Resources: [6]xds_cache.Resources{
			{Version: "43", Items: map[string]xds_cache_types.Resource{
				"endpoint1": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint1"},
			}},
			{Version: "43", Items: map[string]xds_cache_types.Resource{
				"cluster1": &envoyapi.Cluster{Name: "cluster1"},
			}},
			{Version: "43", Items: map[string]xds_cache_types.Resource{}},
			{Version: "43", Items: map[string]xds_cache_types.Resource{}},
			{Version: "43", Items: map[string]xds_cache_types.Resource{}},
			{Version: "43", Items: map[string]xds_cache_types.Resource{}},
		}},
	)

	return &snapshotCache
}

func TestReconcileNodeConfigCache_loadResources(t *testing.T) {
	type fields struct {
		client   client.Client
		scheme   *runtime.Scheme
		adsCache *xds_cache.SnapshotCache
	}
	type args struct {
		ctx    context.Context
		nodeID string
		o      *cachesv1alpha1.NodeConfigCache
		snap   *xds_cache.Snapshot
		ds     envoy.ResourceUnmarshaller
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantErr  bool
		wantSnap *xds_cache.Snapshot
	}{
		{
			name: "Loads resources into the snapshot",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				adsCache: fakeTestCache(),
			},
			args: args{
				ctx:    context.TODO(),
				nodeID: "node1",
				o: &cachesv1alpha1.NodeConfigCache{
					ObjectMeta: metav1.ObjectMeta{Name: "ncc"},
					Spec: cachesv1alpha1.NodeConfigCacheSpec{
						NodeID:  "node1",
						Version: "1",
						Resources: &cachesv1alpha1.EnvoyResources{
							Endpoints: []cachesv1alpha1.EnvoyResource{
								{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
							},
							Clusters: []cachesv1alpha1.EnvoyResource{
								{Name: "cluster", Value: "{\"name\": \"cluster\"}"},
							},
							Routes: []cachesv1alpha1.EnvoyResource{
								{Name: "route", Value: "{\"name\": \"route\"}"},
							},
							Listeners: []cachesv1alpha1.EnvoyResource{
								{Name: "listener", Value: "{\"name\": \"listener\"}"},
							},
							Runtimes: []cachesv1alpha1.EnvoyResource{
								{Name: "runtime", Value: "{\"name\": \"runtime\"}"},
							}}}},
				snap: newNodeSnapshot("node1", "1"),
				ds:   envoy.YAML{},
			},
			wantErr: false,
			wantSnap: &xds_cache.Snapshot{
				Resources: [6]xds_cache.Resources{
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"cluster": &envoyapi.Cluster{Name: "cluster"},
					}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"route": &envoyapi_route.Route{Name: "route"},
					}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"listener": &envoyapi.Listener{Name: "listener"},
					}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"runtime": &envoyapi_discovery.Runtime{Name: "runtime"},
					}},
				},
			},
		},
		{
			name: "Error, bad endpoint value",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				adsCache: fakeTestCache(),
			},
			args: args{
				ctx:    context.TODO(),
				nodeID: "node1",
				o: &cachesv1alpha1.NodeConfigCache{
					ObjectMeta: metav1.ObjectMeta{Name: "ncc"},
					Spec: cachesv1alpha1.NodeConfigCacheSpec{
						NodeID:  "node1",
						Version: "1",
						Resources: &cachesv1alpha1.EnvoyResources{
							Endpoints: []cachesv1alpha1.EnvoyResource{
								{Name: "endpoint", Value: "giberish"},
							}}}},
				snap: newNodeSnapshot("node1", "1"),
				ds:   envoy.YAML{},
			},
			wantErr:  true,
			wantSnap: &xds_cache.Snapshot{},
		},
		{
			name: "Error, bad cluster value",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				adsCache: fakeTestCache(),
			},
			args: args{
				ctx:    context.TODO(),
				nodeID: "node1",
				o: &cachesv1alpha1.NodeConfigCache{
					ObjectMeta: metav1.ObjectMeta{Name: "ncc"},
					Spec: cachesv1alpha1.NodeConfigCacheSpec{
						NodeID:  "node1",
						Version: "1",
						Resources: &cachesv1alpha1.EnvoyResources{
							Clusters: []cachesv1alpha1.EnvoyResource{
								{Name: "cluster", Value: "giberish"},
							}}}},
				snap: newNodeSnapshot("node1", "1"),
				ds:   envoy.YAML{},
			},
			wantErr:  true,
			wantSnap: &xds_cache.Snapshot{},
		},
		{
			name: "Error, bad route value",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				adsCache: fakeTestCache(),
			},
			args: args{
				ctx:    context.TODO(),
				nodeID: "node1",
				o: &cachesv1alpha1.NodeConfigCache{
					ObjectMeta: metav1.ObjectMeta{Name: "ncc"},
					Spec: cachesv1alpha1.NodeConfigCacheSpec{
						NodeID:  "node1",
						Version: "1",
						Resources: &cachesv1alpha1.EnvoyResources{
							Routes: []cachesv1alpha1.EnvoyResource{
								{Name: "route", Value: "giberish"},
							}}}},
				snap: newNodeSnapshot("node1", "1"),
				ds:   envoy.YAML{},
			},
			wantErr:  true,
			wantSnap: &xds_cache.Snapshot{},
		},
		{
			name: "Error, bad listener value",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				adsCache: fakeTestCache(),
			},
			args: args{
				ctx:    context.TODO(),
				nodeID: "node1",
				o: &cachesv1alpha1.NodeConfigCache{
					ObjectMeta: metav1.ObjectMeta{Name: "ncc"},
					Spec: cachesv1alpha1.NodeConfigCacheSpec{
						NodeID:  "node1",
						Version: "1",
						Resources: &cachesv1alpha1.EnvoyResources{
							Listeners: []cachesv1alpha1.EnvoyResource{
								{Name: "listener", Value: "giberish"},
							}}}},
				snap: newNodeSnapshot("node1", "1"),
				ds:   envoy.YAML{},
			},
			wantErr:  true,
			wantSnap: &xds_cache.Snapshot{},
		},
		{
			name: "Error, bad runtime value",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				adsCache: fakeTestCache(),
			},
			args: args{
				ctx:    context.TODO(),
				nodeID: "node1",
				o: &cachesv1alpha1.NodeConfigCache{
					ObjectMeta: metav1.ObjectMeta{Name: "ncc"},
					Spec: cachesv1alpha1.NodeConfigCacheSpec{
						NodeID:  "node1",
						Version: "1",
						Resources: &cachesv1alpha1.EnvoyResources{
							Runtimes: []cachesv1alpha1.EnvoyResource{
								{Name: "runtime", Value: "giberish"},
							}}}},
				snap: newNodeSnapshot("node1", "1"),
				ds:   envoy.YAML{},
			},
			wantErr:  true,
			wantSnap: &xds_cache.Snapshot{},
		},
		{
			name: "Loads secret resources into the snapshot",
			fields: fields{
				client: fake.NewFakeClient(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "default"},
					Type:       corev1.SecretTypeTLS,
					Data:       map[string][]byte{"tls.crt": []byte("cert"), "tls.key": []byte("key")},
				}),
				scheme:   scheme.Scheme,
				adsCache: fakeTestCache(),
			},
			args: args{
				ctx:    context.TODO(),
				nodeID: "node1",
				o: &cachesv1alpha1.NodeConfigCache{
					ObjectMeta: metav1.ObjectMeta{Name: "ncc"},
					Spec: cachesv1alpha1.NodeConfigCacheSpec{
						NodeID:  "node1",
						Version: "1",
						Resources: &cachesv1alpha1.EnvoyResources{
							Secrets: []cachesv1alpha1.EnvoySecretResource{
								{Name: "secret", Ref: corev1.SecretReference{
									Name:      "secret",
									Namespace: "default",
								}},
							}}}},
				snap: newNodeSnapshot("node1", "1"),
				ds:   envoy.YAML{},
			},
			wantErr: false,
			wantSnap: &xds_cache.Snapshot{
				Resources: [6]xds_cache.Resources{
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{
						"secret": &envoyapi_auth.Secret{
							Name: "secret",
							Type: &envoyapi_auth.Secret_TlsCertificate{
								TlsCertificate: &envoyapi_auth.TlsCertificate{
									PrivateKey: &envoyapi_core.DataSource{
										Specifier: &envoyapi_core.DataSource_InlineBytes{InlineBytes: []byte("key")},
									},
									CertificateChain: &envoyapi_core.DataSource{
										Specifier: &envoyapi_core.DataSource_InlineBytes{InlineBytes: []byte("cert")},
									}}}}}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
				},
			},
		},
		{
			name: "Fails with wrong secret type",
			fields: fields{
				client: fake.NewFakeClient(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "default"},
					Type:       corev1.SecretTypeBasicAuth,
					Data:       map[string][]byte{"tls.crt": []byte("cert"), "tls.key": []byte("key")},
				}),
				scheme:   scheme.Scheme,
				adsCache: fakeTestCache(),
			},
			args: args{
				ctx:    context.TODO(),
				nodeID: "node1",
				o: &cachesv1alpha1.NodeConfigCache{
					ObjectMeta: metav1.ObjectMeta{Name: "ncc"},
					Spec: cachesv1alpha1.NodeConfigCacheSpec{
						NodeID:  "node1",
						Version: "1",
						Resources: &cachesv1alpha1.EnvoyResources{
							Secrets: []cachesv1alpha1.EnvoySecretResource{
								{Name: "secret", Ref: corev1.SecretReference{
									Name:      "secret",
									Namespace: "default",
								}},
							}}}},
				snap: newNodeSnapshot("node1", "1"),
				ds:   envoy.YAML{},
			},
			wantErr:  true,
			wantSnap: &xds_cache.Snapshot{},
		},
		{
			name: "Fails when secret does not exist",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				adsCache: fakeTestCache(),
			},
			args: args{
				ctx:    context.TODO(),
				nodeID: "node1",
				o: &cachesv1alpha1.NodeConfigCache{
					ObjectMeta: metav1.ObjectMeta{Name: "ncc"},
					Spec: cachesv1alpha1.NodeConfigCacheSpec{
						NodeID:  "node1",
						Version: "1",
						Resources: &cachesv1alpha1.EnvoyResources{
							Secrets: []cachesv1alpha1.EnvoySecretResource{
								{Name: "secret", Ref: corev1.SecretReference{
									Name:      "secret",
									Namespace: "default",
								}},
							}}}},
				snap: newNodeSnapshot("node1", "1"),
				ds:   envoy.YAML{},
			},
			wantErr:  true,
			wantSnap: &xds_cache.Snapshot{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileNodeConfigCache{
				client:   tt.fields.client,
				scheme:   tt.fields.scheme,
				adsCache: tt.fields.adsCache,
			}
			if err := r.loadResources(tt.args.ctx, tt.args.nodeID, tt.args.o, tt.args.snap, tt.args.ds); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileNodeConfigCache.loadResources() error = %v, wantErr %v", err, tt.wantErr)
			} else if !tt.wantErr && !reflect.DeepEqual(tt.args.snap, tt.wantSnap) {
				t.Errorf("ReconcileNodeConfigCache.loadResources() got = %v, want %v", tt.args.snap, tt.wantSnap)
			}
		})
	}
}

func Test_newNodeSnapshot(t *testing.T) {
	type args struct {
		nodeID  string
		version string
	}
	tests := []struct {
		name string
		args args
		want *xds_cache.Snapshot
	}{
		{
			name: "Generates new empty snapshot",
			args: args{nodeID: "node1", version: "5"},
			want: &xds_cache.Snapshot{
				Resources: [6]xds_cache.Resources{
					{Version: "5", Items: map[string]xds_cache_types.Resource{}},
					{Version: "5", Items: map[string]xds_cache_types.Resource{}},
					{Version: "5", Items: map[string]xds_cache_types.Resource{}},
					{Version: "5", Items: map[string]xds_cache_types.Resource{}},
					{Version: "5", Items: map[string]xds_cache_types.Resource{}},
					{Version: "5", Items: map[string]xds_cache_types.Resource{}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newNodeSnapshot(tt.args.nodeID, tt.args.version); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newNodeSnapshot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_setResource(t *testing.T) {
	type args struct {
		name string
		res  xds_cache_types.Resource
		snap *xds_cache.Snapshot
	}
	tests := []struct {
		name string
		args args
		want *xds_cache.Snapshot
	}{
		{
			name: "Adds envoy resource to the snapshot",
			args: args{
				name: "cluster3",
				res:  &envoyapi.Cluster{Name: "cluster3"},
				snap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "789", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "789", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
							"cluster2": &envoyapi.Cluster{Name: "cluster2"},
						}},
						{Version: "789", Items: map[string]xds_cache_types.Resource{}},
						{Version: "789", Items: map[string]xds_cache_types.Resource{
							"listener1": &envoyapi.Listener{Name: "listener1"},
						}},
						{Version: "789", Items: map[string]xds_cache_types.Resource{}},
						{Version: "789", Items: map[string]xds_cache_types.Resource{}},
					},
				},
			},
			want: &xds_cache.Snapshot{
				Resources: [6]xds_cache.Resources{
					{Version: "789", Items: map[string]xds_cache_types.Resource{
						"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "789", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
						"cluster2": &envoyapi.Cluster{Name: "cluster2"},
						"cluster3": &envoyapi.Cluster{Name: "cluster3"},
					}},
					{Version: "789", Items: map[string]xds_cache_types.Resource{}},
					{Version: "789", Items: map[string]xds_cache_types.Resource{
						"listener1": &envoyapi.Listener{Name: "listener1"},
					}},
					{Version: "789", Items: map[string]xds_cache_types.Resource{}},
					{Version: "789", Items: map[string]xds_cache_types.Resource{}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setResource(tt.args.name, tt.args.res, tt.args.snap)
			if !reflect.DeepEqual(tt.args.snap, tt.want) {
				t.Errorf("setResource() = %v, want %v", tt.args.snap, tt.want)
			}
		})
	}
}

func Test_snapshotIsEqual(t *testing.T) {
	type args struct {
		newSnap *xds_cache.Snapshot
		oldSnap *xds_cache.Snapshot
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Returns true if snapshot resources are equal",
			args: args{
				newSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					},
				},
				oldSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					},
				},
			},
			want: true,
		},
		{
			name: "Returns true if snapshot resources are equal, even with different versions",
			args: args{
				newSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					},
				},
				oldSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					},
				},
			},
			want: true,
		},
		{
			name: "Returns false, different number of resources",
			args: args{
				newSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
							"cluster2": &envoyapi.Cluster{Name: "cluster2"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					},
				},
				oldSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					},
				},
			},
			want: false,
		},
		{
			name: "Returns false, different resource name",
			args: args{
				newSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
							"cluster2": &envoyapi.Cluster{Name: "cluster2"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					},
				},
				oldSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"cluster1":  &envoyapi.Cluster{Name: "cluster1"},
							"different": &envoyapi.Cluster{Name: "cluster2"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					},
				},
			},
			want: false,
		},
		{
			name: "Returns false, different proto message",
			args: args{
				newSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
							"cluster2": &envoyapi.Cluster{Name: "cluster2"},
						}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
						{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					},
				},
				oldSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
							"cluster2": &envoyapi.Cluster{Name: "different"},
						}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
						{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := snapshotIsEqual(tt.args.newSnap, tt.args.oldSnap); got != tt.want {
				t.Errorf("snapshotIsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileNodeConfigCache_Reconcile(t *testing.T) {

	tests := []struct {
		name        string
		nodeID      string
		cr          *cachesv1alpha1.NodeConfigCache
		wantResult  reconcile.Result
		wantSnap    *xds_cache.Snapshot
		wantVersion string
		wantErr     bool
	}{
		{
			name:   "Creates new snapshot for nodeID",
			nodeID: "node3",
			cr: &cachesv1alpha1.NodeConfigCache{
				ObjectMeta: metav1.ObjectMeta{Name: "ncc", Namespace: "default"},
				Spec: cachesv1alpha1.NodeConfigCacheSpec{
					NodeID:  "node3",
					Version: "1",
					Resources: &cachesv1alpha1.EnvoyResources{
						Endpoints: []cachesv1alpha1.EnvoyResource{
							{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
						}}}},
			wantResult: reconcile.Result{},
			wantSnap: &xds_cache.Snapshot{Resources: [6]xds_cache.Resources{
				{Version: "1", Items: map[string]xds_cache_types.Resource{
					"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"}}},
				{Version: "1", Items: map[string]xds_cache_types.Resource{}},
				{Version: "1", Items: map[string]xds_cache_types.Resource{}},
				{Version: "1", Items: map[string]xds_cache_types.Resource{}},
				{Version: "1", Items: map[string]xds_cache_types.Resource{}},
				{Version: "1", Items: map[string]xds_cache_types.Resource{}},
			}},
			wantVersion: "1",
			wantErr:     false,
		},
		{
			name:   "Does not update snapshot if resources don't change",
			nodeID: "node1",
			cr: &cachesv1alpha1.NodeConfigCache{
				ObjectMeta: metav1.ObjectMeta{Name: "ncc", Namespace: "default"},
				Spec: cachesv1alpha1.NodeConfigCacheSpec{
					NodeID:  "node1",
					Version: "44",
					Resources: &cachesv1alpha1.EnvoyResources{
						Endpoints: []cachesv1alpha1.EnvoyResource{
							{Name: "endpoint1", Value: "{\"cluster_name\": \"endpoint1\"}"},
						},
						Clusters: []cachesv1alpha1.EnvoyResource{
							{Name: "cluster1", Value: "{\"name\": \"cluster1\"}"},
						},
					}}},
			wantResult: reconcile.Result{},
			wantSnap: &xds_cache.Snapshot{
				Resources: [6]xds_cache.Resources{
					{Version: "43", Items: map[string]xds_cache_types.Resource{
						"endpoint1": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint1"},
					}},
					{Version: "43", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
					}},
					{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					{Version: "43", Items: map[string]xds_cache_types.Resource{}},
					{Version: "43", Items: map[string]xds_cache_types.Resource{}},
				}},
			wantVersion: "43",
			wantErr:     false,
		},
		{
			name:   "Error and requeue with delay when cannot load resources",
			nodeID: "node1",
			cr: &cachesv1alpha1.NodeConfigCache{
				ObjectMeta: metav1.ObjectMeta{Name: "ncc", Namespace: "default"},
				Spec: cachesv1alpha1.NodeConfigCacheSpec{
					NodeID:  "node1",
					Version: "44",
					Resources: &cachesv1alpha1.EnvoyResources{
						Endpoints: []cachesv1alpha1.EnvoyResource{
							{Name: "endpoint1", Value: "giberish"},
						},
					}}},
			wantResult:  reconcile.Result{RequeueAfter: 30 * time.Second},
			wantSnap:    &xds_cache.Snapshot{},
			wantVersion: "-",
			wantErr:     true,
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			s := scheme.Scheme
			s.AddKnownTypes(cachesv1alpha1.SchemeGroupVersion,
				tt.cr,
				&cachesv1alpha1.NodeConfigRevisionList{},
				&cachesv1alpha1.NodeConfigRevision{},
			)
			cl := fake.NewFakeClient(tt.cr)
			r := &ReconcileNodeConfigCache{client: cl, scheme: s, adsCache: fakeTestCache()}
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "ncc",
					Namespace: "default",
				},
			}

			gotResult, gotErr := r.Reconcile(req)
			gotSnap, _ := (*r.adsCache).GetSnapshot(tt.nodeID)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("ReconcileNodeConfigCache.Reconcile() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("ReconcileNodeConfigCache.Reconcile() = %v, want %v", gotResult, tt.wantResult)
			}
			if !tt.wantErr && !reflect.DeepEqual(&gotSnap, tt.wantSnap) {
				t.Errorf("Snapshot = %v, want %v", &gotSnap, tt.wantSnap)
			}
			// NOTE: we are keep the same version for all resource types
			gotVersion := gotSnap.GetVersion("type.googleapis.com/envoy.api.v2.ClusterLoadAssignment")
			if !tt.wantErr && gotVersion != tt.wantVersion {
				t.Errorf("Snapshot version = %v, want %v", gotVersion, tt.wantVersion)
			}
		})
	}
}

func TestReconcileNodeConfigCache_Reconcile_Finalizer(t *testing.T) {

	cr := &cachesv1alpha1.NodeConfigCache{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ncc",
			Namespace:         "default",
			DeletionTimestamp: func() *metav1.Time { t := metav1.Now(); return &t }(),
			Finalizers:        []string{nodeconfigcacheFinalizer},
		},
		Spec: cachesv1alpha1.NodeConfigCacheSpec{
			NodeID:    "node1",
			Version:   "43",
			Resources: &cachesv1alpha1.EnvoyResources{},
		}}

	s := scheme.Scheme
	s.AddKnownTypes(cachesv1alpha1.SchemeGroupVersion,
		cr,
		&cachesv1alpha1.NodeConfigRevisionList{},
		&cachesv1alpha1.NodeConfigRevision{},
	)
	cl := fake.NewFakeClient(cr)
	r := &ReconcileNodeConfigCache{client: cl, scheme: s, adsCache: fakeTestCache()}
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "ncc",
			Namespace: "default",
		},
	}

	_, gotErr := r.Reconcile(req)

	if gotErr != nil {
		t.Errorf("ReconcileNodeConfigCache.Reconcile_Finalizer() error = %v", gotErr)
		return
	}
	_, err := (*r.adsCache).GetSnapshot(cr.Spec.NodeID)
	if err == nil {
		t.Errorf("ReconcileNodeConfigCache.Reconcile_Finalizer() - snapshot still exists in the ads server cache")
		return
	}

	ncc := &cachesv1alpha1.NodeConfigCache{}
	cl.Get(context.TODO(), types.NamespacedName{Name: "ncc", Namespace: "default"}, ncc)
	if len(ncc.GetObjectMeta().GetFinalizers()) != 0 {
		t.Errorf("ReconcileNodeConfigCache.Reconcile_Finalizer() - finalizer not deleted from object")
		return
	}

}

func TestReconcileNodeConfigCache_finalizeNodeConfigCache(t *testing.T) {
	type fields struct {
		client   client.Client
		scheme   *runtime.Scheme
		adsCache *xds_cache.SnapshotCache
	}
	type args struct {
		nodeID string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Deletes the snapshot from the ads server cache",
			fields: fields{client: fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				adsCache: fakeTestCache(),
			},
			args: args{"node1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileNodeConfigCache{
				client:   tt.fields.client,
				scheme:   tt.fields.scheme,
				adsCache: tt.fields.adsCache,
			}
			r.finalizeNodeConfigCache(tt.args.nodeID)
			if _, err := (*r.adsCache).GetSnapshot(tt.args.nodeID); err == nil {
				t.Errorf("TestReconcileNodeConfigCache_finalizeNodeConfigCache() -> snapshot still in the cache")
			}
		})
	}
}

func TestReconcileNodeConfigCache_addFinalizer(t *testing.T) {
	tests := []struct {
		name    string
		cr      *cachesv1alpha1.NodeConfigCache
		wantErr bool
	}{
		{
			name: "Adds finalizer to NodecacheConfig",
			cr: &cachesv1alpha1.NodeConfigCache{
				ObjectMeta: metav1.ObjectMeta{Name: "ncc", Namespace: "default"},
				Spec: cachesv1alpha1.NodeConfigCacheSpec{
					NodeID:    "node1",
					Version:   "1",
					Resources: &cachesv1alpha1.EnvoyResources{},
				}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scheme.Scheme
			s.AddKnownTypes(cachesv1alpha1.SchemeGroupVersion, tt.cr)
			cl := fake.NewFakeClient(tt.cr)
			r := &ReconcileNodeConfigCache{client: cl, scheme: s, adsCache: fakeTestCache()}

			if err := r.addFinalizer(context.TODO(), tt.cr); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileNodeConfigCache.addFinalizer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				ncc := &cachesv1alpha1.NodeConfigCache{}
				r.client.Get(context.TODO(), types.NamespacedName{Name: "ncc", Namespace: "default"}, ncc)
				if len(ncc.ObjectMeta.Finalizers) != 1 {
					t.Error("ReconcileNodeConfigCache.addFinalizer() wrong number of finalizers present in object")
				}
			}
		})
	}
}

func Test_contains(t *testing.T) {
	type args struct {
		list []string
		s    string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "True -> key in slice",
			args: args{list: []string{"a", "b", "c"}, s: "a"},
			want: true,
		},
		{
			name: "False -> key not in slice",
			args: args{list: []string{"a", "b", "c"}, s: "z"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.args.list, tt.args.s); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
