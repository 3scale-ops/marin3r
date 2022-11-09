package reconcilers

import (
	"context"
	"reflect"
	"testing"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/marin3r/envoyconfig/filters"
	"github.com/3scale-ops/marin3r/pkg/util"
	"github.com/go-logr/logr"
	"github.com/go-test/deep"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	return RevisionReconciler{context.TODO(), ctrl.Log.WithName("test"), fake.NewFakeClientWithScheme(s, objs...), s, instance, nil, nil, nil, nil}
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
			args: args{context.TODO(), logr.Logger{}, fake.NewFakeClient(), s, nil},
			want: RevisionReconciler{context.TODO(), logr.Logger{}, fake.NewFakeClient(), s, nil, nil, nil, nil, nil},
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
			if got := tt.r.DesiredVersion(); got != tt.want {
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

func TestRevisionReconciler_DesiredVersion(t *testing.T) {
	type fields struct {
		ctx              context.Context
		logger           logr.Logger
		client           client.Client
		scheme           *runtime.Scheme
		ec               *marin3rv1alpha1.EnvoyConfig
		desiredVersion   *string
		publishedVersion *string
		cacheState       *string
		revisionList     *marin3rv1alpha1.EnvoyConfigRevisionList
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"Returns the DesiredVersion",
			fields{context.TODO(), logr.Logger{}, nil, nil, nil, pointer.StringPtr("xxxx"), nil, nil, nil},
			"xxxx",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RevisionReconciler{
				ctx:              tt.fields.ctx,
				logger:           tt.fields.logger,
				client:           tt.fields.client,
				scheme:           tt.fields.scheme,
				ec:               tt.fields.ec,
				desiredVersion:   tt.fields.desiredVersion,
				publishedVersion: tt.fields.publishedVersion,
				cacheState:       tt.fields.cacheState,
				revisionList:     tt.fields.revisionList,
			}
			if got := r.DesiredVersion(); got != tt.want {
				t.Errorf("RevisionReconciler.DesiredVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRevisionReconciler_GetRevisionList(t *testing.T) {
	type fields struct {
		ctx              context.Context
		logger           logr.Logger
		client           client.Client
		scheme           *runtime.Scheme
		ec               *marin3rv1alpha1.EnvoyConfig
		desiredVersion   *string
		publishedVersion *string
		cacheState       *string
		revisionList     *marin3rv1alpha1.EnvoyConfigRevisionList
	}
	tests := []struct {
		name   string
		fields fields
		want   *marin3rv1alpha1.EnvoyConfigRevisionList
	}{
		{
			"Returns the revision list",
			fields{context.TODO(), logr.Logger{}, nil, nil, nil, nil, nil, nil, &marin3rv1alpha1.EnvoyConfigRevisionList{}},
			&marin3rv1alpha1.EnvoyConfigRevisionList{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RevisionReconciler{
				ctx:              tt.fields.ctx,
				logger:           tt.fields.logger,
				client:           tt.fields.client,
				scheme:           tt.fields.scheme,
				ec:               tt.fields.ec,
				desiredVersion:   tt.fields.desiredVersion,
				publishedVersion: tt.fields.publishedVersion,
				cacheState:       tt.fields.cacheState,
				revisionList:     tt.fields.revisionList,
			}
			if got := r.GetRevisionList(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RevisionReconciler.GetRevisionList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRevisionReconciler_PublishedVersion(t *testing.T) {
	type fields struct {
		ctx              context.Context
		logger           logr.Logger
		client           client.Client
		scheme           *runtime.Scheme
		ec               *marin3rv1alpha1.EnvoyConfig
		desiredVersion   *string
		publishedVersion *string
		cacheState       *string
		revisionList     *marin3rv1alpha1.EnvoyConfigRevisionList
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"Returns the PublishedVersion",
			fields{context.TODO(), logr.Logger{}, nil, nil, nil, nil, pointer.StringPtr("xxxx"), nil, nil},
			"xxxx",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RevisionReconciler{
				ctx:              tt.fields.ctx,
				logger:           tt.fields.logger,
				client:           tt.fields.client,
				scheme:           tt.fields.scheme,
				ec:               tt.fields.ec,
				desiredVersion:   tt.fields.desiredVersion,
				publishedVersion: tt.fields.publishedVersion,
				cacheState:       tt.fields.cacheState,
				revisionList:     tt.fields.revisionList,
			}
			if got := r.PublishedVersion(); got != tt.want {
				t.Errorf("RevisionReconciler.PublishedVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRevisionReconciler_GetCacheState(t *testing.T) {
	type fields struct {
		ctx              context.Context
		logger           logr.Logger
		client           client.Client
		scheme           *runtime.Scheme
		ec               *marin3rv1alpha1.EnvoyConfig
		desiredVersion   *string
		publishedVersion *string
		cacheState       *string
		revisionList     *marin3rv1alpha1.EnvoyConfigRevisionList
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"Returns the CacheState",
			fields{context.TODO(), logr.Logger{}, nil, nil, nil, nil, nil, pointer.StringPtr(marin3rv1alpha1.InSyncState), nil},
			marin3rv1alpha1.InSyncState,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RevisionReconciler{
				ctx:              tt.fields.ctx,
				logger:           tt.fields.logger,
				client:           tt.fields.client,
				scheme:           tt.fields.scheme,
				ec:               tt.fields.ec,
				desiredVersion:   tt.fields.desiredVersion,
				publishedVersion: tt.fields.publishedVersion,
				cacheState:       tt.fields.cacheState,
				revisionList:     tt.fields.revisionList,
			}
			if got := r.GetCacheState(); got != tt.want {
				t.Errorf("RevisionReconciler.GetCacheState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRevisionReconciler_Reconcile(t *testing.T) {
	type fields struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		scheme *runtime.Scheme
		ec     *marin3rv1alpha1.EnvoyConfig
	}
	tests := []struct {
		name    string
		fields  fields
		want    ctrl.Result
		wantErr bool
	}{
		{
			name: "Creates a new EnvoyConfigRevision, no error and requeue",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s),
				scheme: s,
				ec: &marin3rv1alpha1.EnvoyConfig{
					TypeMeta:   metav1.TypeMeta{Kind: "EnvoyConfig", APIVersion: "v1alpha1"},
					ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "test"},
					Spec: marin3rv1alpha1.EnvoyConfigSpec{
						NodeID:         "node",
						EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
					},
				},
			},
			want:    ctrl.Result{Requeue: true},
			wantErr: false,
		},
		{
			name: "Multiple EnvoyConfigRevision for current version, error and requeue",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s,
					&marin3rv1alpha1.EnvoyConfigRevision{
						TypeMeta: metav1.TypeMeta{Kind: "EnvoyConfigRevision", APIVersion: "v1alpha1"},
						ObjectMeta: metav1.ObjectMeta{
							Name: "ecr1", Namespace: "test",
							Labels: map[string]string{
								filters.NodeIDTag:   "node",
								filters.EnvoyAPITag: envoy.APIv3.String(),
								filters.VersionTag:  util.Hash(&marin3rv1alpha1.EnvoyResources{}),
							},
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{},
					},
					&marin3rv1alpha1.EnvoyConfigRevision{
						TypeMeta: metav1.TypeMeta{Kind: "EnvoyConfigRevision", APIVersion: "v1alpha1"},
						ObjectMeta: metav1.ObjectMeta{
							Name: "ecr2", Namespace: "test",
							Labels: map[string]string{
								filters.NodeIDTag:   "node",
								filters.EnvoyAPITag: envoy.APIv3.String(),
								filters.VersionTag:  util.Hash(&marin3rv1alpha1.EnvoyResources{}),
							},
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{},
					},
				),
				scheme: s,
				ec: &marin3rv1alpha1.EnvoyConfig{
					TypeMeta:   metav1.TypeMeta{Kind: "EnvoyConfig", APIVersion: "v1alpha1"},
					ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "test"},
					Spec: marin3rv1alpha1.EnvoyConfigSpec{
						NodeID:         "node",
						EnvoyAPI:       pointer.StringPtr(envoy.APIv3.String()),
						EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
					},
				},
			},
			want:    ctrl.Result{},
			wantErr: true,
		},
		{
			name: "EnvoyConfigRevision exists for current version, reconcile withiout error or requeue",
			fields: fields{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(s,
					&marin3rv1alpha1.EnvoyConfigRevision{
						TypeMeta: metav1.TypeMeta{Kind: "EnvoyConfigRevision", APIVersion: "v1alpha1"},
						ObjectMeta: metav1.ObjectMeta{
							Name: "ecr1", Namespace: "test",
							Labels: map[string]string{
								filters.NodeIDTag:   "node",
								filters.EnvoyAPITag: envoy.APIv3.String(),
								filters.VersionTag:  util.Hash(&marin3rv1alpha1.EnvoyResources{}),
							},
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{},
					},
				),
				scheme: s,
				ec: &marin3rv1alpha1.EnvoyConfig{
					TypeMeta:   metav1.TypeMeta{Kind: "EnvoyConfig", APIVersion: "v1alpha1"},
					ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "test"},
					Spec: marin3rv1alpha1.EnvoyConfigSpec{
						NodeID:         "node",
						EnvoyAPI:       pointer.StringPtr(envoy.APIv3.String()),
						EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
					},
				},
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RevisionReconciler{
				ctx:    tt.fields.ctx,
				logger: tt.fields.logger,
				client: tt.fields.client,
				scheme: tt.fields.scheme,
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

func TestRevisionReconciler_getVersionToPublish(t *testing.T) {
	tests := []struct {
		name           string
		revisionList   *marin3rv1alpha1.EnvoyConfigRevisionList
		wantVersion    string
		wantCacheState string
	}{
		{
			name: "Returns expected version and InSync state",
			revisionList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "aaaa"}},
					{Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "xxxx"}},
				},
			},
			wantVersion:    "xxxx",
			wantCacheState: marin3rv1alpha1.InSyncState,
		},
		{
			name: "Returns expected version and Rollback state",
			revisionList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "aaaa"}},
					{Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "xxxx"},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: []metav1.Condition{{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
								Status: metav1.ConditionTrue,
							}}}},
				},
			},
			wantVersion:    "aaaa",
			wantCacheState: marin3rv1alpha1.RollbackState,
		},
		{
			name: "Returns no version and and RollbackFailed state",
			revisionList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "aaaa"},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: []metav1.Condition{{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
								Status: metav1.ConditionTrue,
							}}}},
					{Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "xxxx"},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: []metav1.Condition{{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
								Status: metav1.ConditionTrue,
							}}}},
				},
			},
			wantVersion:    "",
			wantCacheState: marin3rv1alpha1.RollbackFailedState,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := testRevisionReconcilerBuilder(s, &marin3rv1alpha1.EnvoyConfig{})
			r.revisionList = tt.revisionList
			gotVersion, gotCacheState := r.getVersionToPublish()
			if gotVersion != tt.wantVersion {
				t.Errorf("RevisionReconciler.getVersionToPublish() got = %v, want %v", gotVersion, tt.wantVersion)
			}
			if gotCacheState != tt.wantCacheState {
				t.Errorf("RevisionReconciler.getVersionToPublish() got1 = %v, want %v", gotCacheState, tt.wantCacheState)
			}
		})
	}
}

