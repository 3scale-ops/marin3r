package nodeconfigcache

import (
	"context"
	"testing"

	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	xds_cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var s *runtime.Scheme = scheme.Scheme

func init() {
	s.AddKnownTypes(cachesv1alpha1.SchemeGroupVersion,
		&cachesv1alpha1.NodeConfigRevision{},
		&cachesv1alpha1.NodeConfigRevisionList{},
		&cachesv1alpha1.NodeConfigCache{},
	)
}

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

func TestReconcileNodeConfigCache_Reconcile(t *testing.T) {

	t.Run("Creates a new NodeConfigRevision and publishes it", func(t *testing.T) {
		ncc := &cachesv1alpha1.NodeConfigCache{
			ObjectMeta: metav1.ObjectMeta{Name: "ncc", Namespace: "default"},
			Spec: cachesv1alpha1.NodeConfigCacheSpec{
				NodeID:    "node1",
				Resources: &cachesv1alpha1.EnvoyResources{},
			},
		}
		r := &ReconcileNodeConfigCache{
			client:   fake.NewFakeClient(ncc),
			scheme:   s,
			adsCache: fakeTestCache(),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ncc",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("ReconcileNodeConfigRevision.Reconcile() error = %v", gotErr)
			return
		}

		ncrList := &cachesv1alpha1.NodeConfigRevisionList{}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ncc.Spec.NodeID},
		})
		r.client.List(context.TODO(), ncrList, &client.ListOptions{LabelSelector: selector})
		if len(ncrList.Items) != 1 {
			t.Errorf("Got wrong number of nodeconfigrevisions: %v", len(ncrList.Items))
			return
		}
		ncr := ncrList.Items[0]
		if !ncr.Status.Conditions.IsTrueFor(cachesv1alpha1.RevisionPublishedCondition) {
			t.Errorf("Revision created but not marked as published")
			return
		}
		r.client.Get(context.TODO(), types.NamespacedName{Name: "ncc", Namespace: "default"}, ncc)
		if ncc.ObjectMeta.Finalizers[0] != cachesv1alpha1.NodeConfigCacheFinalizer {
			t.Errorf("NodeCacheConfig missing finalizer")
			return
		}
		if len(ncc.Status.ConfigRevisions) != 1 {
			t.Errorf("ConfigRevisions list was not updated")
			return
		}
		version := calculateRevisionHash(ncc.Spec.Resources)
		if ncc.Status.PublishedVersion != version ||
			ncc.Status.DesiredVersion != version ||
			ncc.Status.CacheState != cachesv1alpha1.InSyncState ||
			!ncc.Status.Conditions.IsFalseFor(cachesv1alpha1.CacheOutOfSyncCondition) {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

	t.Run("Publishes an already existent revision if versions (the resources hash) match", func(t *testing.T) {
		version := calculateRevisionHash(&cachesv1alpha1.EnvoyResources{})
		ncc := &cachesv1alpha1.NodeConfigCache{
			ObjectMeta: metav1.ObjectMeta{Name: "ncc", Namespace: "default"},
			Spec: cachesv1alpha1.NodeConfigCacheSpec{
				NodeID:    "node1",
				Resources: &cachesv1alpha1.EnvoyResources{},
			},
		}
		ncr := &cachesv1alpha1.NodeConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ncr",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: version},
			},
			Spec: cachesv1alpha1.NodeConfigRevisionSpec{
				NodeID:    "node1",
				Version:   version,
				Resources: &cachesv1alpha1.EnvoyResources{},
			},
		}

		r := &ReconcileNodeConfigCache{
			client:   fake.NewFakeClient(ncc, ncr),
			scheme:   s,
			adsCache: fakeTestCache(),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ncc",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("ReconcileNodeConfigRevision.Reconcile() error = %v", gotErr)
			return
		}

		ncrList := &cachesv1alpha1.NodeConfigRevisionList{}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ncc.Spec.NodeID},
		})
		r.client.List(context.TODO(), ncrList, &client.ListOptions{LabelSelector: selector})
		if len(ncrList.Items) != 1 {
			t.Errorf("Got wrong number of nodeconfigrevisions: %v", len(ncrList.Items))
			return
		}
		ncr = &ncrList.Items[0]
		if !ncr.Status.Conditions.IsTrueFor(cachesv1alpha1.RevisionPublishedCondition) {
			t.Errorf("Revision not marked as published")
			return
		}
		r.client.Get(context.TODO(), types.NamespacedName{Name: "ncc", Namespace: "default"}, ncc)
		if len(ncc.Status.ConfigRevisions) != 1 {
			t.Errorf("ConfigRevisions list was not updated")
			return
		}
		if ncc.Status.PublishedVersion != version ||
			ncc.Status.DesiredVersion != version ||
			ncc.Status.CacheState != cachesv1alpha1.InSyncState ||
			!ncc.Status.Conditions.IsFalseFor(cachesv1alpha1.CacheOutOfSyncCondition) {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

	t.Run("From top to bottom, publishes the first non tainted revision of the ConfigRevisions list", func(t *testing.T) {
		version := calculateRevisionHash(&cachesv1alpha1.EnvoyResources{})
		ncc := &cachesv1alpha1.NodeConfigCache{
			ObjectMeta: metav1.ObjectMeta{Name: "ncc", Namespace: "default"},
			Spec: cachesv1alpha1.NodeConfigCacheSpec{
				NodeID:    "node1",
				Resources: &cachesv1alpha1.EnvoyResources{},
			},
			Status: cachesv1alpha1.NodeConfigCacheStatus{
				ConfigRevisions: []cachesv1alpha1.ConfigRevisionRef{
					{Version: "aaaa", Ref: corev1.ObjectReference{Name: "ncr1", Namespace: "default"}},
					{Version: version, Ref: corev1.ObjectReference{Name: "ncr2", Namespace: "default"}},
				},
			},
		}
		ncrList := &cachesv1alpha1.NodeConfigRevisionList{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
			Items: []cachesv1alpha1.NodeConfigRevision{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ncr1",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: "aaaa"},
					},
					Spec: cachesv1alpha1.NodeConfigRevisionSpec{
						NodeID:    "node1",
						Version:   "aaaa",
						Resources: &cachesv1alpha1.EnvoyResources{},
					}},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ncr2",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: version},
					},
					Spec: cachesv1alpha1.NodeConfigRevisionSpec{
						NodeID:    "node1",
						Version:   version,
						Resources: &cachesv1alpha1.EnvoyResources{},
					},
					Status: cachesv1alpha1.NodeConfigRevisionStatus{
						Conditions: status.NewConditions(
							status.Condition{
								Type:   cachesv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							},
						),
					}}},
		}

		r := &ReconcileNodeConfigCache{
			client:   fake.NewFakeClient(ncc, ncrList),
			scheme:   s,
			adsCache: fakeTestCache(),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ncc",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("ReconcileNodeConfigRevision.Reconcile() error = %v", gotErr)
			return
		}

		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ncc.Spec.NodeID},
		})
		r.client.List(context.TODO(), ncrList, &client.ListOptions{LabelSelector: selector})

		if !ncrList.Items[0].Status.Conditions.IsTrueFor(cachesv1alpha1.RevisionPublishedCondition) {
			t.Errorf("Revision not marked as published")
			return
		}

		r.client.Get(context.TODO(), types.NamespacedName{Name: "ncc", Namespace: "default"}, ncc)
		if ncc.Status.PublishedVersion != "aaaa" ||
			ncc.Status.DesiredVersion != version ||
			ncc.Status.CacheState != cachesv1alpha1.RollbackState ||
			!ncc.Status.Conditions.IsTrueFor(cachesv1alpha1.CacheOutOfSyncCondition) {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

	t.Run("Set RollbackFailed state if all versions are tainted", func(t *testing.T) {
		version := calculateRevisionHash(&cachesv1alpha1.EnvoyResources{})
		ncc := &cachesv1alpha1.NodeConfigCache{
			ObjectMeta: metav1.ObjectMeta{Name: "ncc", Namespace: "default"},
			Spec: cachesv1alpha1.NodeConfigCacheSpec{
				NodeID:    "node1",
				Resources: &cachesv1alpha1.EnvoyResources{},
			},
			Status: cachesv1alpha1.NodeConfigCacheStatus{
				ConfigRevisions: []cachesv1alpha1.ConfigRevisionRef{
					{Version: "aaaa", Ref: corev1.ObjectReference{Name: "ncr1", Namespace: "default"}},
					{Version: version, Ref: corev1.ObjectReference{Name: "ncr2", Namespace: "default"}},
				},
			},
		}
		ncrList := &cachesv1alpha1.NodeConfigRevisionList{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
			Items: []cachesv1alpha1.NodeConfigRevision{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ncr1",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: "aaaa"},
					},
					Spec: cachesv1alpha1.NodeConfigRevisionSpec{
						NodeID:    "node1",
						Version:   "aaaa",
						Resources: &cachesv1alpha1.EnvoyResources{},
					},
					Status: cachesv1alpha1.NodeConfigRevisionStatus{
						Conditions: status.NewConditions(
							status.Condition{
								Type:   cachesv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							},
						),
					}},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ncr2",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: version},
					},
					Spec: cachesv1alpha1.NodeConfigRevisionSpec{
						NodeID:    "node1",
						Version:   version,
						Resources: &cachesv1alpha1.EnvoyResources{},
					},
					Status: cachesv1alpha1.NodeConfigRevisionStatus{
						Conditions: status.NewConditions(
							status.Condition{
								Type:   cachesv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							},
						),
					}}},
		}

		r := &ReconcileNodeConfigCache{
			client:   fake.NewFakeClient(ncc, ncrList),
			scheme:   s,
			adsCache: fakeTestCache(),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ncc",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("ReconcileNodeConfigRevision.Reconcile() error = %v", gotErr)
			return
		}

		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ncc.Spec.NodeID},
		})
		r.client.List(context.TODO(), ncrList, &client.ListOptions{LabelSelector: selector})

		for _, ncr := range ncrList.Items {
			if ncr.Status.Conditions.IsTrueFor(cachesv1alpha1.RevisionPublishedCondition) {
				t.Errorf("A revison is marked as published and it shouldn't")
				return
			}
		}

		r.client.Get(context.TODO(), types.NamespacedName{Name: "ncc", Namespace: "default"}, ncc)
		if ncc.Status.CacheState != cachesv1alpha1.RollbackFailedState ||
			!ncc.Status.Conditions.IsTrueFor(cachesv1alpha1.CacheOutOfSyncCondition) ||
			!ncc.Status.Conditions.IsTrueFor(cachesv1alpha1.RollbackFailedCondition) {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

	// TODO:test the clearance of the Rollback failed condition
	t.Run("Set RollbackFailed state if all versions are tainted", func(t *testing.T) {
		ncc := &cachesv1alpha1.NodeConfigCache{
			ObjectMeta: metav1.ObjectMeta{Name: "ncc", Namespace: "default"},
			Spec: cachesv1alpha1.NodeConfigCacheSpec{
				NodeID: "node1",
				Resources: &cachesv1alpha1.EnvoyResources{
					Endpoints: []cachesv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
					},
				},
			},
			Status: cachesv1alpha1.NodeConfigCacheStatus{
				CacheState: cachesv1alpha1.RollbackFailedState,
				ConfigRevisions: []cachesv1alpha1.ConfigRevisionRef{
					{Version: "aaaa", Ref: corev1.ObjectReference{Name: "ncr1", Namespace: "default"}},
				},
				Conditions: status.NewConditions(status.Condition{
					Type:   cachesv1alpha1.RollbackFailedCondition,
					Status: corev1.ConditionTrue,
				}),
			},
		}
		ncrList := &cachesv1alpha1.NodeConfigRevisionList{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
			Items: []cachesv1alpha1.NodeConfigRevision{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ncr1",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: "aaaa"},
					},
					Spec: cachesv1alpha1.NodeConfigRevisionSpec{
						NodeID:    "node1",
						Version:   "aaaa",
						Resources: &cachesv1alpha1.EnvoyResources{},
					},
					Status: cachesv1alpha1.NodeConfigRevisionStatus{
						Conditions: status.NewConditions(
							status.Condition{
								Type:   cachesv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							},
						),
					}},
			},
		}

		r := &ReconcileNodeConfigCache{
			client:   fake.NewFakeClient(ncc, ncrList),
			scheme:   s,
			adsCache: fakeTestCache(),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ncc",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("ReconcileNodeConfigRevision.Reconcile() error = %v", gotErr)
			return
		}

		r.client.Get(context.TODO(), types.NamespacedName{Name: "ncc", Namespace: "default"}, ncc)
		if ncc.Status.Conditions.IsTrueFor(cachesv1alpha1.RollbackFailedCondition) ||
			ncc.Status.Conditions.IsTrueFor(cachesv1alpha1.CacheOutOfSyncCondition) ||
			ncc.Status.CacheState != cachesv1alpha1.InSyncState {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

}

