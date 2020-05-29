package nodeconfigcache

import (
	"context"
	"reflect"
	"testing"

	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileNodeConfigCache_ensureNodeConfigRevision(t *testing.T) {

	t.Run("Creates a new NodeConfigRevision if one does not exist", func(t *testing.T) {
		ncr := &cachesv1alpha1.NodeConfigRevision{}
		ncrl := &cachesv1alpha1.NodeConfigRevisionList{}
		ncc := &cachesv1alpha1.NodeConfigCache{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ncc",
				Namespace: "default",
			},
			Spec: cachesv1alpha1.NodeConfigCacheSpec{
				NodeID:  "node1",
				Version: "43",
				Resources: &cachesv1alpha1.EnvoyResources{
					Endpoints: []cachesv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
					}},
			}}

		s := scheme.Scheme
		s.AddKnownTypes(cachesv1alpha1.SchemeGroupVersion, ncc, ncr, ncrl)

		cl := fake.NewFakeClient(ncc)
		r := &ReconcileNodeConfigCache{client: cl, scheme: s, adsCache: fakeTestCache()}

		gotErr := r.ensureNodeConfigRevision(context.TODO(), ncc)
		if gotErr != nil {
			t.Errorf("TestReconcileNodeConfigCache_ensureNodeConfigRevision() error = %v", gotErr)
			return
		}

		ncrList := &cachesv1alpha1.NodeConfigRevisionList{}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				nodeIDTag:  ncc.Spec.NodeID,
				versionTag: ncc.Spec.Version,
			},
		})
		r.client.List(context.TODO(), ncrList, &client.ListOptions{LabelSelector: selector})
		if len(ncrList.Items) != 1 {
			t.Errorf("TestReconcileNodeConfigCache_ensureNodeConfigRevision() - no NodeConfigRevision was created")
			return
		}

		if !apiequality.Semantic.DeepEqual(&ncrList.Items[0].Spec.Resources, ncc.Spec.Resources) {
			t.Errorf("TestReconcileNodeConfigCache_ensureNodeConfigRevision() - resources '%v', want '%v'", &ncrList.Items[0].Spec.Resources, ncc.Spec.Resources)
			return
		}
		if ncrList.Items[0].Spec.Version != ncc.Spec.Version {
			t.Errorf("TestReconcileNodeConfigCache_ensureNodeConfigRevision() - version '%v', want '%v'", ncrList.Items[0].Spec.Version, ncc.Spec.Version)
			return
		}
	})

	t.Run("Does not create a new  does not exist", func(t *testing.T) {
		ncr := &cachesv1alpha1.NodeConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ncr",
				Namespace: "default",
				Labels: map[string]string{
					nodeIDTag:  "node1",
					versionTag: "43",
				},
			},
			Spec: cachesv1alpha1.NodeConfigRevisionSpec{
				Version:   "43",
				NodeID:    "node1",
				Resources: cachesv1alpha1.EnvoyResources{},
			},
		}
		ncc := &cachesv1alpha1.NodeConfigCache{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ncc",
				Namespace: "default",
			},
			Spec: cachesv1alpha1.NodeConfigCacheSpec{
				NodeID:    "node1",
				Version:   "43",
				Resources: &cachesv1alpha1.EnvoyResources{},
			}}

		s := scheme.Scheme
		s.AddKnownTypes(cachesv1alpha1.SchemeGroupVersion, ncc, ncr, &cachesv1alpha1.NodeConfigRevisionList{})

		cl := fake.NewFakeClient(ncc, ncr)
		r := &ReconcileNodeConfigCache{client: cl, scheme: s, adsCache: fakeTestCache()}

		gotErr := r.ensureNodeConfigRevision(context.TODO(), ncc)
		if gotErr != nil {
			t.Errorf("TestReconcileNodeConfigCache_ensureNodeConfigRevision() error = %v", gotErr)
			return
		}

		ncrList := &cachesv1alpha1.NodeConfigRevisionList{}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				nodeIDTag:  ncc.Spec.NodeID,
				versionTag: ncc.Spec.Version,
			},
		})
		r.client.List(context.TODO(), ncrList, &client.ListOptions{LabelSelector: selector})
		if len(ncrList.Items) != 1 {
			t.Errorf("TestReconcileNodeConfigCache_ensureNodeConfigRevision() got '%v' ncr objects, expected 1", len(ncrList.Items))
			return
		}
	})

}