func TestRevisionReconciler_isRevisionPublishedConditionReconciled(t *testing.T) {
	tests := []struct {
		name             string
		r                RevisionReconciler
		revisionList     *marin3rv1alpha1.EnvoyConfigRevisionList
		versionToPublish string
		wantPublished    *types.NamespacedName
		wantUnpublished  *[]types.NamespacedName
	}{
		{
			name: "Sets RevisionPublished condition in revision",
			r:    testRevisionReconcilerBuilder(s, &marin3rv1alpha1.EnvoyConfig{}),
			revisionList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ecr",
						Namespace: "test",
					},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						Version: "xxxx",
					},
					Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{},
				}},
			},
			versionToPublish: "xxxx",
			wantPublished:    &types.NamespacedName{Name: "ecr", Namespace: "test"},
			wantUnpublished:  &[]types.NamespacedName{},
		},
		{
			name: "Removes RevisionPublished condition in revisions that require it",
			r:    testRevisionReconcilerBuilder(s, &marin3rv1alpha1.EnvoyConfig{}),
			revisionList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "test",
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "xxxx",
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: []metav1.Condition{{
								Type:   marin3rv1alpha1.RevisionPublishedCondition,
								Status: metav1.ConditionTrue,
							}},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr2",
							Namespace: "test",
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "aaaa",
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: []metav1.Condition{},
						},
					},
				},
			},
			versionToPublish: "zzzz",
			wantPublished:    nil,
			wantUnpublished:  &[]types.NamespacedName{{Name: "ecr1", Namespace: "test"}},
		},
		{
			name: "Removes RevisionPublished condition in revisions",
			r:    testRevisionReconcilerBuilder(s, &marin3rv1alpha1.EnvoyConfig{}),
			revisionList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ecr",
						Namespace: "test",
					},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						Version: "xxxx",
					},
					Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
						Conditions: []metav1.Condition{{
							Type:   marin3rv1alpha1.RevisionPublishedCondition,
							Status: metav1.ConditionTrue,
						}},
					},
				}},
			},
			versionToPublish: "zzzz",
			wantPublished:    nil,
			wantUnpublished:  &[]types.NamespacedName{{Name: "ecr", Namespace: "test"}},
		},
		{
			name: "Returns nil when no changes",
			r:    testRevisionReconcilerBuilder(s, &marin3rv1alpha1.EnvoyConfig{}),
			revisionList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ecr",
						Namespace: "test",
					},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						Version: "xxxx",
					},
					Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
						Conditions: []metav1.Condition{{
							Type:   marin3rv1alpha1.RevisionPublishedCondition,
							Status: metav1.ConditionTrue,
						}},
					},
				}},
			},
			versionToPublish: "xxxx",
			wantPublished:    nil,
			wantUnpublished:  nil,
		},
		{
			name: "Publish a previous revision",
			r:    testRevisionReconcilerBuilder(s, &marin3rv1alpha1.EnvoyConfig{}),
			revisionList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "yyyy",
							Namespace: "test",
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "yyyy",
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "xxxx",
							Namespace: "test",
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "xxxx",
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: []metav1.Condition{{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
								Status: metav1.ConditionTrue,
							}},
						},
					},
				},
			},
			versionToPublish: "yyyy",
			wantPublished:    &types.NamespacedName{Name: "yyyy", Namespace: "test"},
			wantUnpublished:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.revisionList = tt.revisionList
			got, got1 := tt.r.isRevisionPublishedConditionReconciled(tt.versionToPublish)

			// Check the published ecr is ok
			if tt.wantPublished == nil {
				if got != nil {
					t.Errorf("RevisionReconciler.isRevisionPublishedConditionReconciled() gotPublished '%v', wanted 'nil'", types.NamespacedName{Name: got.GetName(), Namespace: got.GetNamespace()})
				}
			} else {
				if got.GetName() != tt.wantPublished.Name || got.GetNamespace() != tt.wantPublished.Namespace || !meta.IsStatusConditionTrue(got.Status.Conditions, marin3rv1alpha1.RevisionPublishedCondition) {
					t.Errorf("RevisionReconciler.isRevisionPublishedConditionReconciled() gotPublished '%v', wantPublished %v", types.NamespacedName{Name: got.GetName(), Namespace: got.GetNamespace()}, tt.wantPublished)
				}
			}

			// Check the unpublished ecr are correct
			if tt.wantUnpublished == nil {
				if got1 != nil {
					t.Errorf("RevisionReconciler.isRevisionPublishedConditionReconciled() gotUnpublished '%v', wantUnpublished 'nil'", got)
				}
			} else {

				if len(*tt.wantUnpublished) != len(got1) {
					t.Errorf("RevisionReconciler.isRevisionPublishedConditionReconciled() got wrong number of unpublished revisions")
				}
				for idx, ecr := range got1 {
					if ecr.GetName() != (*tt.wantUnpublished)[idx].Name || ecr.GetNamespace() != (*tt.wantUnpublished)[idx].Namespace || meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionPublishedCondition) {
						t.Errorf("RevisionReconciler.isRevisionPublishedConditionReconciled() gotUnpublished '%v', wantUnpublished %v", ecr, (*tt.wantUnpublished)[idx])
					}
				}
			}
		})
	}
}