func TestReconcileNodeConfigCache_Reconcile_Finalizer(t *testing.T) {

	cr := &cachesv1alpha1.NodeConfigCache{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ncc",
			Namespace:         "default",
			DeletionTimestamp: func() *metav1.Time { t := metav1.Now(); return &t }(),
			Finalizers:        []string{cachesv1alpha1.NodeConfigCacheFinalizer},
		},
		Spec: cachesv1alpha1.NodeConfigCacheSpec{
			NodeID:    "node1",
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

func TestReconcileNodeConfigCache_getVersionToPublish(t *testing.T) {

	tests := []struct {
		name    string
		ncc     *cachesv1alpha1.NodeConfigCache
		ncrList *cachesv1alpha1.NodeConfigRevisionList
		want    string
		wantErr bool
	}{
		{
			name: "Returns the desiredVersion on seeing a new version",
			ncc: &cachesv1alpha1.NodeConfigCache{
				ObjectMeta: metav1.ObjectMeta{Name: "ncc", Namespace: "default"},
				Spec: cachesv1alpha1.NodeConfigCacheSpec{
					NodeID:    "node1",
					Resources: &cachesv1alpha1.EnvoyResources{},
				},
				Status: cachesv1alpha1.NodeConfigCacheStatus{
					ConfigRevisions: []cachesv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: v1.ObjectReference{Name: "ncr1", Namespace: "default"}},
					},
				},
			},
			ncrList: &cachesv1alpha1.NodeConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []cachesv1alpha1.NodeConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ncr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: cachesv1alpha1.NodeConfigRevisionSpec{
							NodeID:    "node1",
							Version:   "xxx",
							Resources: &cachesv1alpha1.EnvoyResources{},
						},
					},
				},
			},
			want:    "xxx",
			wantErr: false,
		},
		{
			name: "Returns the highest index untainted revision of the ConfigRevision list",
			ncc: &cachesv1alpha1.NodeConfigCache{
				ObjectMeta: metav1.ObjectMeta{Name: "ncc", Namespace: "default"},
				Spec: cachesv1alpha1.NodeConfigCacheSpec{
					NodeID:    "node1",
					Resources: &cachesv1alpha1.EnvoyResources{},
				},
				Status: cachesv1alpha1.NodeConfigCacheStatus{
					ConfigRevisions: []cachesv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: v1.ObjectReference{Name: "ncr1", Namespace: "default"}},
						{Version: "zzz", Ref: v1.ObjectReference{Name: "ncr2", Namespace: "default"}},
					},
				},
			},
			ncrList: &cachesv1alpha1.NodeConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []cachesv1alpha1.NodeConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ncr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: cachesv1alpha1.NodeConfigRevisionSpec{
							NodeID:    "node1",
							Version:   "xxx",
							Resources: &cachesv1alpha1.EnvoyResources{},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ncr2",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: cachesv1alpha1.NodeConfigRevisionSpec{
							NodeID:    "node1",
							Version:   "zzz",
							Resources: &cachesv1alpha1.EnvoyResources{},
						},
						Status: cachesv1alpha1.NodeConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   cachesv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							}),
						},
					},
				},
			},
			want:    "xxx",
			wantErr: false,
		},
		{
			name: "Returns an error if all revisions are tainted",
			ncc: &cachesv1alpha1.NodeConfigCache{
				ObjectMeta: metav1.ObjectMeta{Name: "ncc", Namespace: "default"},
				Spec: cachesv1alpha1.NodeConfigCacheSpec{
					NodeID:    "node1",
					Resources: &cachesv1alpha1.EnvoyResources{},
				},
				Status: cachesv1alpha1.NodeConfigCacheStatus{
					ConfigRevisions: []cachesv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: v1.ObjectReference{Name: "ncr1", Namespace: "default"}},
						{Version: "zzz", Ref: v1.ObjectReference{Name: "ncr2", Namespace: "default"}},
					},
				},
			},
			ncrList: &cachesv1alpha1.NodeConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []cachesv1alpha1.NodeConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ncr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: cachesv1alpha1.NodeConfigRevisionSpec{
							NodeID:    "node1",
							Version:   "xxx",
							Resources: &cachesv1alpha1.EnvoyResources{},
						},
						Status: cachesv1alpha1.NodeConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   cachesv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							}),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ncr2",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: cachesv1alpha1.NodeConfigRevisionSpec{
							NodeID:    "node1",
							Version:   "zzz",
							Resources: &cachesv1alpha1.EnvoyResources{},
						},
						Status: cachesv1alpha1.NodeConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   cachesv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							}),
						},
					},
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileNodeConfigCache{
				client:   fake.NewFakeClient(tt.ncc, tt.ncrList),
				scheme:   s,
				adsCache: fakeTestCache(),
			}
			got, err := r.getVersionToPublish(context.TODO(), tt.ncc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileNodeConfigCache.getVersionToPublish() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReconcileNodeConfigCache.getVersionToPublish() = %v, want %v", got, tt.want)
			}
		})

	}
}