func TestReconcileNodeConfigCache_consolidateRevisionList(t *testing.T) {
	t.Run("Consolidates the revision list in the ncc status", func(t *testing.T) {
		ncr := &cachesv1alpha1.NodeConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ncr",
				Namespace: "default",
				Labels: map[string]string{
					nodeIDTag:  "node1",
					versionTag: "43",
				},
			},
			Spec: cachesv1alpha1.NodeConfigRevisionSpec{
				Version:   "43",
				NodeID:    "node1",
				Resources: cachesv1alpha1.EnvoyResources{},
			},
		}
		ncc := &cachesv1alpha1.NodeConfigCache{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ncc",
				Namespace: "default",
			},
			Spec: cachesv1alpha1.NodeConfigCacheSpec{
				NodeID:    "node1",
				Version:   "43",
				Resources: &cachesv1alpha1.EnvoyResources{},
			},
		}

		s := scheme.Scheme
		s.AddKnownTypes(cachesv1alpha1.SchemeGroupVersion, ncc, ncr, &cachesv1alpha1.NodeConfigRevisionList{})

		cl := fake.NewFakeClient(ncc, ncr)
		r := &ReconcileNodeConfigCache{client: cl, scheme: s, adsCache: fakeTestCache()}

		gotErr := r.consolidateRevisionList(context.TODO(), ncc)
		if gotErr != nil {
			t.Errorf("TestReconcileNodeConfigCache_consolidateRevisionList() error = %v", gotErr)
			return
		}

		gotNCC := &cachesv1alpha1.NodeConfigCache{}
		wantConfigRevisions := []cachesv1alpha1.ConfigRevisionRef{
			{Version: "43", Ref: corev1.ObjectReference{Name: "ncr", Namespace: "default"}},
		}
		r.client.Get(context.TODO(), types.NamespacedName{Name: "ncc", Namespace: "default"}, gotNCC)

		if !apiequality.Semantic.DeepEqual(gotNCC.Status.ConfigRevisions, wantConfigRevisions) {
			t.Errorf("TestReconcileNodeConfigCache_consolidateRevisionList() got '%v', want '%v'", gotNCC.Status.ConfigRevisions, wantConfigRevisions)
			return
		}
	})
}

func TestReconcileNodeConfigCache_deleteUnreferencedRevisions(t *testing.T) {
	type args struct {
		ctx context.Context
		ncc *cachesv1alpha1.NodeConfigCache
	}
	tests := []struct {
		name    string
		r       *ReconcileNodeConfigCache
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.r.deleteUnreferencedRevisions(tt.args.ctx, tt.args.ncc); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileNodeConfigCache.deleteUnreferencedRevisions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_trimRevisions(t *testing.T) {
	type args struct {
		list []cachesv1alpha1.ConfigRevisionRef
		max  int
	}
	tests := []struct {
		name string
		args args
		want []cachesv1alpha1.ConfigRevisionRef
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

func Test_revisionName(t *testing.T) {
	type args struct {
		nodeID    string
		version   string
		resources *cachesv1alpha1.EnvoyResources
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := revisionName(tt.args.nodeID, tt.args.version, tt.args.resources); got != tt.want {
				t.Errorf("revisionName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getRevisionIndex(t *testing.T) {
	type args struct {
		version   string
		revisions []cachesv1alpha1.ConfigRevisionRef
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