func TestRevisionReconciler_newRevisionForCurrentResources(t *testing.T) {
	tests := []struct {
		name string
		r    RevisionReconciler
		want *marin3rv1alpha1.EnvoyConfigRevision
	}{
		{
			name: "Generates a new EnvoyConfigRevision v3 for current EnvoyConfig resources",
			r: testRevisionReconcilerBuilder(s,
				&marin3rv1alpha1.EnvoyConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ec",
						Namespace: "test",
					},
					Spec: marin3rv1alpha1.EnvoyConfigSpec{
						NodeID: "node",
						EnvoyResources: &marin3rv1alpha1.EnvoyResources{
							Endpoints: []marin3rv1alpha1.EnvoyResource{
								{Name: pointer.String("endpoint"), Value: "{\"cluster_name\": \"correct_endpoint\"}"},
							},
						},
					},
				},
			),
			want: &marin3rv1alpha1.EnvoyConfigRevision{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "node-v3-6c554ddcb8",
					Namespace: "test",
					Labels: map[string]string{
						filters.EnvoyAPITag: envoy.APIv3.String(),
						filters.NodeIDTag:   "node",
						filters.VersionTag:  "6c554ddcb8",
					},
				},
				Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
					NodeID:        "node",
					EnvoyAPI:      pointer.StringPtr(envoy.APIv3.String()),
					Version:       "6c554ddcb8",
					Serialization: pointer.StringPtr(string(envoy_serializer.JSON)),
					EnvoyResources: &marin3rv1alpha1.EnvoyResources{
						Endpoints: []marin3rv1alpha1.EnvoyResource{
							{Name: pointer.String("endpoint"), Value: "{\"cluster_name\": \"correct_endpoint\"}"},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := deep.Equal(tt.r.newRevisionForCurrentResources(), tt.want); len(diff) > 0 {
				t.Errorf("RevisionReconciler.newRevisionForCurrentResources() = diff %v", diff)
			}
		})
	}
}

