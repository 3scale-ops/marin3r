package controllers

import (
	"context"
	"reflect"
	"testing"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"

	xdss "github.com/3scale/marin3r/pkg/discoveryservice/xdss"
	xdss_v2 "github.com/3scale/marin3r/pkg/discoveryservice/xdss/v2"
	envoy "github.com/3scale/marin3r/pkg/envoy"
	testutil "github.com/3scale/marin3r/pkg/util/test"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"

	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
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

func fakeTestCache() xdss.Cache {
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
		{
			name:   "Writes a new snapshot for nodeID to the xDS cache",
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
			wantVersion: "xxxx",
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
					{Version: "aaaa-557db659d4", Items: map[string]cache_types.Resource{}},
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
			gotVersion := gotSnap.GetVersion(envoy.Cluster)
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
