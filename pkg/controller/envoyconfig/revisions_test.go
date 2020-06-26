package envoyconfig

import (
	"context"
	"reflect"
	"testing"

	marin3rv1alpha1 "github.com/3scale/marin3r/pkg/apis/marin3r/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileEnvoyConfig_ensureEnvoyConfigRevision(t *testing.T) {

	t.Run("Creates a new EnvoyConfigRevision if one does not exist", func(t *testing.T) {
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ec",
				Namespace: "default",
			},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID: "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{
					Endpoints: []marin3rv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
					}},
			}}

		cl := fake.NewFakeClient(ec)
		r := &ReconcileEnvoyConfig{client: cl, scheme: s, adsCache: fakeTestCache()}

		gotErr := r.ensureEnvoyConfigRevision(context.TODO(), ec, "xxxx")
		if gotErr != nil {
			t.Errorf("TestReconcileEnvoyConfig_ensureEnvoyConfigRevision() error = %v", gotErr)
			return
		}

		ecrList := &marin3rv1alpha1.EnvoyConfigRevisionList{}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				nodeIDTag:  ec.Spec.NodeID,
				versionTag: "xxxx",
			},
		})
		r.client.List(context.TODO(), ecrList, &client.ListOptions{LabelSelector: selector})
		if len(ecrList.Items) != 1 {
			t.Errorf("TestReconcileEnvoyConfig_ensureEnvoyConfigRevision() - no EnvoyConfigRevision was created")
			return
		}

		if !apiequality.Semantic.DeepEqual(ecrList.Items[0].Spec.Resources, ec.Spec.Resources) {
			t.Errorf("TestReconcileEnvoyConfig_ensureEnvoyConfigRevision() - resources '%v', want '%v'", &ecrList.Items[0].Spec.Resources, ec.Spec.Resources)
			return
		}
	})

	t.Run("Publishes an already existent revision", func(t *testing.T) {
		ecr := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr",
				Namespace: "default",
				Labels: map[string]string{
					nodeIDTag:  "node1",
					versionTag: "xxxx",
				},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:   "xxxx",
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
		}
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ec",
				Namespace: "default",
			},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			}}

		cl := fake.NewFakeClient(ec, ecr)
		r := &ReconcileEnvoyConfig{client: cl, scheme: s, adsCache: fakeTestCache()}

		gotErr := r.ensureEnvoyConfigRevision(context.TODO(), ec, "xxxx")
		if gotErr != nil {
			t.Errorf("TestReconcileEnvoyConfig_ensureEnvoyConfigRevision() error = %v", gotErr)
			return
		}

		ecrList := &marin3rv1alpha1.EnvoyConfigRevisionList{}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				nodeIDTag:  ec.Spec.NodeID,
				versionTag: "xxxx",
			},
		})
		r.client.List(context.TODO(), ecrList, &client.ListOptions{LabelSelector: selector})
		if len(ecrList.Items) != 1 {
			t.Errorf("TestReconcileEnvoyConfig_ensureEnvoyConfigRevision() got '%v' ecr objects, expected 1", len(ecrList.Items))
			return
		}
	})

}

func TestReconcileEnvoyConfig_consolidateRevisionList(t *testing.T) {
	t.Run("Consolidates the revision list in the ec status", func(t *testing.T) {
		ecr := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr",
				Namespace: "default",
				Labels: map[string]string{
					nodeIDTag:  "node1",
					versionTag: "xxxx",
				},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:   "xxxx",
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
		}
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ec",
				Namespace: "default",
			},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
		}

		cl := fake.NewFakeClient(ec, ecr)
		r := &ReconcileEnvoyConfig{client: cl, scheme: s, adsCache: fakeTestCache()}

		gotErr := r.consolidateRevisionList(context.TODO(), ec, "xxxx")
		if gotErr != nil {
			t.Errorf("TestReconcileEnvoyConfig_consolidateRevisionList() error = %v", gotErr)
			return
		}

		gotNCC := &marin3rv1alpha1.EnvoyConfig{}
		wantConfigRevisions := []marin3rv1alpha1.ConfigRevisionRef{
			{Version: "xxxx", Ref: corev1.ObjectReference{Name: "ecr", Namespace: "default"}},
		}
		r.client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, gotNCC)

		if !apiequality.Semantic.DeepEqual(gotNCC.Status.ConfigRevisions, wantConfigRevisions) {
			t.Errorf("TestReconcileEnvoyConfig_consolidateRevisionList() got '%v', want '%v'", gotNCC.Status.ConfigRevisions, wantConfigRevisions)
			return
		}
	})

	t.Run("Moves the published revision to the last position of the list", func(t *testing.T) {
		ecr1 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr1",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "1"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:   "1",
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
		}
		ecr2 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr2",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "2"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:   "2",
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
		}
		ecr3 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr3",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "3"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:   "3",
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
		}
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ec",
				Namespace: "default",
			},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigStatus{
				ConfigRevisions: []marin3rv1alpha1.ConfigRevisionRef{
					{Version: "1", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
					{Version: "2", Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
					{Version: "3", Ref: corev1.ObjectReference{Name: "ecr3", Namespace: "default"}},
				},
			},
		}

		cl := fake.NewFakeClient(ec, ecr1, ecr2, ecr3)
		r := &ReconcileEnvoyConfig{client: cl, scheme: s, adsCache: fakeTestCache()}

		gotErr := r.consolidateRevisionList(context.TODO(), ec, "1")
		if gotErr != nil {
			t.Errorf("TestReconcileEnvoyConfig_consolidateRevisionList() error = %v", gotErr)
			return
		}

		gotNCC := &marin3rv1alpha1.EnvoyConfig{}
		wantConfigRevisions := []marin3rv1alpha1.ConfigRevisionRef{
			{Version: "2", Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
			{Version: "3", Ref: corev1.ObjectReference{Name: "ecr3", Namespace: "default"}},
			{Version: "1", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
		}
		r.client.Get(context.TODO(), types.NamespacedName{Name: "ec", Namespace: "default"}, gotNCC)

		if !apiequality.Semantic.DeepEqual(gotNCC.Status.ConfigRevisions, wantConfigRevisions) {
			t.Errorf("TestReconcileEnvoyConfig_consolidateRevisionList() got '%v', want '%v'", gotNCC.Status.ConfigRevisions, wantConfigRevisions)
			return
		}
	})
}