func TestRevisionReconciler_isRevisionRetentionReconciled(t *testing.T) {
	type fields struct {
		ctx              context.Context
		logger           logr.Logger
		client           client.Client
		scheme           *runtime.Scheme
		ec               *marin3rv1alpha1.EnvoyConfig
		desiredVersion   *string
		publishedVersion *string
		cacheState       *string
		revisionList     *marin3rv1alpha1.EnvoyConfigRevisionList
	}
	type args struct {
		retention int
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantTrimmed []marin3rv1alpha1.EnvoyConfigRevision
		wantList    *marin3rv1alpha1.EnvoyConfigRevisionList
	}{
		{
			name: "Resulting list has 'retention' elements and returns trimmed elements",
			fields: fields{nil, logr.Logger{}, nil, nil, nil, nil, nil, nil,
				&marin3rv1alpha1.EnvoyConfigRevisionList{
					Items: []marin3rv1alpha1.EnvoyConfigRevision{
						{ObjectMeta: metav1.ObjectMeta{Name: "ecr1"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "ecr2"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "ecr3"}},
					},
				},
			},
			args: args{retention: 1},
			wantTrimmed: []marin3rv1alpha1.EnvoyConfigRevision{
				{ObjectMeta: metav1.ObjectMeta{Name: "ecr1"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "ecr2"}},
			},
			wantList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{ObjectMeta: metav1.ObjectMeta{Name: "ecr3"}},
				},
			},
		},
		{
			name: "List is not modified if elements within 'retention' parameter",
			fields: fields{nil, logr.Logger{}, nil, nil, nil, nil, nil, nil,
				&marin3rv1alpha1.EnvoyConfigRevisionList{
					Items: []marin3rv1alpha1.EnvoyConfigRevision{
						{ObjectMeta: metav1.ObjectMeta{Name: "ecr1"}},
					},
				},
			},
			args:        args{retention: 1},
			wantTrimmed: []marin3rv1alpha1.EnvoyConfigRevision{},
			wantList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{ObjectMeta: metav1.ObjectMeta{Name: "ecr1"}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RevisionReconciler{
				ctx:              tt.fields.ctx,
				logger:           tt.fields.logger,
				client:           tt.fields.client,
				scheme:           tt.fields.scheme,
				ec:               tt.fields.ec,
				desiredVersion:   tt.fields.desiredVersion,
				publishedVersion: tt.fields.publishedVersion,
				cacheState:       tt.fields.cacheState,
				revisionList:     tt.fields.revisionList,
			}
			if got := r.isRevisionRetentionReconciled(tt.args.retention); !reflect.DeepEqual(got, tt.wantTrimmed) {
				t.Errorf("RevisionReconciler.isRevisionRetentionReconciled() = %v, want %v", got, tt.wantTrimmed)
			}
			if !reflect.DeepEqual(r.GetRevisionList(), tt.wantList) {
				t.Errorf("RevisionReconciler.isRevisionRetentionReconciled() = %v, want %v", tt.fields.revisionList, tt.wantList)
			}
		})
	}
}

