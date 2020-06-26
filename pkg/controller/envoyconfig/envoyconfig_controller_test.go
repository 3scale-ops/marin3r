package envoyconfig

import (
	"context"
	"testing"

	marin3rv1alpha1 "github.com/3scale/marin3r/pkg/apis/marin3r/v1alpha1"
	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	xds_cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
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
	s.AddKnownTypes(marin3rv1alpha1.SchemeGroupVersion,
		&marin3rv1alpha1.EnvoyConfigRevision{},
		&marin3rv1alpha1.EnvoyConfigRevisionList{},
		&marin3rv1alpha1.EnvoyConfig{},
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

func TestReconcileEnvoyConfig_Reconcile(t *testing.T) {

	t.Run("Creates a new EnvoyConfigRevision and publishes it", func(t *testing.T) {
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
		}
		r := &ReconcileEnvoyConfig{
			client:   fake.NewFakeClient(ec),
			scheme:   s,
			adsCache: fakeTestCache(),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ec",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("ReconcileEnvoyConfigRevision.Reconcile() error = %v", gotErr)
			return
		}

		ecrList := &marin3rv1alpha1.EnvoyConfigRevisionList{}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID},
		})
		r.client.List(context.TODO(), ecrList, &client.ListOptions{LabelSelector: selector})
		if len(ecrList.Items) != 1 {
			t.Errorf("Got wrong number of envoyconfigrevisions: %v", len(ecrList.Items))
			return
		}
		ecr := ecrList.Items[0]
		if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			t.Errorf("Revision created but not marked as published")
			return
		}
		r.client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
		if ec.ObjectMeta.Finalizers[0] != marin3rv1alpha1.EnvoyConfigFinalizer {
			t.Errorf("NodeCacheConfig missing finalizer")
			return
		}
		if len(ec.Status.ConfigRevisions) != 1 {
			t.Errorf("ConfigRevisions list was not updated")
			return
		}
		version := calculateRevisionHash(ec.Spec.Resources)
		if ec.Status.PublishedVersion != version ||
			ec.Status.DesiredVersion != version ||
			ec.Status.CacheState != marin3rv1alpha1.InSyncState ||
			!ec.Status.Conditions.IsFalseFor(marin3rv1alpha1.CacheOutOfSyncCondition) {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

	t.Run("Publishes an already existent revision if versions (the resources hash) match", func(t *testing.T) {
		version := calculateRevisionHash(&marin3rv1alpha1.EnvoyResources{})
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
		}
		ecr := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: version},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				NodeID:    "node1",
				Version:   version,
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
		}

		r := &ReconcileEnvoyConfig{
			client:   fake.NewFakeClient(ec, ecr),
			scheme:   s,
			adsCache: fakeTestCache(),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ec",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("ReconcileEnvoyConfigRevision.Reconcile() error = %v", gotErr)
			return
		}

		ecrList := &marin3rv1alpha1.EnvoyConfigRevisionList{}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID},
		})
		r.client.List(context.TODO(), ecrList, &client.ListOptions{LabelSelector: selector})
		if len(ecrList.Items) != 1 {
			t.Errorf("Got wrong number of envoyconfigrevisions: %v", len(ecrList.Items))
			return
		}
		ecr = &ecrList.Items[0]
		if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			t.Errorf("Revision not marked as published")
			return
		}
		r.client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
		if len(ec.Status.ConfigRevisions) != 1 {
			t.Errorf("ConfigRevisions list was not updated")
			return
		}
		if ec.Status.PublishedVersion != version ||
			ec.Status.DesiredVersion != version ||
			ec.Status.CacheState != marin3rv1alpha1.InSyncState ||
			!ec.Status.Conditions.IsFalseFor(marin3rv1alpha1.CacheOutOfSyncCondition) {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

	t.Run("From top to bottom, publishes the first non tainted revision of the ConfigRevisions list", func(t *testing.T) {
		version := calculateRevisionHash(&marin3rv1alpha1.EnvoyResources{})
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigStatus{
				ConfigRevisions: []marin3rv1alpha1.ConfigRevisionRef{
					{Version: "aaaa", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
					{Version: version, Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
				},
			},
		}
		ecrList := &marin3rv1alpha1.EnvoyConfigRevisionList{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
			Items: []marin3rv1alpha1.EnvoyConfigRevision{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ecr1",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: "aaaa"},
					},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:    "node1",
						Version:   "aaaa",
						Resources: &marin3rv1alpha1.EnvoyResources{},
					}},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ecr2",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: version},
					},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:    "node1",
						Version:   version,
						Resources: &marin3rv1alpha1.EnvoyResources{},
					},
					Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
						Conditions: status.NewConditions(
							status.Condition{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							},
						),
					}}},
		}

		r := &ReconcileEnvoyConfig{
			client:   fake.NewFakeClient(ec, ecrList),
			scheme:   s,
			adsCache: fakeTestCache(),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ec",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("ReconcileEnvoyConfigRevision.Reconcile() error = %v", gotErr)
			return
		}

		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID},
		})
		r.client.List(context.TODO(), ecrList, &client.ListOptions{LabelSelector: selector})

		if !ecrList.Items[0].Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			t.Errorf("Revision not marked as published")
			return
		}

		r.client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
		if ec.Status.PublishedVersion != "aaaa" ||
			ec.Status.DesiredVersion != version ||
			ec.Status.CacheState != marin3rv1alpha1.RollbackState ||
			!ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.CacheOutOfSyncCondition) {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

	t.Run("Set RollbackFailed state if all versions are tainted", func(t *testing.T) {
		version := calculateRevisionHash(&marin3rv1alpha1.EnvoyResources{})
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigStatus{
				ConfigRevisions: []marin3rv1alpha1.ConfigRevisionRef{
					{Version: "aaaa", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
					{Version: version, Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
				},
			},
		}
		ecrList := &marin3rv1alpha1.EnvoyConfigRevisionList{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
			Items: []marin3rv1alpha1.EnvoyConfigRevision{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ecr1",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: "aaaa"},
					},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:    "node1",
						Version:   "aaaa",
						Resources: &marin3rv1alpha1.EnvoyResources{},
					},
					Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
						Conditions: status.NewConditions(
							status.Condition{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							},
						),
					}},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ecr2",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: version},
					},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:    "node1",
						Version:   version,
						Resources: &marin3rv1alpha1.EnvoyResources{},
					},
					Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
						Conditions: status.NewConditions(
							status.Condition{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							},
						),
					}}},
		}

		r := &ReconcileEnvoyConfig{
			client:   fake.NewFakeClient(ec, ecrList),
			scheme:   s,
			adsCache: fakeTestCache(),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ec",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("ReconcileEnvoyConfigRevision.Reconcile() error = %v", gotErr)
			return
		}

		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID},
		})
		r.client.List(context.TODO(), ecrList, &client.ListOptions{LabelSelector: selector})

		for _, ecr := range ecrList.Items {
			if ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
				t.Errorf("A revison is marked as published and it shouldn't")
				return
			}
		}

		r.client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
		if ec.Status.CacheState != marin3rv1alpha1.RollbackFailedState ||
			!ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.CacheOutOfSyncCondition) ||
			!ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.RollbackFailedCondition) {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

	// TODO:test the clearance of the Rollback failed condition
	t.Run("Set RollbackFailed state if all versions are tainted", func(t *testing.T) {
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID: "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{
					Endpoints: []marin3rv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
					},
				},
			},
			Status: marin3rv1alpha1.EnvoyConfigStatus{
				CacheState: marin3rv1alpha1.RollbackFailedState,
				ConfigRevisions: []marin3rv1alpha1.ConfigRevisionRef{
					{Version: "aaaa", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
				},
				Conditions: status.NewConditions(status.Condition{
					Type:   marin3rv1alpha1.RollbackFailedCondition,
					Status: corev1.ConditionTrue,
				}),
			},
		}
		ecrList := &marin3rv1alpha1.EnvoyConfigRevisionList{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
			Items: []marin3rv1alpha1.EnvoyConfigRevision{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ecr1",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: "aaaa"},
					},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:    "node1",
						Version:   "aaaa",
						Resources: &marin3rv1alpha1.EnvoyResources{},
					},
					Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
						Conditions: status.NewConditions(
							status.Condition{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							},
						),
					}},
			},
		}

		r := &ReconcileEnvoyConfig{
			client:   fake.NewFakeClient(ec, ecrList),
			scheme:   s,
			adsCache: fakeTestCache(),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ec",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("ReconcileEnvoyConfigRevision.Reconcile() error = %v", gotErr)
			return
		}

		r.client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
		if ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.RollbackFailedCondition) ||
			ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.CacheOutOfSyncCondition) ||
			ec.Status.CacheState != marin3rv1alpha1.InSyncState {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

}

