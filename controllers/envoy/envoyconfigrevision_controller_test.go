package controllers

import (
	"context"
	"reflect"
	"testing"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"

	xdss "github.com/3scale/marin3r/pkg/discoveryservice/xdss"
	xdss_v2 "github.com/3scale/marin3r/pkg/discoveryservice/xdss/v2"
	envoy_resources "github.com/3scale/marin3r/pkg/envoy/resources"
	testutil "github.com/3scale/marin3r/pkg/util/test"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"

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
		wantSnap    xdss.Snapshot
		wantVersion string
		wantErr     bool
	}{
		// {
		// 	name:   "Creates new snapshot for nodeID",
		// 	nodeID: "node3",
		// 	cr: &envoyv1alpha1.EnvoyConfigRevision{
		// 		ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "default"},
		// 		Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
		// 			NodeID:  "node3",
		// 			Version: "xxxx",
		// 			EnvoyResources: &envoyv1alpha1.EnvoyResources{
		// 				Endpoints: []envoyv1alpha1.EnvoyResource{
		// 					{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
		// 				}}},
		// 		Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
		// 			Conditions: status.NewConditions(status.Condition{
		// 				Type:   envoyv1alpha1.RevisionPublishedCondition,
		// 				Status: corev1.ConditionTrue,
		// 			})},
		// 	},
		// 	wantResult: reconcile.Result{},
		// 	wantSnap: xdss_v2.NewSnapshot(&cache_v2.Snapshot{
		// 		Resources: [6]cache_v2.Resources{
		// 			{Version: "xxxx", Items: map[string]cache_types.Resource{
		// 				"endpoint": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint"}}},
		// 			{Version: "xxxx", Items: map[string]cache_types.Resource{}},
		// 			{Version: "xxxx", Items: map[string]cache_types.Resource{}},
		// 			{Version: "xxxx", Items: map[string]cache_types.Resource{}},
		// 			{Version: "xxxx-74d569cc4", Items: map[string]cache_types.Resource{}},
		// 			{Version: "xxxx", Items: map[string]cache_types.Resource{}},
		// 		}}),
		// 	wantVersion: "xxxx",
		// 	wantErr:     false,
		// },
		{
			name:   "Does not update snapshot if version doesn't change",
			nodeID: "node1",
			cr: &envoyv1alpha1.EnvoyConfigRevision{
				ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "default"},
				Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
					NodeID:         "node1",
					Version:        "aaaa",
					EnvoyResources: &envoyv1alpha1.EnvoyResources{},
				},
				Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
					Conditions: status.NewConditions(status.Condition{
						Type:   envoyv1alpha1.RevisionPublishedCondition,
						Status: corev1.ConditionTrue,
					})},
			},
			wantResult: reconcile.Result{},
			wantSnap: xdss_v2.NewSnapshot(&cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "aaaa", Items: map[string]cache_types.Resource{
						"endpoint1": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint1"},
					}},
					{Version: "aaaa", Items: map[string]cache_types.Resource{
						"cluster1": &envoy_api_v2.Cluster{Name: "cluster1"},
					}},
					{Version: "aaaa", Items: map[string]cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]cache_types.Resource{}},
					{Version: "aaaa-557db659d4", Items: map[string]cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]cache_types.Resource{}},
				}}),
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
			wantSnap: xdss_v2.NewSnapshot(&cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "aaaa", Items: map[string]cache_types.Resource{
						"endpoint1": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint1"},
					}},
					{Version: "aaaa", Items: map[string]cache_types.Resource{
						"cluster1": &envoy_api_v2.Cluster{Name: "cluster1"},
					}},
					{Version: "aaaa", Items: map[string]cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]cache_types.Resource{}},
					{Version: "aaaa", Items: map[string]cache_types.Resource{}},
				}}),
			wantVersion: "aaaa",
			wantErr:     false,
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			r := &EnvoyConfigRevisionReconciler{
				Client:   fake.NewFakeClient(tt.cr),
				Scheme:   s,
				XdsCache: fakeTestCache(),
				Log:      ctrl.Log.WithName("test"),
			}
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "ecr",
					Namespace: "default",
				},
			}

			gotResult, gotErr := r.Reconcile(req)
			gotSnap, _ := r.XdsCache.GetSnapshot(tt.nodeID)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("EnvoyConfigRevisionReconciler.Reconcile() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("EnvoyConfigRevisionReconciler.Reconcile() = %v, want %v", gotResult, tt.wantResult)
			}

			if !tt.wantErr && !testutil.SnapshotsAreEqual(gotSnap, tt.wantSnap) {
				t.Errorf("Snapshot = %v, want %v", gotSnap, tt.wantSnap)
			}
			// NOTE: we are keeping the same version for all resource types
			gotVersion := gotSnap.GetVersion(envoy_resources.Cluster)
			if !tt.wantErr && gotVersion != tt.wantVersion {
				t.Errorf("Snapshot version = %v, want %v", gotVersion, tt.wantVersion)
			}
		})
	}

	t.Run("No error if ecr not found", func(t *testing.T) {
		r := &EnvoyConfigRevisionReconciler{
			Client:   fake.NewFakeClient(),
			Scheme:   s,
			XdsCache: fakeTestCache(),
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
			XdsCache: fakeTestCache(),
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
			XdsCache: fakeTestCache(),
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
			XdsCache: fakeTestCache(),
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
		xdsCache xdss.Cache
	}
	type args struct {
		ctx           context.Context
		name          string
		namespace     string
		serialization string
		resources     *envoyv1alpha1.EnvoyResources
		snap          xdss.Snapshot
	}

	cache := fakeTestCache()

	tests := []struct {
		name     string
		fields   fields
		args     args
		wantErr  bool
		wantSnap xdss.Snapshot
	}{
		{
			name: "Loads resources into the snapshot",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				xdsCache: cache,
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
				snap: cache.NewSnapshot("1"),
			},
			wantErr: false,
			wantSnap: xdss_v2.NewSnapshot(&cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "1", Items: map[string]cache_types.Resource{
						"endpoint": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint"},
					}},
					{Version: "1", Items: map[string]cache_types.Resource{
						"cluster": &envoy_api_v2.Cluster{Name: "cluster"},
					}},
					{Version: "1", Items: map[string]cache_types.Resource{
						"route": &envoy_api_v2_route.Route{Name: "route"},
					}},
					{Version: "1", Items: map[string]cache_types.Resource{
						"listener": &envoy_api_v2.Listener{Name: "listener"},
					}},
					{Version: "1-74d569cc4", Items: map[string]cache_types.Resource{}},
					{Version: "1", Items: map[string]cache_types.Resource{
						"runtime": &envoy_service_discovery_v2.Runtime{Name: "runtime"},
					}},
				},
			}),
		},
		{
			name: "Error, bad endpoint value",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				xdsCache: cache,
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
				snap: cache.NewSnapshot("1"),
			},
			wantErr:  true,
			wantSnap: xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
		},
		{
			name: "Error, bad cluster value",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				xdsCache: cache,
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
				snap: cache.NewSnapshot("1"),
			},
			wantErr:  true,
			wantSnap: xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
		},
		{
			name: "Error, bad route value",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				xdsCache: cache,
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
				snap: cache.NewSnapshot("1"),
			},
			wantErr:  true,
			wantSnap: xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
		},
		{
			name: "Error, bad listener value",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				xdsCache: cache,
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
				snap: cache.NewSnapshot("1"),
			},
			wantErr:  true,
			wantSnap: xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
		},
		{
			name: "Error, bad runtime value",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				xdsCache: cache,
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
				snap: cache.NewSnapshot("1"),
			},
			wantErr:  true,
			wantSnap: xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
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
				xdsCache: fakeTestCache(),
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
				snap: cache.NewSnapshot("1"),
			},
			wantErr: false,
			wantSnap: xdss_v2.NewSnapshot(&cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "1", Items: map[string]cache_types.Resource{}},
					{Version: "1", Items: map[string]cache_types.Resource{}},
					{Version: "1", Items: map[string]cache_types.Resource{}},
					{Version: "1", Items: map[string]cache_types.Resource{}},
					{Version: "1-6cf7fd9d65", Items: map[string]cache_types.Resource{
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
					{Version: "1", Items: map[string]cache_types.Resource{}},
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
				scheme:   scheme.Scheme,
				xdsCache: fakeTestCache(),
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
				snap: cache.NewSnapshot("1"),
			},
			wantErr:  true,
			wantSnap: xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
		},
		{
			name: "Fails when secret does not exist",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   scheme.Scheme,
				xdsCache: cache,
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
				snap: cache.NewSnapshot("1"),
			},
			wantErr:  true,
			wantSnap: xdss_v2.NewSnapshot(&cache_v2.Snapshot{}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EnvoyConfigRevisionReconciler{
				Client:   tt.fields.client,
				Scheme:   tt.fields.scheme,
				XdsCache: tt.fields.xdsCache,
				Log:      ctrl.Log.WithName("test"),
			}
			if err := r.loadResources(tt.args.ctx, tt.args.name, tt.args.namespace, tt.args.serialization, tt.args.resources, field.NewPath("spec", "resources"), tt.args.snap); (err != nil) != tt.wantErr {
				t.Errorf("EnvoyConfigRevisionReconciler.loadResources() error = %v, wantErr %v", err, tt.wantErr)
			} else if !tt.wantErr && !testutil.SnapshotsAreEqual(tt.args.snap, tt.wantSnap) {
				t.Errorf("EnvoyConfigRevisionReconciler.loadResources() got = %v, want %v", tt.args.snap, tt.wantSnap)
			}
		})
	}
}
