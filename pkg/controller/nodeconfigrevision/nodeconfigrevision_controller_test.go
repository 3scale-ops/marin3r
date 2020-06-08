package nodeconfigrevision

import (
	"context"
	"reflect"
	"testing"

	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoyapi_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoyapi_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoyapi_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoyapi_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	xds_cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func fakeTestCache() *xds_cache.SnapshotCache {

	snapshotCache := xds_cache.NewSnapshotCache(true, xds_cache.IDHash{}, nil)

	snapshotCache.SetSnapshot("node1", xds_cache.Snapshot{
		Resources: [6]xds_cache.Resources{
			{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
				"endpoint1": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint1"},
			}},
			{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
				"cluster1": &envoyapi.Cluster{Name: "cluster1"},
			}},
			{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
			{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
			{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
			{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
		}},
	)

	return &snapshotCache
}

func TestReconcileNodeConfigRevision_Reconcile(t *testing.T) {

	tests := []struct {
		name        string
		nodeID      string
		cr          *cachesv1alpha1.NodeConfigRevision
		wantResult  reconcile.Result
		wantSnap    *xds_cache.Snapshot
		wantVersion string
		wantErr     bool
	}{
		{
			name:   "Creates new snapshot for nodeID",
			nodeID: "node3",
			cr: &cachesv1alpha1.NodeConfigRevision{
				ObjectMeta: metav1.ObjectMeta{Name: "ncr", Namespace: "default"},
				Spec: cachesv1alpha1.NodeConfigRevisionSpec{
					NodeID:  "node3",
					Version: "xxxx",
					Resources: &cachesv1alpha1.EnvoyResources{
						Endpoints: []cachesv1alpha1.EnvoyResource{
							{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
						}}},
				Status: cachesv1alpha1.NodeConfigRevisionStatus{
					Conditions: status.NewConditions(status.Condition{
						Type:   cachesv1alpha1.RevisionPublishedCondition,
						Status: corev1.ConditionTrue,
					})},
			},
			wantResult: reconcile.Result{},
			wantSnap: &xds_cache.Snapshot{Resources: [6]xds_cache.Resources{
				{Version: "xxxx", Items: map[string]xds_cache_types.Resource{
					"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"}}},
				{Version: "xxxx", Items: map[string]xds_cache_types.Resource{}},
				{Version: "xxxx", Items: map[string]xds_cache_types.Resource{}},
				{Version: "xxxx", Items: map[string]xds_cache_types.Resource{}},
				{Version: "xxxx-74d569cc4", Items: map[string]xds_cache_types.Resource{}},
				{Version: "xxxx", Items: map[string]xds_cache_types.Resource{}},
			}},
			wantVersion: "xxxx",
			wantErr:     false,
		},
		{
			name:   "Does not update snapshot if resources don't change",
			nodeID: "node1",
			cr: &cachesv1alpha1.NodeConfigRevision{
				ObjectMeta: metav1.ObjectMeta{Name: "ncr", Namespace: "default"},
				Spec: cachesv1alpha1.NodeConfigRevisionSpec{
					NodeID:  "node1",
					Version: "bbbb",
					Resources: &cachesv1alpha1.EnvoyResources{
						Endpoints: []cachesv1alpha1.EnvoyResource{
							{Name: "endpoint1", Value: "{\"cluster_name\": \"endpoint1\"}"},
						},
						Clusters: []cachesv1alpha1.EnvoyResource{
							{Name: "cluster1", Value: "{\"name\": \"cluster1\"}"},
						},
					}},
				Status: cachesv1alpha1.NodeConfigRevisionStatus{
					Conditions: status.NewConditions(status.Condition{
						Type:   cachesv1alpha1.RevisionPublishedCondition,
						Status: corev1.ConditionTrue,
					})},
			},
			wantResult: reconcile.Result{},
			wantSnap: &xds_cache.Snapshot{
				Resources: [6]xds_cache.Resources{
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"endpoint1": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint1"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
				}},
			wantVersion: "aaaa",
			wantErr:     false,
		},
		// {
		// 	name:   "Error and requeue with delay when cannot load resources",
		// 	nodeID: "node1",
		// 	cr: &cachesv1alpha1.NodeConfigRevision{
		// 		ObjectMeta: metav1.ObjectMeta{Name: "ncr", Namespace: "default"},
		// 		Spec: cachesv1alpha1.NodeConfigRevisionSpec{
		// 			NodeID:  "node1",
		// 			Version: "bbbb",
		// 			Resources: &cachesv1alpha1.EnvoyResources{
		// 				Endpoints: []cachesv1alpha1.EnvoyResource{
		// 					{Name: "endpoint1", Value: "giberish"},
		// 				},
		// 			}},
		// 		Status: cachesv1alpha1.NodeConfigRevisionStatus{
		// 			Conditions: status.NewConditions(status.Condition{
		// 				Type:   cachesv1alpha1.RevisionPublishedCondition,
		// 				Status: corev1.ConditionTrue,
		// 			})},
		// 	},
		// 	wantResult:  reconcile.Result{RequeueAfter: 30 * time.Second},
		// 	wantSnap:    &xds_cache.Snapshot{},
		// 	wantVersion: "-",
		// 	wantErr:     true,
		// },
		{
			name:   "No changes to xds server cache when ncr has condition 'cachesv1alpha1.RevisionPublishedCondition' to false",
			nodeID: "node1",
			cr: &cachesv1alpha1.NodeConfigRevision{
				ObjectMeta: metav1.ObjectMeta{Name: "ncr", Namespace: "default"},
				Spec: cachesv1alpha1.NodeConfigRevisionSpec{
					NodeID:    "node1",
					Version:   "bbbb",
					Resources: &cachesv1alpha1.EnvoyResources{},
				},
				Status: cachesv1alpha1.NodeConfigRevisionStatus{
					Conditions: status.NewConditions(status.Condition{
						Type:   cachesv1alpha1.RevisionPublishedCondition,
						Status: corev1.ConditionFalse,
					})},
			},
			wantResult: reconcile.Result{},
			wantSnap: &xds_cache.Snapshot{
				Resources: [6]xds_cache.Resources{
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"endpoint1": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint1"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
				}},
			wantVersion: "aaaa",
			wantErr:     false,
		},
		{
			name:   "No changes to xds server cache when ncr has condition 'cachesv1alpha1.RevisionPublishedCondition' not present",
			nodeID: "node1",
			cr: &cachesv1alpha1.NodeConfigRevision{
				ObjectMeta: metav1.ObjectMeta{Name: "ncr", Namespace: "default"},
				Spec: cachesv1alpha1.NodeConfigRevisionSpec{
					NodeID:    "node1",
					Version:   "bbbb",
					Resources: &cachesv1alpha1.EnvoyResources{},
				},
			},
			wantResult: reconcile.Result{},
			wantSnap: &xds_cache.Snapshot{
				Resources: [6]xds_cache.Resources{
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"endpoint1": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint1"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
						"cluster1": &envoyapi.Cluster{Name: "cluster1"},
					}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
				}},
			wantVersion: "aaaa",
			wantErr:     false,
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			s := scheme.Scheme
			s.AddKnownTypes(cachesv1alpha1.SchemeGroupVersion, tt.cr)
			cl := fake.NewFakeClient(tt.cr)
			r := &ReconcileNodeConfigRevision{client: cl, scheme: s, adsCache: fakeTestCache()}
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "ncr",
					Namespace: "default",
				},
			}

			gotResult, gotErr := r.Reconcile(req)
			gotSnap, _ := (*r.adsCache).GetSnapshot(tt.nodeID)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("ReconcileNodeConfigRevision.Reconcile() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("ReconcileNodeConfigRevision.Reconcile() = %v, want %v", gotResult, tt.wantResult)
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

func TestReconcileNodeConfigRevision_loadResources(t *testing.T) {
	type fields struct {
		client   client.Client
		scheme   *runtime.Scheme
		adsCache *xds_cache.SnapshotCache
	}
	type args struct {
		ctx           context.Context
		name          string
		namespace     string
		serialization string
		resources     *cachesv1alpha1.EnvoyResources
		snap          *xds_cache.Snapshot
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
				ctx:           context.TODO(),
				name:          "ncr",
				namespace:     "default",
				serialization: "json",
				resources: &cachesv1alpha1.EnvoyResources{
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
					}},
				snap: newNodeSnapshot("node1", "1"),
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
					{Version: "1-74d569cc4", Items: map[string]xds_cache_types.Resource{}},
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
				ctx:           context.TODO(),
				name:          "ncr",
				namespace:     "default",
				serialization: "json",
				resources: &cachesv1alpha1.EnvoyResources{
					Endpoints: []cachesv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "giberish"},
					}},
				snap: newNodeSnapshot("node1", "1"),
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
				ctx:           context.TODO(),
				name:          "ncr",
				namespace:     "default",
				serialization: "json",
				resources: &cachesv1alpha1.EnvoyResources{
					Clusters: []cachesv1alpha1.EnvoyResource{
						{Name: "cluster", Value: "giberish"},
					}},
				snap: newNodeSnapshot("node1", "1"),
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
				ctx:           context.TODO(),
				name:          "ncr",
				namespace:     "default",
				serialization: "json",
				resources: &cachesv1alpha1.EnvoyResources{
					Routes: []cachesv1alpha1.EnvoyResource{
						{Name: "route", Value: "giberish"},
					}},
				snap: newNodeSnapshot("node1", "1"),
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
				ctx:           context.TODO(),
				name:          "ncr",
				namespace:     "default",
				serialization: "json",
				resources: &cachesv1alpha1.EnvoyResources{
					Listeners: []cachesv1alpha1.EnvoyResource{
						{Name: "listener", Value: "giberish"},
					}},
				snap: newNodeSnapshot("node1", "1"),
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
				ctx:           context.TODO(),
				name:          "ncr",
				namespace:     "default",
				serialization: "json",
				resources: &cachesv1alpha1.EnvoyResources{
					Runtimes: []cachesv1alpha1.EnvoyResource{
						{Name: "runtime", Value: "giberish"},
					}},
				snap: newNodeSnapshot("node1", "1"),
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
				ctx:           context.TODO(),
				name:          "ncr",
				namespace:     "default",
				serialization: "json",
				resources: &cachesv1alpha1.EnvoyResources{
					Secrets: []cachesv1alpha1.EnvoySecretResource{
						{Name: "secret", Ref: corev1.SecretReference{
							Name:      "secret",
							Namespace: "default",
						}},
					}},
				snap: newNodeSnapshot("node1", "1"),
			},
			wantErr: false,
			wantSnap: &xds_cache.Snapshot{
				Resources: [6]xds_cache.Resources{
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1", Items: map[string]xds_cache_types.Resource{}},
					{Version: "1-6cf7fd9d65", Items: map[string]xds_cache_types.Resource{
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
				ctx:           context.TODO(),
				name:          "ncr",
				namespace:     "default",
				serialization: "json",
				resources: &cachesv1alpha1.EnvoyResources{
					Secrets: []cachesv1alpha1.EnvoySecretResource{
						{Name: "secret", Ref: corev1.SecretReference{
							Name:      "secret",
							Namespace: "default",
						}},
					}},
				snap: newNodeSnapshot("node1", "1"),
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
				ctx:           context.TODO(),
				name:          "ncr",
				namespace:     "default",
				serialization: "json",
				resources: &cachesv1alpha1.EnvoyResources{
					Secrets: []cachesv1alpha1.EnvoySecretResource{
						{Name: "secret", Ref: corev1.SecretReference{
							Name:      "secret",
							Namespace: "default",
						}},
					}},
				snap: newNodeSnapshot("node1", "1"),
			},
			wantErr:  true,
			wantSnap: &xds_cache.Snapshot{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileNodeConfigRevision{
				client:   tt.fields.client,
				scheme:   tt.fields.scheme,
				adsCache: tt.fields.adsCache,
			}
			if err := r.loadResources(tt.args.ctx, tt.args.name, tt.args.namespace, tt.args.serialization, tt.args.resources, field.NewPath("spec", "resources"), tt.args.snap); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileNodeConfigRevision.loadResources() error = %v, wantErr %v", err, tt.wantErr)
			} else if !tt.wantErr && !reflect.DeepEqual(tt.args.snap, tt.wantSnap) {
				t.Errorf("ReconcileNodeConfigRevision.loadResources() got = %v, want %v", tt.args.snap, tt.wantSnap)
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
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
						}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
					},
				},
				oldSnap: &xds_cache.Snapshot{
					Resources: [6]xds_cache.Resources{
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
						}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
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
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
						}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
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
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
							"cluster2": &envoyapi.Cluster{Name: "cluster2"},
						}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
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
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
							"cluster2": &envoyapi.Cluster{Name: "cluster2"},
						}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
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
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
							"endpoint": &envoyapi.ClusterLoadAssignment{ClusterName: "endpoint"},
						}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{
							"cluster1": &envoyapi.Cluster{Name: "cluster1"},
							"cluster2": &envoyapi.Cluster{Name: "cluster2"},
						}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
						{Version: "aaaa", Items: map[string]xds_cache_types.Resource{}},
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