func TestReconcileEnvoyConfig_Reconcile_Finalizer(t *testing.T) {

	cr := &marin3rv1alpha1.EnvoyConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ec",
			Namespace:         "default",
			DeletionTimestamp: func() *metav1.Time { t := metav1.Now(); return &t }(),
			Finalizers:        []string{marin3rv1alpha1.EnvoyConfigFinalizer},
		},
		Spec: marin3rv1alpha1.EnvoyConfigSpec{
			NodeID:    "node1",
			Resources: &marin3rv1alpha1.EnvoyResources{},
		}}

	s := scheme.Scheme
	s.AddKnownTypes(marin3rv1alpha1.SchemeGroupVersion,
		cr,
		&marin3rv1alpha1.EnvoyConfigRevisionList{},
		&marin3rv1alpha1.EnvoyConfigRevision{},
	)
	cl := fake.NewFakeClient(cr)
	r := &ReconcileEnvoyConfig{client: cl, scheme: s, adsCache: fakeTestCache()}
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "ec",
			Namespace: "default",
		},
	}

	_, gotErr := r.Reconcile(req)

	if gotErr != nil {
		t.Errorf("ReconcileEnvoyConfig.Reconcile_Finalizer() error = %v", gotErr)
		return
	}
	_, err := (*r.adsCache).GetSnapshot(cr.Spec.NodeID)
	if err == nil {
		t.Errorf("ReconcileEnvoyConfig.Reconcile_Finalizer() - snapshot still exists in the ads server cache")
		return
	}

	ec := &marin3rv1alpha1.EnvoyConfig{}
	cl.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
	if len(ec.GetObjectMeta().GetFinalizers()) != 0 {
		t.Errorf("ReconcileEnvoyConfig.Reconcile_Finalizer() - finalizer not deleted from object")
		return
	}

}

