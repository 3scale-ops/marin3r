package reconcilers

import (
	"context"
	"reflect"
	"testing"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/envoy"
	"github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/filters"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var s *runtime.Scheme = scheme.Scheme

func init() {
	s.AddKnownTypes(marin3rv1alpha1.GroupVersion,
		&marin3rv1alpha1.EnvoyConfigRevision{},
		&marin3rv1alpha1.EnvoyConfigRevisionList{},
		&marin3rv1alpha1.EnvoyConfig{},
	)
}

func testRevisionReconcilerBuilder(s *runtime.Scheme, instance *marin3rv1alpha1.EnvoyConfig, objs ...runtime.Object) RevisionReconciler {
	return RevisionReconciler{context.TODO(), ctrl.Log.WithName("test"), fake.NewFakeClientWithScheme(s, objs...), s, instance, nil}
}

func TestNewRevisionReconciler(t *testing.T) {
	type args struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		s      *runtime.Scheme
		ec     *marin3rv1alpha1.EnvoyConfig
	}
	tests := []struct {
		name string
		args args
		want RevisionReconciler
	}{
		{
			name: "Returns a RevisionReconciler",
			args: args{context.TODO(), nil, fake.NewFakeClient(), s, nil},
			want: RevisionReconciler{context.TODO(), nil, fake.NewFakeClient(), s, nil, nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRevisionReconciler(tt.args.ctx, tt.args.logger, tt.args.client, tt.args.s, tt.args.ec); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRevisionReconciler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRevisionReconciler_Instance(t *testing.T) {
	tests := []struct {
		name string
		r    RevisionReconciler
		want *marin3rv1alpha1.EnvoyConfig
	}{
		{
			"Returns the EnvoyConfig instance to reconcile",
			testRevisionReconcilerBuilder(s, &marin3rv1alpha1.EnvoyConfig{}),
			&marin3rv1alpha1.EnvoyConfig{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Instance(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RevisionReconciler.Instance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRevisionReconciler_Namespace(t *testing.T) {
	tests := []struct {
		name string
		r    RevisionReconciler
		want string
	}{
		{
			"Returns the namespace of the EnvoyConfig instance to reconcile",
			testRevisionReconcilerBuilder(s, &marin3rv1alpha1.EnvoyConfig{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}}),
			"test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Namespace(); got != tt.want {
				t.Errorf("RevisionReconciler.Namespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRevisionReconciler_NodeID(t *testing.T) {
	tests := []struct {
		name string
		r    RevisionReconciler
		want string
	}{
		{
			"Returns the nodeID of the EnvoyConfig instance to reconcile",
			testRevisionReconcilerBuilder(s, &marin3rv1alpha1.EnvoyConfig{Spec: marin3rv1alpha1.EnvoyConfigSpec{NodeID: "test"}}),
			"test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.NodeID(); got != tt.want {
				t.Errorf("RevisionReconciler.NodeID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRevisionReconciler_Version(t *testing.T) {
	tests := []struct {
		name string
		r    RevisionReconciler
		want string
	}{
		{
			"Returns the calculated version of the EnvoyConfig instance to reconcile",
			testRevisionReconcilerBuilder(s,
				&marin3rv1alpha1.EnvoyConfig{
					Spec: marin3rv1alpha1.EnvoyConfigSpec{
						EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
					},
				}),
			(&marin3rv1alpha1.EnvoyConfig{
				Spec: marin3rv1alpha1.EnvoyConfigSpec{
					EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
				},
			}).GetEnvoyResourcesVersion(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Version(); got != tt.want {
				t.Errorf("RevisionReconciler.Version() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRevisionReconciler_EnvoyAPI(t *testing.T) {
	tests := []struct {
		name string
		r    RevisionReconciler
		want envoy.APIVersion
	}{
		{
			"Returns the envoy API version of the EnvoyConfig instance to reconcile",
			testRevisionReconcilerBuilder(s, &marin3rv1alpha1.EnvoyConfig{Spec: marin3rv1alpha1.EnvoyConfigSpec{EnvoyAPI: pointer.StringPtr("v3")}}),
			envoy.APIv3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.EnvoyAPI(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RevisionReconciler.EnvoyAPI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRevisionReconciler_ListRevisions(t *testing.T) {
	tests := []struct {
		name      string
		r         RevisionReconciler
		filters   []filters.RevisionFilter
		wantCount int
		wantErr   bool
	}{
		{
			"Returns all EnvoyConfigRevisions for the nodeID",
			testRevisionReconcilerBuilder(s,
				&marin3rv1alpha1.EnvoyConfig{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr1",
					Namespace: "test",
					Labels:    map[string]string{filters.NodeIDTag: "test"},
				}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr2",
					Namespace: "test",
					Labels:    map[string]string{filters.NodeIDTag: "test"},
				}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr3",
					Namespace: "test",
					Labels:    map[string]string{filters.NodeIDTag: "other"},
				}},
			),
			[]filters.RevisionFilter{filters.ByNodeID("test")},
			2,
			false,
		},
		{
			"Returns all EnvoyConfigRevisions for the nodeID and version",
			testRevisionReconcilerBuilder(s,
				&marin3rv1alpha1.EnvoyConfig{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr1",
					Namespace: "test",
					Labels:    map[string]string{filters.NodeIDTag: "test", filters.VersionTag: "1"},
				}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr2",
					Namespace: "test",
					Labels:    map[string]string{filters.NodeIDTag: "test", filters.VersionTag: "2"},
				}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr3",
					Namespace: "test",
					Labels:    map[string]string{filters.NodeIDTag: "test", filters.VersionTag: "3"},
				}},
			),
			[]filters.RevisionFilter{filters.ByNodeID("test"), filters.ByVersion("1")},
			1,
			false,
		},
		{
			"Only returns revisions in the same Namespace",
			testRevisionReconcilerBuilder(s,
				&marin3rv1alpha1.EnvoyConfig{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr",
					Namespace: "test",
					Labels:    map[string]string{filters.NodeIDTag: "test"},
				}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr",
					Namespace: "other",
					Labels:    map[string]string{filters.NodeIDTag: "test"},
				}},
			),
			[]filters.RevisionFilter{filters.ByNodeID("test")},
			1,
			false,
		},
		{
			"Returns an error if no revisions are found that match the provided filters",
			testRevisionReconcilerBuilder(s,
				&marin3rv1alpha1.EnvoyConfig{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}},
			),
			[]filters.RevisionFilter{filters.ByNodeID("test")},
			0,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.ListRevisions(tt.filters...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RevisionReconciler.ListRevisions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got.Items) != tt.wantCount {
				t.Errorf("RevisionReconciler.ListRevisions() = %v, want %v", len(got.Items), tt.wantCount)
			}
		})
	}
}

func TestRevisionReconciler_GetRevision(t *testing.T) {
	tests := []struct {
		name    string
		r       RevisionReconciler
		filters []filters.RevisionFilter
		want    *marin3rv1alpha1.EnvoyConfigRevision
		wantErr bool
	}{
		{
			"Returns all the EnvoyConfigRevision that matches the given filters",
			testRevisionReconcilerBuilder(s,
				&marin3rv1alpha1.EnvoyConfig{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr1",
					Namespace: "test",
					Labels:    map[string]string{filters.NodeIDTag: "test", filters.VersionTag: "1"},
				}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr2",
					Namespace: "test",
					Labels:    map[string]string{filters.NodeIDTag: "test", filters.VersionTag: "2"},
				}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr3",
					Namespace: "test",
					Labels:    map[string]string{filters.NodeIDTag: "test", filters.VersionTag: "3"},
				}},
			),
			[]filters.RevisionFilter{filters.ByNodeID("test"), filters.ByVersion("1")},
			&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr1",
				Namespace: "test",
				Labels:    map[string]string{filters.NodeIDTag: "test", filters.VersionTag: "1"},
			}},
			false,
		},
		{
			"Returns an error if API returns more than one EnvoyConfigRevision",
			testRevisionReconcilerBuilder(s,
				&marin3rv1alpha1.EnvoyConfig{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr1",
					Namespace: "test",
					Labels:    map[string]string{filters.NodeIDTag: "test", filters.VersionTag: "1"},
				}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr2",
					Namespace: "test",
					Labels:    map[string]string{filters.NodeIDTag: "test", filters.VersionTag: "1"},
				}},
				&marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{
					Name:      "ecr3",
					Namespace: "test",
					Labels:    map[string]string{filters.NodeIDTag: "test", filters.VersionTag: "3"},
				}},
			),
			[]filters.RevisionFilter{filters.ByNodeID("test"), filters.ByVersion("1")},
			nil,
			true,
		},
		{
			"Returns an error if API returns no EnvoyConfigRevision",
			testRevisionReconcilerBuilder(s,
				&marin3rv1alpha1.EnvoyConfig{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}},
			),
			[]filters.RevisionFilter{filters.ByNodeID("test"), filters.ByVersion("1")},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := tt.r.GetRevision(tt.filters...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RevisionReconciler.GetRevision() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RevisionReconciler.GetRevision() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRevisionReconciler_Reconcile(t *testing.T) {
	type fields struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		ec     *marin3rv1alpha1.EnvoyConfig
	}
	tests := []struct {
		name    string
		fields  fields
		want    ctrl.Result
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RevisionReconciler{
				ctx:    tt.fields.ctx,
				logger: tt.fields.logger,
				client: tt.fields.client,
				ec:     tt.fields.ec,
			}
			got, err := r.Reconcile()
			if (err != nil) != tt.wantErr {
				t.Errorf("RevisionReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RevisionReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRevisionReconciler_AreRevisionLabelsOk(t *testing.T) {
	tests := []struct {
		name string
		r    RevisionReconciler
		list *marin3rv1alpha1.EnvoyConfigRevisionList
		want bool
	}{
		{
			"Returns true if labels up to date",
			testRevisionReconcilerBuilder(s, &marin3rv1alpha1.EnvoyConfig{}),
			&marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								filters.NodeIDTag:   "test",
								filters.VersionTag:  "1",
								filters.EnvoyAPITag: "v3",
							}},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:   "test",
							Version:  "1",
							EnvoyAPI: pointer.StringPtr("v3"),
						},
					},
				},
			},
			true,
		},
		{
			"Returns false if at least one EnvoyConfigRevision needs update",
			testRevisionReconcilerBuilder(s, &marin3rv1alpha1.EnvoyConfig{}),
			&marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								filters.NodeIDTag:   "test",
								filters.VersionTag:  "1",
								filters.EnvoyAPITag: "v3",
							}},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:   "test",
							Version:  "1",
							EnvoyAPI: pointer.StringPtr("v3"),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{}},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:   "test",
							Version:  "1",
							EnvoyAPI: pointer.StringPtr("v3"),
						},
					},
				},
			},
			false,
		},
		{
			"Returns false if all EnvoyConfigRevisions needs update",
			testRevisionReconcilerBuilder(s, &marin3rv1alpha1.EnvoyConfig{}),
			&marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:   "test",
							Version:  "1",
							EnvoyAPI: pointer.StringPtr("v3"),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:   "test",
							Version:  "1",
							EnvoyAPI: pointer.StringPtr("v3"),
						},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.AreRevisionLabelsOk(tt.list); got != tt.want {
				t.Errorf("RevisionReconciler.AreRevisionLabelsOk() = %v, want %v", got, tt.want)
			}
		})
	}
}
