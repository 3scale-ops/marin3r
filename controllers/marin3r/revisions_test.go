package controllers

import (
	"context"
	"reflect"
	"testing"
	"time"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	xdss "github.com/3scale/marin3r/pkg/discoveryservice/xdss"
	xdss_v2 "github.com/3scale/marin3r/pkg/discoveryservice/xdss/v2"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"

	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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

func TestEnvoyConfigReconciler_ensureEnvoyConfigRevision(t *testing.T) {

	t.Run("Creates a new EnvoyConfigRevision if one does not exist", func(t *testing.T) {
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ec",
				Namespace: "default",
			},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID: "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{
					Endpoints: []marin3rv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
					}},
			}}

		cl := fake.NewFakeClient(ec)
		r := &EnvoyConfigReconciler{Client: cl, Scheme: s, Log: ctrl.Log.WithName("test")}

		gotErr := r.ensureEnvoyConfigRevision(context.TODO(), ec, "xxxx", nil)
		if gotErr != nil {
			t.Errorf("TestEnvoyConfigReconciler_ensureEnvoyConfigRevision() error = %v", gotErr)
			return
		}

		ecrList := &marin3rv1alpha1.EnvoyConfigRevisionList{}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				nodeIDTag:  ec.Spec.NodeID,
				versionTag: "xxxx",
			},
		})
		r.Client.List(context.TODO(), ecrList, &client.ListOptions{LabelSelector: selector})
		if len(ecrList.Items) != 1 {
			t.Errorf("TestEnvoyConfigReconciler_ensureEnvoyConfigRevision() - no EnvoyConfigRevision was created")
			return
		}

		if !apiequality.Semantic.DeepEqual(ecrList.Items[0].Spec.EnvoyResources, ec.Spec.EnvoyResources) {
			t.Errorf("TestEnvoyConfigReconciler_ensureEnvoyConfigRevision() - resources '%v', want '%v'", &ecrList.Items[0].Spec.EnvoyResources, ec.Spec.EnvoyResources)
			return
		}
	})

	t.Run("Publishes an already existent revision", func(t *testing.T) {
		ecr := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr",
				Namespace: "default",
				Labels: map[string]string{
					nodeIDTag:   "node1",
					versionTag:  "xxxx",
					envoyAPITag: "v2",
				},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:        "xxxx",
				NodeID:         "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
		}
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ec",
				Namespace: "default",
			},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID:         "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			}}

		cl := fake.NewFakeClient(ec, ecr)
		r := &EnvoyConfigReconciler{Client: cl, Scheme: s, Log: ctrl.Log.WithName("test")}

		gotErr := r.ensureEnvoyConfigRevision(context.TODO(), ec, "xxxx", nil)
		if gotErr != nil {
			t.Errorf("TestEnvoyConfigReconciler_ensureEnvoyConfigRevision() error = %v", gotErr)
			return
		}

		ecrList := &marin3rv1alpha1.EnvoyConfigRevisionList{}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				nodeIDTag:  ec.Spec.NodeID,
				versionTag: "xxxx",
			},
		})
		r.Client.List(context.TODO(), ecrList, &client.ListOptions{LabelSelector: selector})
		if len(ecrList.Items) != 1 {
			t.Errorf("TestEnvoyConfigReconciler_ensureEnvoyConfigRevision() got '%v' ecr objects, expected 1", len(ecrList.Items))
			return
		}
	})

}

func TestEnvoyConfigReconciler_consolidateRevisionList(t *testing.T) {
	t.Run("Generates an ordered list with all the config revisions, with the desired onw on the highes index", func(t *testing.T) {
		ecr1 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr1",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "1", envoyAPITag: "v2"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:        "1",
				NodeID:         "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
				LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.NewTime(time.Now().Add(-4 * time.Second))),
			},
		}
		ecr2 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr2",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "2", envoyAPITag: "v2"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:        "2",
				NodeID:         "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
				LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.NewTime(time.Now().Add(-2 * time.Second))),
			},
		}
		ecr3 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr3",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "3", envoyAPITag: "v2"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:        "3",
				NodeID:         "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
				LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.NewTime(time.Now().Add(-1 * time.Second))),
			},
		}
		ecr4 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "ecr4",
				Namespace:         "default",
				Labels:            map[string]string{nodeIDTag: "node1", versionTag: "1", envoyAPITag: "v2"},
				CreationTimestamp: metav1.NewTime(time.Now().Add(-3 * time.Second)),
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:        "4",
				NodeID:         "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
		}
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ec",
				Namespace: "default",
			},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID:         "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
		}

		cl := fake.NewFakeClient(ec, ecr1, ecr2, ecr3, ecr4)
		r := &EnvoyConfigReconciler{Client: cl, Scheme: s, Log: ctrl.Log.WithName("test")}

		gotErr := r.reconcileRevisionList(context.TODO(), ec, "2", nil)
		if gotErr != nil {
			t.Errorf("TestEnvoyConfigReconciler_consolidateRevisionList() error = %v", gotErr)
			return
		}

		gotNCC := &marin3rv1alpha1.EnvoyConfig{}
		wantConfigRevisions := []marin3rv1alpha1.ConfigRevisionRef{
			{Version: "1", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
			{Version: "4", Ref: corev1.ObjectReference{Name: "ecr4", Namespace: "default"}},
			{Version: "3", Ref: corev1.ObjectReference{Name: "ecr3", Namespace: "default"}},
			{Version: "2", Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
		}
		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, gotNCC)

		if !apiequality.Semantic.DeepEqual(gotNCC.Status.ConfigRevisions, wantConfigRevisions) {
			t.Errorf("TestEnvoyConfigReconciler_consolidateRevisionList() got '%v', want '%v'", gotNCC.Status.ConfigRevisions, wantConfigRevisions)
			return
		}
	})
}

