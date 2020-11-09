package controllers

import (
	"context"
	"testing"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"

	xdss "github.com/3scale/marin3r/pkg/discoveryservice/xdss"

	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

func TestEnvoyConfigReconciler_Reconcile(t *testing.T) {

	t.Run("Creates a new EnvoyConfigRevision and publishes it", func(t *testing.T) {
		ec := &envoyv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
			Spec: envoyv1alpha1.EnvoyConfigSpec{
				NodeID:         "node1",
				EnvoyResources: &envoyv1alpha1.EnvoyResources{},
			},
		}
		r := &EnvoyConfigReconciler{
			Client:   fake.NewFakeClient(ec),
			Scheme:   s,
			XdsCache: fakeCacheV2(),
			Log:      ctrl.Log.WithName("test"),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ec",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("EnvoyConfigReconcilerRevision.Reconcile() error = %v", gotErr)
			return
		}

		ecrList := &envoyv1alpha1.EnvoyConfigRevisionList{}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID},
		})
		r.Client.List(context.TODO(), ecrList, &client.ListOptions{LabelSelector: selector})
		if len(ecrList.Items) != 1 {
			t.Errorf("Got wrong number of envoyconfigrevisions: %v", len(ecrList.Items))
			return
		}
		ecr := ecrList.Items[0]
		if !ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition) {
			t.Errorf("Revision created but not marked as published")
			return
		}
		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
		if ec.ObjectMeta.Finalizers[0] != envoyv1alpha1.EnvoyConfigFinalizer {
			t.Errorf("EnvoyConfig missing finalizer")
			return
		}
		if len(ec.Status.ConfigRevisions) != 1 {
			t.Errorf("ConfigRevisions list was not updated")
			return
		}
		version := calculateRevisionHash(ec.Spec.EnvoyResources)
		if ec.Status.PublishedVersion != version ||
			ec.Status.DesiredVersion != version ||
			ec.Status.CacheState != envoyv1alpha1.InSyncState ||
			!ec.Status.Conditions.IsFalseFor(envoyv1alpha1.CacheOutOfSyncCondition) {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

	t.Run("Publishes an already existent revision if versions (the resources hash) match", func(t *testing.T) {
		version := calculateRevisionHash(&envoyv1alpha1.EnvoyResources{})
		ec := &envoyv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
			Spec: envoyv1alpha1.EnvoyConfigSpec{
				NodeID:         "node1",
				EnvoyResources: &envoyv1alpha1.EnvoyResources{},
			},
		}
		ecr := &envoyv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: version},
			},
			Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
				NodeID:         "node1",
				Version:        version,
				EnvoyResources: &envoyv1alpha1.EnvoyResources{},
			},
		}

		r := &EnvoyConfigReconciler{
			Client:   fake.NewFakeClient(ec, ecr),
			Scheme:   s,
			XdsCache: fakeCacheV2(),
			Log:      ctrl.Log.WithName("test"),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ec",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("EnvoyConfigReconcilerRevision.Reconcile() error = %v", gotErr)
			return
		}

		ecrList := &envoyv1alpha1.EnvoyConfigRevisionList{}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID},
		})
		r.Client.List(context.TODO(), ecrList, &client.ListOptions{LabelSelector: selector})
		if len(ecrList.Items) != 1 {
			t.Errorf("Got wrong number of envoyconfigrevisions: %v", len(ecrList.Items))
			return
		}
		ecr = &ecrList.Items[0]
		if !ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition) {
			t.Errorf("Revision not marked as published")
			return
		}
		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
		if len(ec.Status.ConfigRevisions) != 1 {
			t.Errorf("ConfigRevisions list was not updated")
			return
		}
		if ec.Status.PublishedVersion != version ||
			ec.Status.DesiredVersion != version ||
			ec.Status.CacheState != envoyv1alpha1.InSyncState ||
			!ec.Status.Conditions.IsFalseFor(envoyv1alpha1.CacheOutOfSyncCondition) {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

	t.Run("From top to bottom, publishes the first non tainted revision of the ConfigRevisions list", func(t *testing.T) {
		version := calculateRevisionHash(&envoyv1alpha1.EnvoyResources{})
		ec := &envoyv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
			Spec: envoyv1alpha1.EnvoyConfigSpec{
				NodeID:         "node1",
				EnvoyResources: &envoyv1alpha1.EnvoyResources{},
			},
			Status: envoyv1alpha1.EnvoyConfigStatus{
				ConfigRevisions: []envoyv1alpha1.ConfigRevisionRef{
					{Version: "aaaa", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
					{Version: version, Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
				},
			},
		}
		ecrList := &envoyv1alpha1.EnvoyConfigRevisionList{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
			Items: []envoyv1alpha1.EnvoyConfigRevision{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ecr1",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: "aaaa"},
					},
					Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:         "node1",
						Version:        "aaaa",
						EnvoyResources: &envoyv1alpha1.EnvoyResources{},
					}},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ecr2",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: version},
					},
					Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:         "node1",
						Version:        version,
						EnvoyResources: &envoyv1alpha1.EnvoyResources{},
					},
					Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
						Conditions: status.NewConditions(
							status.Condition{
								Type:   envoyv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							},
						),
					}}},
		}

		r := &EnvoyConfigReconciler{
			Client:   fake.NewFakeClient(ec, ecrList),
			Scheme:   s,
			XdsCache: fakeCacheV2(),
			Log:      ctrl.Log.WithName("test"),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ec",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("EnvoyConfigReconcilerRevision.Reconcile() error = %v", gotErr)
			return
		}

		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID},
		})
		r.Client.List(context.TODO(), ecrList, &client.ListOptions{LabelSelector: selector})

		if !ecrList.Items[0].Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition) {
			t.Errorf("Revision not marked as published")
			return
		}

		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
		if ec.Status.PublishedVersion != "aaaa" ||
			ec.Status.DesiredVersion != version ||
			ec.Status.CacheState != envoyv1alpha1.RollbackState ||
			!ec.Status.Conditions.IsTrueFor(envoyv1alpha1.CacheOutOfSyncCondition) {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

	t.Run("Set RollbackFailed state if all versions are tainted", func(t *testing.T) {
		version := calculateRevisionHash(&envoyv1alpha1.EnvoyResources{})
		ec := &envoyv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
			Spec: envoyv1alpha1.EnvoyConfigSpec{
				NodeID:         "node1",
				EnvoyResources: &envoyv1alpha1.EnvoyResources{},
			},
			Status: envoyv1alpha1.EnvoyConfigStatus{
				ConfigRevisions: []envoyv1alpha1.ConfigRevisionRef{
					{Version: "aaaa", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
					{Version: version, Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
				},
			},
		}
		ecrList := &envoyv1alpha1.EnvoyConfigRevisionList{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
			Items: []envoyv1alpha1.EnvoyConfigRevision{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ecr1",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: "aaaa"},
					},
					Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:         "node1",
						Version:        "aaaa",
						EnvoyResources: &envoyv1alpha1.EnvoyResources{},
					},
					Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
						Conditions: status.NewConditions(
							status.Condition{
								Type:   envoyv1alpha1.RevisionTaintedCondition,
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
					Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:         "node1",
						Version:        version,
						EnvoyResources: &envoyv1alpha1.EnvoyResources{},
					},
					Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
						Conditions: status.NewConditions(
							status.Condition{
								Type:   envoyv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							},
						),
					}}},
		}

		r := &EnvoyConfigReconciler{
			Client:   fake.NewFakeClient(ec, ecrList),
			Scheme:   s,
			XdsCache: fakeCacheV2(),
			Log:      ctrl.Log.WithName("test"),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ec",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("EnvoyConfigReconcilerRevision.Reconcile() error = %v", gotErr)
			return
		}

		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: ec.Spec.NodeID},
		})
		r.Client.List(context.TODO(), ecrList, &client.ListOptions{LabelSelector: selector})

		for _, ecr := range ecrList.Items {
			if ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition) {
				t.Errorf("A revison is marked as published and it shouldn't")
				return
			}
		}

		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
		if ec.Status.CacheState != envoyv1alpha1.RollbackFailedState ||
			!ec.Status.Conditions.IsTrueFor(envoyv1alpha1.CacheOutOfSyncCondition) ||
			!ec.Status.Conditions.IsTrueFor(envoyv1alpha1.RollbackFailedCondition) {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

	// TODO:test the clearance of the Rollback failed condition
	t.Run("Set RollbackFailed state if all versions are tainted", func(t *testing.T) {
		ec := &envoyv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
			Spec: envoyv1alpha1.EnvoyConfigSpec{
				NodeID: "node1",
				EnvoyResources: &envoyv1alpha1.EnvoyResources{
					Endpoints: []envoyv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
					},
				},
			},
			Status: envoyv1alpha1.EnvoyConfigStatus{
				CacheState: envoyv1alpha1.RollbackFailedState,
				ConfigRevisions: []envoyv1alpha1.ConfigRevisionRef{
					{Version: "aaaa", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
				},
				Conditions: status.NewConditions(status.Condition{
					Type:   envoyv1alpha1.RollbackFailedCondition,
					Status: corev1.ConditionTrue,
				}),
			},
		}
		ecrList := &envoyv1alpha1.EnvoyConfigRevisionList{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
			Items: []envoyv1alpha1.EnvoyConfigRevision{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ecr1",
						Namespace: "default",
						Labels:    map[string]string{nodeIDTag: "node1", versionTag: "aaaa"},
					},
					Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:         "node1",
						Version:        "aaaa",
						EnvoyResources: &envoyv1alpha1.EnvoyResources{},
					},
					Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
						Conditions: status.NewConditions(
							status.Condition{
								Type:   envoyv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							},
						),
					}},
			},
		}

		r := &EnvoyConfigReconciler{
			Client:   fake.NewFakeClient(ec, ecrList),
			Scheme:   s,
			XdsCache: fakeCacheV2(),
			Log:      ctrl.Log.WithName("test"),
		}
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "ec",
				Namespace: "default",
			},
		}

		_, gotErr := r.Reconcile(req)
		if gotErr != nil {
			t.Errorf("EnvoyConfigReconcilerRevision.Reconcile() error = %v", gotErr)
			return
		}

		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
		if ec.Status.Conditions.IsTrueFor(envoyv1alpha1.RollbackFailedCondition) ||
			ec.Status.Conditions.IsTrueFor(envoyv1alpha1.CacheOutOfSyncCondition) ||
			ec.Status.CacheState != envoyv1alpha1.InSyncState {
			t.Errorf("Status was not correctly updated")
			return
		}
	})

}

