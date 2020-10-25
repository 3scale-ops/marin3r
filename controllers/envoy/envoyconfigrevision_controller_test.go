package controllers

import (
	"context"
	"reflect"
	"testing"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoyapi_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoyapi_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoyapi_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoyapi_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	xds_cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var s *runtime.Scheme = scheme.Scheme

func init() {
	s.AddKnownTypes(envoyv1alpha1.GroupVersion,
		&envoyv1alpha1.EnvoyConfigRevision{},
		&envoyv1alpha1.EnvoyConfigRevisionList{},
		&envoyv1alpha1.EnvoyConfig{},
	)
}

func TestEnvoyConfigRevisionReconciler_Reconcile(t *testing.T) {

	tests := []struct {
		name        string
		nodeID      string
		cr          *envoyv1alpha1.EnvoyConfigRevision
		wantResult  reconcile.Result
		wantSnap    *xds_cache.Snapshot
		wantVersion string
		wantErr     bool
	}{
		{
			name:   "Creates new snapshot for nodeID",
			nodeID: "node3",
			cr: &envoyv1alpha1.EnvoyConfigRevision{
				ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "default"},
				Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
					NodeID:  "node3",
					Version: "xxxx",
					EnvoyResources: &envoyv1alpha1.EnvoyResources{
						Endpoints: []envoyv1alpha1.EnvoyResource{
							{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
						}}},
				Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
					Conditions: status.NewConditions(status.Condition{
						Type:   envoyv1alpha1.RevisionPublishedCondition,
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
			cr: &envoyv1alpha1.EnvoyConfigRevision{
				ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "default"},
				Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
					NodeID:  "node1",
					Version: "bbbb",
					EnvoyResources: &envoyv1alpha1.EnvoyResources{
						Endpoints: []envoyv1alpha1.EnvoyResource{
							{Name: "endpoint1", Value: "{\"cluster_name\": \"endpoint1\"}"},
						},
						Clusters: []envoyv1alpha1.EnvoyResource{
							{Name: "cluster1", Value: "{\"name\": \"cluster1\"}"},
						},
					}},
				Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
					Conditions: status.NewConditions(status.Condition{
						Type:   envoyv1alpha1.RevisionPublishedCondition,
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
		{
			name:   "No changes to xds server cache when ecr has condition 'envoyv1alpha1.RevisionPublishedCondition' to false",
			nodeID: "node1",
			cr: &envoyv1alpha1.EnvoyConfigRevision{
				ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "default"},
				Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
					NodeID:         "node1",
					Version:        "bbbb",
					EnvoyResources: &envoyv1alpha1.EnvoyResources{},
				},
				Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
					Conditions: status.NewConditions(status.Condition{
						Type:   envoyv1alpha1.RevisionPublishedCondition,
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
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			r := &EnvoyConfigRevisionReconciler{
				Client:   fake.NewFakeClient(tt.cr),
				Scheme:   s,
				ADSCache: fakeTestCache(),
				Log:      ctrl.Log.WithName("test"),
			}
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "ecr",
					Namespace: "default",
				},
			}

			gotResult, gotErr := r.Reconcile(req)
			gotSnap, _ := (*r.ADSCache).GetSnapshot(tt.nodeID)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("EnvoyConfigRevisionReconciler.Reconcile() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("EnvoyConfigRevisionReconciler.Reconcile() = %v, want %v", gotResult, tt.wantResult)
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

	t.Run("No error if ecr not found", func(t *testing.T) {
		r := &EnvoyConfigRevisionReconciler{
			Client:   fake.NewFakeClient(),
			Scheme:   s,
			ADSCache: fakeTestCache(),
			Log:      ctrl.Log.WithName("test"),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ecr",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("EnvoyConfigRevisionReconciler.Reconcile() error = %v", gotErr)
			return
		}
	})

	t.Run("Taints itself if it fails to load resources", func(t *testing.T) {
		ecr := &envoyv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "default"},
			Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
				NodeID:  "node1",
				Version: "xxxx",
				EnvoyResources: &envoyv1alpha1.EnvoyResources{
					Endpoints: []envoyv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"wrong_property\": \"abcd\"}"},
					}}},
			Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
				Conditions: status.NewConditions(status.Condition{
					Type:   envoyv1alpha1.RevisionPublishedCondition,
					Status: corev1.ConditionTrue,
				})},
		}

		r := &EnvoyConfigRevisionReconciler{
			Client:   fake.NewFakeClient(ecr),
			Scheme:   s,
			ADSCache: fakeTestCache(),
			Log:      ctrl.Log.WithName("test"),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ecr",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("EnvoyConfigRevisionReconciler.Reconcile() error = %v", gotErr)
			return
		}

		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ecr", Namespace: "default"}, ecr)
		if !ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionTaintedCondition) {
			t.Errorf("EnvoyConfigRevisionReconciler.Reconcile() ecr has not been tainted")
		}
	})
}