func TestReconcileEnvoyConfig_deleteUnreferencedRevisions(t *testing.T) {
	type args struct {
		ctx context.Context
		ec *marin3rv1alpha1.EnvoyConfig
	}
	tests := []struct {
		name    string
		r       *ReconcileEnvoyConfig
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.r.deleteUnreferencedRevisions(tt.args.ctx, tt.args.ec); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileEnvoyConfig.deleteUnreferencedRevisions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconcileEnvoyConfig_markRevisionPublished(t *testing.T) {
	t.Run("Keeps current revision published", func(t *testing.T) {
		ecr1 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr1",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "1"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:   "1",
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
				Conditions: []status.Condition{{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionFalse}},
			},
		}

		ecr2 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr2",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "2"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:   "2",
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
				Conditions: []status.Condition{{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionTrue}},
			},
		}

		cl := fake.NewFakeClient(ecr1, ecr2)
		r := &ReconcileEnvoyConfig{client: cl, scheme: s, adsCache: fakeTestCache()}

		gotErr := r.markRevisionPublished(context.TODO(), "node1", "2", "reason", "msg")
		if gotErr != nil {
			t.Errorf("TestReconcileEnvoyConfig_markRevisionPublished() error = %v", gotErr)
			return
		}

		ecr := &marin3rv1alpha1.EnvoyConfigRevision{}

		// ecr2 should still be marked as published
		r.client.Get(context.TODO(), types.NamespacedName{Name: "ecr2", Namespace: "default"}, ecr)
		if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			t.Errorf("TestReconcileEnvoyConfig_ensureEnvoyConfigRevision() - ecr2 RevisionPublishedCondition != True or missing")
		}

		// ecr1 should not be marked as published
		r.client.Get(context.TODO(), types.NamespacedName{Name: "ecr1", Namespace: "default"}, ecr)
		if ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			t.Errorf("TestReconcileEnvoyConfig_ensureEnvoyConfigRevision() - ecr1 RevisionPublishedCondition == True")
		}
	})

	t.Run("Changes the published revision", func(t *testing.T) {
		ecr1 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr1",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "1"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:   "1",
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
				Conditions: []status.Condition{{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionFalse}},
			},
		}

		ecr2 := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr2",
				Namespace: "default",
				Labels:    map[string]string{nodeIDTag: "node1", versionTag: "2"},
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				Version:   "2",
				NodeID:    "node1",
				Resources: &marin3rv1alpha1.EnvoyResources{},
			},
			Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
				Conditions: []status.Condition{{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionTrue}},
			},
		}

		cl := fake.NewFakeClient(ecr1, ecr2)
		r := &ReconcileEnvoyConfig{client: cl, scheme: s, adsCache: fakeTestCache()}

		gotErr := r.markRevisionPublished(context.TODO(), "node1", "1", "reason", "msg")
		if gotErr != nil {
			t.Errorf("TestReconcileEnvoyConfig_markRevisionPublished() error = %v", gotErr)
			return
		}

		ecr := &marin3rv1alpha1.EnvoyConfigRevision{}

		// ecr2 should not be marked as published
		r.client.Get(context.TODO(), types.NamespacedName{Name: "ecr2", Namespace: "default"}, ecr)
		if ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			t.Errorf("TestReconcileEnvoyConfig_ensureEnvoyConfigRevision() - ecr2 RevisionPublishedCondition == True")
		}

		// ecr1 should not be marked as published
		r.client.Get(context.TODO(), types.NamespacedName{Name: "ecr1", Namespace: "default"}, ecr)
		if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
			t.Errorf("TestReconcileEnvoyConfig_ensureEnvoyConfigRevision() - ecr1 RevisionPublishedCondition != True or missing")
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

func Test_moveRevisionToLast(t *testing.T) {
	type args struct {
		list []marin3rv1alpha1.ConfigRevisionRef
		idx  int
	}
	tests := []struct {
		name string
		args args
		want []marin3rv1alpha1.ConfigRevisionRef
	}{
		{
			name: "Moves the revision to the last position in the list",
			args: args{
				list: []marin3rv1alpha1.ConfigRevisionRef{
					{Version: "1", Ref: corev1.ObjectReference{}},
					{Version: "2", Ref: corev1.ObjectReference{}},
					{Version: "3", Ref: corev1.ObjectReference{}},
					{Version: "4", Ref: corev1.ObjectReference{}},
					{Version: "5", Ref: corev1.ObjectReference{}},
					{Version: "6", Ref: corev1.ObjectReference{}},
				},
				idx: 3,
			},
			want: []marin3rv1alpha1.ConfigRevisionRef{
				{Version: "1", Ref: corev1.ObjectReference{}},
				{Version: "2", Ref: corev1.ObjectReference{}},
				{Version: "3", Ref: corev1.ObjectReference{}},
				{Version: "5", Ref: corev1.ObjectReference{}},
				{Version: "6", Ref: corev1.ObjectReference{}},
				{Version: "4", Ref: corev1.ObjectReference{}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := moveRevisionToLast(tt.args.list, tt.args.idx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("moveRevisionToLast() = %v, want %v", got, tt.want)
			}
		})
	}
}