func TestEnvoyConfigReconciler_Reconcile_Finalizer(t *testing.T) {

	cr := &envoyv1alpha1.EnvoyConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ec",
			Namespace:         "default",
			DeletionTimestamp: func() *metav1.Time { t := metav1.Now(); return &t }(),
			Finalizers:        []string{envoyv1alpha1.EnvoyConfigFinalizer},
		},
		Spec: envoyv1alpha1.EnvoyConfigSpec{
			NodeID:         "node1",
			EnvoyResources: &envoyv1alpha1.EnvoyResources{},
		}}

	s := scheme.Scheme
	s.AddKnownTypes(envoyv1alpha1.GroupVersion,
		cr,
		&envoyv1alpha1.EnvoyConfigRevisionList{},
		&envoyv1alpha1.EnvoyConfigRevision{},
	)
	cl := fake.NewFakeClient(cr)
	r := &EnvoyConfigReconciler{Client: cl, Scheme: s, XdsCache: fakeCacheV2(), Log: ctrl.Log.WithName("test")}
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "ec",
			Namespace: "default",
		},
	}

	_, gotErr := r.Reconcile(req)

	if gotErr != nil {
		t.Errorf("EnvoyConfigReconciler.Reconcile_Finalizer() error = %v", gotErr)
		return
	}
	_, err := r.XdsCache.GetSnapshot(cr.Spec.NodeID)
	if err == nil {
		t.Errorf("EnvoyConfigReconciler.Reconcile_Finalizer() - snapshot still exists in the ads server cache")
		return
	}

	ec := &envoyv1alpha1.EnvoyConfig{}
	cl.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
	if len(ec.GetObjectMeta().GetFinalizers()) != 0 {
		t.Errorf("EnvoyConfigReconciler.Reconcile_Finalizer() - finalizer not deleted from object")
		return
	}

}