func TestEnvoyConfigRevisionReconciler_taintSelf(t *testing.T) {

	t.Run("Taints the ecr object", func(t *testing.T) {
		ecr := &envoyv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "default"},
			Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
				NodeID:         "node1",
				Version:        "bbbb",
				EnvoyResources: &envoyv1alpha1.EnvoyResources{},
			},
		}
		r := &EnvoyConfigRevisionReconciler{
			Client:   fake.NewFakeClient(ecr),
			Scheme:   s,
			ADSCache: fakeTestCache(),
			Log:      ctrl.Log.WithName("test"),
		}
		if err := r.taintSelf(context.TODO(), ecr, "test", "test"); err != nil {
			t.Errorf("EnvoyConfigRevisionReconciler.taintSelf() error = %v", err)
		}
		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ecr", Namespace: "default"}, ecr)
		if !ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionTaintedCondition) {
			t.Errorf("EnvoyConfigRevisionReconciler.taintSelf() ecr is not tainted")
		}
	})
}

func TestEnvoyConfigRevisionReconciler_updateStatus(t *testing.T) {
	t.Run("Updates the status of the ecr object", func(t *testing.T) {
		ecr := &envoyv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "default"},
			Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
				NodeID:         "node1",
				Version:        "bbbb",
				EnvoyResources: &envoyv1alpha1.EnvoyResources{},
			},
			Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
				Conditions: status.NewConditions(
					status.Condition{
						Type:   envoyv1alpha1.ResourcesOutOfSyncCondition,
						Status: corev1.ConditionTrue,
					},
				),
			},
		}
		r := &EnvoyConfigRevisionReconciler{
			Client:   fake.NewFakeClient(ecr),
			Scheme:   s,
			ADSCache: fakeTestCache(),
			Log:      ctrl.Log.WithName("test"),
		}
		if err := r.updateStatus(context.TODO(), ecr); err != nil {
			t.Errorf("EnvoyConfigRevisionReconciler.updateStatus() error = %v", err)
		}
		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ecr", Namespace: "default"}, ecr)
		if ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.ResourcesOutOfSyncCondition) {
			t.Errorf("EnvoyConfigRevisionReconciler.updateStatus() status not updated")
		}
	})
}

func TestEnvoyConfigRevisionReconciler_loadResources(t *testing.T) {
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
		resources     *envoyv1alpha1.EnvoyResources
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
				name:          "ecr",
				namespace:     "default",
				serialization: "json",
				resources: &envoyv1alpha1.EnvoyResources{
					Endpoints: []envoyv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
					},
					Clusters: []envoyv1alpha1.EnvoyResource{
						{Name: "cluster", Value: "{\"name\": \"cluster\"}"},
					},
					Routes: []envoyv1alpha1.EnvoyResource{
						{Name: "route", Value: "{\"name\": \"route\"}"},
					},
					Listeners: []envoyv1alpha1.EnvoyResource{
						{Name: "listener", Value: "{\"name\": \"listener\"}"},
					},
					Runtimes: []envoyv1alpha1.EnvoyResource{
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
				name:          "ecr",
				namespace:     "default",
				serialization: "json",
				resources: &envoyv1alpha1.EnvoyResources{
					Endpoints: []envoyv1alpha1.EnvoyResource{
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
				name:          "ecr",
				namespace:     "default",
				serialization: "json",
				resources: &envoyv1alpha1.EnvoyResources{
					Clusters: []envoyv1alpha1.EnvoyResource{
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
				name:          "ecr",
				namespace:     "default",
				serialization: "json",
				resources: &envoyv1alpha1.EnvoyResources{
					Routes: []envoyv1alpha1.EnvoyResource{
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
				name:          "ecr",
				namespace:     "default",
				serialization: "json",
				resources: &envoyv1alpha1.EnvoyResources{
					Listeners: []envoyv1alpha1.EnvoyResource{
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
				name:          "ecr",
				namespace:     "default",
				serialization: "json",
				resources: &envoyv1alpha1.EnvoyResources{
					Runtimes: []envoyv1alpha1.EnvoyResource{
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
				name:          "ecr",
				namespace:     "default",
				serialization: "json",
				resources: &envoyv1alpha1.EnvoyResources{
					Secrets: []envoyv1alpha1.EnvoySecretResource{
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
				name:          "ecr",
				namespace:     "default",
				serialization: "json",
				resources: &envoyv1alpha1.EnvoyResources{
					Secrets: []envoyv1alpha1.EnvoySecretResource{
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
				name:          "ecr",
				namespace:     "default",
				serialization: "json",
				resources: &envoyv1alpha1.EnvoyResources{
					Secrets: []envoyv1alpha1.EnvoySecretResource{
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
			r := &EnvoyConfigRevisionReconciler{
				Client:   tt.fields.client,
				Scheme:   tt.fields.scheme,
				ADSCache: tt.fields.adsCache,
				Log:      ctrl.Log.WithName("test"),
			}
			if err := r.loadResources(tt.args.ctx, tt.args.name, tt.args.namespace, tt.args.serialization, tt.args.resources, field.NewPath("spec", "resources"), tt.args.snap); (err != nil) != tt.wantErr {
				t.Errorf("EnvoyConfigRevisionReconciler.loadResources() error = %v, wantErr %v", err, tt.wantErr)
			} else if !tt.wantErr && !reflect.DeepEqual(tt.args.snap, tt.wantSnap) {
				t.Errorf("EnvoyConfigRevisionReconciler.loadResources() got = %v, want %v", tt.args.snap, tt.wantSnap)
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