func TestEnvoyConfigReconciler_deleteUnreferencedRevisions(t *testing.T) {
	type args struct {
		ctx context.Context
		ec  *marin3rv1alpha1.EnvoyConfig
	}
	tests := []struct {
		name    string
		r       *EnvoyConfigReconciler
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.r.deleteUnreferencedRevisions(tt.args.ctx, tt.args.ec, nil); (err != nil) != tt.wantErr {
				t.Errorf("EnvoyConfigReconciler.deleteUnreferencedRevisions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvoyConfigReconciler_markRevisionPublished(t *testing.T) {
	t.Run("Keeps current revision published", func(t *testing.T) {
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ec",
				Namespace: "default",
			},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID:         "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
		}
		ecr1 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr1",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "1", envoyAPITag: "v2"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:        "1",
				NodeID:         "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
				Conditions: []status.Condition{{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionFalse}},
			},
		}

		ecr2 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr2",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "2", envoyAPITag: "v2"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:        "2",
				NodeID:         "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
				Conditions: []status.Condition{{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionTrue}},
			},
		}

		cl := fake.NewFakeClient(ecr1, ecr2)
		r := &EnvoyConfigReconciler{Client: cl, Scheme: s, Log: ctrl.Log.WithName("test")}

		gotErr := r.markRevisionPublished(context.TODO(), ec, "2", "reason", "msg", nil)
		if gotErr != nil {
			t.Errorf("TestEnvoyConfigReconciler_markRevisionPublished() error = %v", gotErr)
			return
		}

		ecr := &marin3rv1alpha1.EnvoyConfigRevision{}

		// ecr2 should still be marked as published
		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ecr2", Namespace: "default"}, ecr)
		if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			t.Errorf("TestEnvoyConfigReconciler_ensureEnvoyConfigRevision() - ecr2 RevisionPublishedCondition != True or missing")
		}

		// ecr1 should not be marked as published
		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ecr1", Namespace: "default"}, ecr)
		if ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			t.Errorf("TestEnvoyConfigReconciler_ensureEnvoyConfigRevision() - ecr1 RevisionPublishedCondition == True")
		}
	})

	t.Run("Changes the published revision", func(t *testing.T) {
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ec",
				Namespace: "default",
			},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID:         "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
		}
		ecr1 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr1",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "1", envoyAPITag: "v2"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:        "1",
				NodeID:         "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
				Conditions: []status.Condition{{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionFalse}},
			},
		}

		ecr2 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr2",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "2", envoyAPITag: "v2"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:        "2",
				NodeID:         "node1",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
				Conditions: []status.Condition{{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionTrue}},
			},
		}

		cl := fake.NewFakeClient(ecr1, ecr2)
		r := &EnvoyConfigReconciler{Client: cl, Scheme: s, Log: ctrl.Log.WithName("test")}

		gotErr := r.markRevisionPublished(context.TODO(), ec, "1", "reason", "msg", nil)
		if gotErr != nil {
			t.Errorf("TestEnvoyConfigReconciler_markRevisionPublished() error = %v", gotErr)
			return
		}

		ecr := &marin3rv1alpha1.EnvoyConfigRevision{}

		// ecr2 should not be marked as published
		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ecr2", Namespace: "default"}, ecr)
		if ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			t.Errorf("TestEnvoyConfigReconciler_ensureEnvoyConfigRevision() - ecr2 RevisionPublishedCondition == True")
		}

		// ecr1 should not be marked as published
		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ecr1", Namespace: "default"}, ecr)
		if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			t.Errorf("TestEnvoyConfigReconciler_ensureEnvoyConfigRevision() - ecr1 RevisionPublishedCondition != True or missing")
		}
	})
}

func Test_trimRevisions(t *testing.T) {
	type args struct {
		list []marin3rv1alpha1.ConfigRevisionRef
		max  int
	}
	tests := []struct {
		name string
		args args
		want []marin3rv1alpha1.ConfigRevisionRef
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimRevisions(tt.args.list, tt.args.max); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("trimRevisions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getRevisionIndex(t *testing.T) {
	type args struct {
		version   string
		revisions []marin3rv1alpha1.ConfigRevisionRef
	}
	tests := []struct {
		name string
		args args
		want *int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRevisionIndex(tt.args.version, tt.args.revisions); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getRevisionIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}