func TestEnvoyConfigReconciler_finalizeEnvoyConfig(t *testing.T) {
	type fields struct {
		client   client.Client
		scheme   *runtime.Scheme
		xdsCache xdss.Cache
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
				xdsCache: fakeCacheV2(),
			},
			args: args{"node1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EnvoyConfigReconciler{
				Client:   tt.fields.client,
				Scheme:   tt.fields.scheme,
				XdsCache: tt.fields.xdsCache,
				Log:      ctrl.Log.WithName("test"),
			}
			r.finalizeEnvoyConfig(tt.args.nodeID)
			if _, err := r.XdsCache.GetSnapshot(tt.args.nodeID); err == nil {
				t.Errorf("TestEnvoyConfigReconciler_finalizeEnvoyConfig() -> snapshot still in the cache")
			}
		})
	}
}

func TestEnvoyConfigReconciler_addFinalizer(t *testing.T) {
	tests := []struct {
		name    string
		cr      *envoyv1alpha1.EnvoyConfig
		wantErr bool
	}{
		{
			name: "Adds finalizer to NodecacheConfig",
			cr: &envoyv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: envoyv1alpha1.EnvoyConfigSpec{
					NodeID:         "node1",
					EnvoyResources: &envoyv1alpha1.EnvoyResources{},
				}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scheme.Scheme
			s.AddKnownTypes(envoyv1alpha1.GroupVersion, tt.cr)
			cl := fake.NewFakeClient(tt.cr)
			r := &EnvoyConfigReconciler{Client: cl, Scheme: s, XdsCache: fakeCacheV2(), Log: ctrl.Log.WithName("test")}

			if err := r.addFinalizer(context.TODO(), tt.cr); (err != nil) != tt.wantErr {
				t.Errorf("EnvoyConfigReconciler.addFinalizer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				ec := &envoyv1alpha1.EnvoyConfig{}
				r.Client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, ec)
				if len(ec.ObjectMeta.Finalizers) != 1 {
					t.Error("EnvoyConfigReconciler.addFinalizer() wrong number of finalizers present in object")
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

func TestEnvoyConfigReconciler_getVersionToPublish(t *testing.T) {

	tests := []struct {
		name    string
		ec      *envoyv1alpha1.EnvoyConfig
		ecrList *envoyv1alpha1.EnvoyConfigRevisionList
		want    string
		wantErr bool
	}{
		{
			name: "Returns the desiredVersion on seeing a new version",
			ec: &envoyv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: envoyv1alpha1.EnvoyConfigSpec{
					NodeID:         "node1",
					EnvoyResources: &envoyv1alpha1.EnvoyResources{},
				},
				Status: envoyv1alpha1.EnvoyConfigStatus{
					ConfigRevisions: []envoyv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
					},
				},
			},
			ecrList: &envoyv1alpha1.EnvoyConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []envoyv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "xxx",
							EnvoyResources: &envoyv1alpha1.EnvoyResources{},
						},
					},
				},
			},
			want:    "xxx",
			wantErr: false,
		},
		{
			name: "Returns the highest index untainted revision of the ConfigRevision list",
			ec: &envoyv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: envoyv1alpha1.EnvoyConfigSpec{
					NodeID:         "node1",
					EnvoyResources: &envoyv1alpha1.EnvoyResources{},
				},
				Status: envoyv1alpha1.EnvoyConfigStatus{
					ConfigRevisions: []envoyv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
						{Version: "zzz", Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
					},
				},
			},
			ecrList: &envoyv1alpha1.EnvoyConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []envoyv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "xxx",
							EnvoyResources: &envoyv1alpha1.EnvoyResources{},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr2",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "zzz",
							EnvoyResources: &envoyv1alpha1.EnvoyResources{},
						},
						Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   envoyv1alpha1.RevisionTaintedCondition,
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
			ec: &envoyv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: envoyv1alpha1.EnvoyConfigSpec{
					NodeID:         "node1",
					EnvoyResources: &envoyv1alpha1.EnvoyResources{},
				},
				Status: envoyv1alpha1.EnvoyConfigStatus{
					ConfigRevisions: []envoyv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
						{Version: "zzz", Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
					},
				},
			},
			ecrList: &envoyv1alpha1.EnvoyConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []envoyv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "xxx",
							EnvoyResources: &envoyv1alpha1.EnvoyResources{},
						},
						Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   envoyv1alpha1.RevisionTaintedCondition,
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
						Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "zzz",
							EnvoyResources: &envoyv1alpha1.EnvoyResources{},
						},
						Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   envoyv1alpha1.RevisionTaintedCondition,
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
			r := &EnvoyConfigReconciler{
				Client:   fake.NewFakeClient(tt.ec, tt.ecrList),
				Scheme:   s,
				XdsCache: fakeCacheV2(),
				Log:      ctrl.Log.WithName("test"),
			}
			got, err := r.getVersionToPublish(context.TODO(), tt.ec)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnvoyConfigReconciler.getVersionToPublish() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EnvoyConfigReconciler.getVersionToPublish() = %v, want %v", got, tt.want)
			}
		})

	}
}