func Test_popRevision(t *testing.T) {
	type args struct {
		list *[]marin3rv1alpha1.EnvoyConfigRevision
	}
	tests := []struct {
		name     string
		list     []marin3rv1alpha1.EnvoyConfigRevision
		wantItem marin3rv1alpha1.EnvoyConfigRevision
		wantList []marin3rv1alpha1.EnvoyConfigRevision
	}{
		{
			name: "Pops an element from the list",
			list: []marin3rv1alpha1.EnvoyConfigRevision{
				{ObjectMeta: metav1.ObjectMeta{Name: "ecr1"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "ecr2"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "ecr3"}},
			},
			wantItem: marin3rv1alpha1.EnvoyConfigRevision{ObjectMeta: metav1.ObjectMeta{Name: "ecr1"}},
			wantList: []marin3rv1alpha1.EnvoyConfigRevision{
				{ObjectMeta: metav1.ObjectMeta{Name: "ecr2"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "ecr3"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := popRevision(&tt.list); !reflect.DeepEqual(got, tt.wantItem) {
				t.Errorf("popRevision() = %v, want %v", got, tt.wantItem)
			}
			if !reflect.DeepEqual(tt.list, tt.wantList) {
				t.Errorf("popRevision() = %v, want %v", tt.list, tt.wantList)
			}
		})
	}
}