func TestReconcileEnvoyConfig_finalizeEnvoyConfig(t *testing.T) {
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
			r := &ReconcileEnvoyConfig{
				client:   tt.fields.client,
				scheme:   tt.fields.scheme,
				adsCache: tt.fields.adsCache,
			}
			r.finalizeEnvoyConfig(tt.args.nodeID)
			if _, err := (*r.adsCache).GetSnapshot(tt.args.nodeID); err == nil {
				t.Errorf("TestReconcileEnvoyConfig_finalizeEnvoyConfig() -> snapshot still in the cache")
			}
		})
	}
}

func TestReconcileEnvoyConfig_addFinalizer(t *testing.T) {
	tests := []struct {
		name    string
		cr      *marin3rv1alpha1.EnvoyConfig
		wantErr bool
	}{
		{
			name: "Adds finalizer to NodecacheConfig",
			cr: &marin3rv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: marin3rv1alpha1.EnvoyConfigSpec{
					NodeID:    "node1",
					Resources: &marin3rv1alpha1.EnvoyResources{},
				}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scheme.Scheme
			s.AddKnownTypes(marin3rv1alpha1.SchemeGroupVersion, tt.cr)
			cl := fake.NewFakeClient(tt.cr)
			r := &ReconcileEnvoyConfig{client: cl, scheme: s, adsCache: fakeTestCache()}

			if err := r.addFinalizer(context.TODO(), tt.cr); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileEnvoyConfig.addFinalizer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				ec := &marin3rv1alpha1.EnvoyConfig{}
				r.client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
				if len(ec.ObjectMeta.Finalizers) != 1 {
					t.Error("ReconcileEnvoyConfig.addFinalizer() wrong number of finalizers present in object")
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

func TestReconcileEnvoyConfig_getVersionToPublish(t *testing.T) {

	tests := []struct {
		name    string
		ec     *marin3rv1alpha1.EnvoyConfig
		ecrList *marin3rv1alpha1.EnvoyConfigRevisionList
		want    string
		wantErr bool
	}{
		{
			name: "Returns the desiredVersion on seeing a new version",
			ec: &marin3rv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: marin3rv1alpha1.EnvoyConfigSpec{
					NodeID:    "node1",
					Resources: &marin3rv1alpha1.EnvoyResources{},
				},
				Status: marin3rv1alpha1.EnvoyConfigStatus{
					ConfigRevisions: []marin3rv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
					},
				},
			},
			ecrList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:    "node1",
							Version:   "xxx",
							Resources: &marin3rv1alpha1.EnvoyResources{},
						},
					},
				},
			},
			want:    "xxx",
			wantErr: false,
		},
		{
			name: "Returns the highest index untainted revision of the ConfigRevision list",
			ec: &marin3rv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: marin3rv1alpha1.EnvoyConfigSpec{
					NodeID:    "node1",
					Resources: &marin3rv1alpha1.EnvoyResources{},
				},
				Status: marin3rv1alpha1.EnvoyConfigStatus{
					ConfigRevisions: []marin3rv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
						{Version: "zzz", Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
					},
				},
			},
			ecrList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:    "node1",
							Version:   "xxx",
							Resources: &marin3rv1alpha1.EnvoyResources{},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr2",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:    "node1",
							Version:   "zzz",
							Resources: &marin3rv1alpha1.EnvoyResources{},
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
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
			ec: &marin3rv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: marin3rv1alpha1.EnvoyConfigSpec{
					NodeID:    "node1",
					Resources: &marin3rv1alpha1.EnvoyResources{},
				},
				Status: marin3rv1alpha1.EnvoyConfigStatus{
					ConfigRevisions: []marin3rv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
						{Version: "zzz", Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
					},
				},
			},
			ecrList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:    "node1",
							Version:   "xxx",
							Resources: &marin3rv1alpha1.EnvoyResources{},
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							}),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr2",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:    "node1",
							Version:   "zzz",
							Resources: &marin3rv1alpha1.EnvoyResources{},
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
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
			r := &ReconcileEnvoyConfig{
				client:   fake.NewFakeClient(tt.ec, tt.ecrList),
				scheme:   s,
				adsCache: fakeTestCache(),
			}
			got, err := r.getVersionToPublish(context.TODO(), tt.ec)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileEnvoyConfig.getVersionToPublish() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReconcileEnvoyConfig.getVersionToPublish() = %v, want %v", got, tt.want)
			}
		})

	}
}
