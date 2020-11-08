package controllers

import (
	"context"
	"reflect"
	"testing"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	xdss "github.com/3scale/marin3r/pkg/discoveryservice/xdss"
	xdss_v2 "github.com/3scale/marin3r/pkg/discoveryservice/xdss/v2"
	xdss_v3 "github.com/3scale/marin3r/pkg/discoveryservice/xdss/v3"
	envoy "github.com/3scale/marin3r/pkg/envoy"
	testutil "github.com/3scale/marin3r/pkg/util/test"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
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

func fakeCacheV2() xdss.Cache {
	cache := xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil))
	cache.SetSnapshot("node1", xdss_v2.NewSnapshot(&cache_v2.Snapshot{
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
	)
	return cache
}

func fakeCacheV3() xdss.Cache {
	cache := xdss_v3.NewCache(cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil))
	cache.SetSnapshot("node1", xdss_v3.NewSnapshot(&cache_v3.Snapshot{
		Resources: [6]cache_v3.Resources{
			{Version: "aaaa", Items: map[string]cache_types.Resource{
				"endpoint1": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint1"},
			}},
			{Version: "aaaa", Items: map[string]cache_types.Resource{
				"cluster1": &envoy_config_cluster_v3.Cluster{Name: "cluster1"},
			}},
			{Version: "aaaa", Items: map[string]cache_types.Resource{}},
			{Version: "aaaa", Items: map[string]cache_types.Resource{}},
			{Version: "aaaa-557db659d4", Items: map[string]cache_types.Resource{}},
			{Version: "aaaa", Items: map[string]cache_types.Resource{}},
		}}),
	)
	return cache
}

func TestEnvoyConfigRevisionReconciler_Reconcile(t *testing.T) {
	type fields struct {
		Client     client.Client
		Log        logr.Logger
		Scheme     *runtime.Scheme
		XdsCache   xdss.Cache
		APIVersion envoy.APIVersion
	}
	type args struct {
		req ctrl.Request
		cr  *envoyv1alpha1.EnvoyConfigRevision
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		want     ctrl.Result
		wantErr  bool
		wantSnap xdss.Snapshot
	}{
		{
			name: "Writes a new snapshot for nodeID to the xDS cache (v2)",
			fields: fields{
				Scheme:     s,
				XdsCache:   fakeCacheV2(),
				Log:        ctrl.Log.WithName("test"),
				APIVersion: envoy.APIv2,
			},
			args: args{
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: "ecr", Namespace: "default"}},
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
			name: "Writes a new snapshot for nodeID to the xDS cache (v3)",
			fields: fields{
				Scheme:     s,
				XdsCache:   fakeCacheV3(),
				Log:        ctrl.Log.WithName("test"),
				APIVersion: envoy.APIv3,
			},
			args: args{
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: "ecr", Namespace: "default"}},
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
			name: "No changes to xds server cache when ecr has condition 'envoyv1alpha1.RevisionPublishedCondition' to false",
			fields: fields{
				Scheme:     s,
				XdsCache:   fakeCacheV2(),
				Log:        ctrl.Log.WithName("test"),
				APIVersion: envoy.APIv2,
			},
			args: args{
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: "ecr", Namespace: "default"}},
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
			},
			want:    reconcile.Result{},
			wantErr: false,
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EnvoyConfigRevisionReconciler{
				Client:     fake.NewFakeClient(tt.args.cr),
				Log:        tt.fields.Log,
				Scheme:     tt.fields.Scheme,
				XdsCache:   tt.fields.XdsCache,
				APIVersion: tt.fields.APIVersion,
			}
			got, err := r.Reconcile(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnvoyConfigRevisionReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EnvoyConfigRevisionReconciler.Reconcile() = %v, want %v", got, tt.want)
			}

			gotSnap, _ := r.XdsCache.GetSnapshot(tt.args.cr.Spec.NodeID)
			if !tt.wantErr && !testutil.SnapshotsAreEqual(gotSnap, tt.wantSnap) {
				t.Errorf("Snapshot = %v, want %v", gotSnap, tt.wantSnap)
			}
		})
	}
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
			XdsCache: fakeCacheV2(),
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

func Test_filterByAPIVersion(t *testing.T) {
	type args struct {
		obj     runtime.Object
		version envoy.APIVersion
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "V2 EnvoyConfigRevision with V2 controller returns true",
			args: args{
				obj: &envoyv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "xx", Namespace: "xx"},
					Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
						EnvoyAPI: pointer.StringPtr(string(envoy.APIv2)),
					},
				},
				version: envoy.APIv2,
			},
			want: true,
		},
		{
			name: "V3 EnvoyConfigRevision with V3 controller returns true",
			args: args{
				obj: &envoyv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "xx", Namespace: "xx"},
					Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
						EnvoyAPI: pointer.StringPtr(string(envoy.APIv3)),
					},
				},
				version: envoy.APIv3,
			},
			want: true,
		},
		{
			name: "V2 EnvoyConfigRevision with V3 controller returns false",
			args: args{
				obj: &envoyv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "xx", Namespace: "xx"},
					Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
						EnvoyAPI: pointer.StringPtr(string(envoy.APIv2)),
					},
				},
				version: envoy.APIv3,
			},
			want: false,
		},
		{
			name: "V3 EnvoyConfigRevision with V2 controller returns false",
			args: args{
				obj: &envoyv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "xx", Namespace: "xx"},
					Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
						EnvoyAPI: pointer.StringPtr(string(envoy.APIv2)),
					},
				},
				version: envoy.APIv3,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterByAPIVersion(tt.args.obj, tt.args.version); got != tt.want {
				t.Errorf("filterByAPIVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
