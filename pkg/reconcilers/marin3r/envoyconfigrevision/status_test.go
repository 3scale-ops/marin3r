package reconcilers

import (
	"testing"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	xdss_v3 "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/v3"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func testCacheGenerator(nodeID, version string) func() xdss.Cache {
	return func() xdss.Cache {
		cache := xdss_v3.NewCache(cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil))
		snap := cache_v3.NewSnapshot(version,
			[]cache_types.Resource{},
			[]cache_types.Resource{},
			[]cache_types.Resource{},
			[]cache_types.Resource{},
			[]cache_types.Resource{},
			[]cache_types.Resource{},
		)
		cache.SetSnapshot(nodeID, xdss_v3.NewSnapshot(&snap))
		return cache
	}
}

func TestIsStatusReconciled(t *testing.T) {
	type args struct {
		envoyConfigRevisionFactory func() *marin3rv1alpha1.EnvoyConfigRevision
		versionTrackerFactory      func() *marin3rv1alpha1.VersionTracker
		xdssCacheFactory           func() xdss.Cache
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Revision pusblished, status needs update",
			args: args{
				envoyConfigRevisionFactory: func() *marin3rv1alpha1.EnvoyConfigRevision {
					return &marin3rv1alpha1.EnvoyConfigRevision{
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "xxxx",
							NodeID:  "test",
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.Conditions{
								{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionTrue},
							},
						},
					}
				},
				versionTrackerFactory: func() *marin3rv1alpha1.VersionTracker {
					return &marin3rv1alpha1.VersionTracker{
						Endpoints: "a",
						Clusters:  "b",
						Routes:    "c",
						Listeners: "d",
						Secrets:   "e",
						Runtimes:  "f",
					}
				},
				xdssCacheFactory: testCacheGenerator("test", "xxxx"),
			},
			want: false,
		},
		{
			name: "Revision pusblished, status already up to date",
			args: args{
				envoyConfigRevisionFactory: func() *marin3rv1alpha1.EnvoyConfigRevision {
					return &marin3rv1alpha1.EnvoyConfigRevision{
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "xxxx",
							NodeID:  "test",
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Published: pointer.BoolPtr(true),
							ProvidesVersions: &marin3rv1alpha1.VersionTracker{
								Endpoints: "a",
								Clusters:  "b",
								Routes:    "c",
								Listeners: "d",
								Secrets:   "e",
								Runtimes:  "f",
							},
							LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.Now()),
							Conditions: status.Conditions{
								{
									Type:   marin3rv1alpha1.RevisionPublishedCondition,
									Status: corev1.ConditionTrue,
								},
								{
									Type:    marin3rv1alpha1.ResourcesInSyncCondition,
									Status:  corev1.ConditionTrue,
									Reason:  "ResourcesSynced",
									Message: "EnvoyConfigRevision resources successfully synced with xDS server cache",
								},
							},
						},
					}
				},
				versionTrackerFactory: func() *marin3rv1alpha1.VersionTracker {
					return &marin3rv1alpha1.VersionTracker{
						Endpoints: "a",
						Clusters:  "b",
						Routes:    "c",
						Listeners: "d",
						Secrets:   "e",
						Runtimes:  "f",
					}
				},
				xdssCacheFactory: testCacheGenerator("test", "xxxx"),
			},
			want: true,
		},
		{
			name: "Revision unpublished, status needs update",
			args: args{
				envoyConfigRevisionFactory: func() *marin3rv1alpha1.EnvoyConfigRevision {
					return &marin3rv1alpha1.EnvoyConfigRevision{
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "xxxx",
							NodeID:  "test",
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Published:       pointer.BoolPtr(true),
							LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.Now()),
							Conditions: status.Conditions{
								{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionFalse},
								{Type: marin3rv1alpha1.ResourcesInSyncCondition, Status: corev1.ConditionTrue},
							},
						},
					}
				},
				versionTrackerFactory: func() *marin3rv1alpha1.VersionTracker { return nil },
				xdssCacheFactory:      testCacheGenerator("test", "xxxx"),
			},
			want: false,
		},
		{
			name: "Revision unpublished, status already up to date",
			args: args{
				envoyConfigRevisionFactory: func() *marin3rv1alpha1.EnvoyConfigRevision {
					return &marin3rv1alpha1.EnvoyConfigRevision{
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "xxxx",
							NodeID:  "test",
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Published:       pointer.BoolPtr(false),
							LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.Now()),
							Conditions: status.Conditions{
								{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionFalse},
							},
						},
					}
				},
				versionTrackerFactory: func() *marin3rv1alpha1.VersionTracker { return nil },
				xdssCacheFactory:      testCacheGenerator("test", "xxxx"),
			},
			want: true,
		},
		{
			name: "Revision tainted, status needs update",
			args: args{
				envoyConfigRevisionFactory: func() *marin3rv1alpha1.EnvoyConfigRevision {
					return &marin3rv1alpha1.EnvoyConfigRevision{
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "xxxx",
							NodeID:  "test",
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.Conditions{
								{Type: marin3rv1alpha1.RevisionTaintedCondition, Status: corev1.ConditionTrue},
							},
						},
					}
				},
				versionTrackerFactory: func() *marin3rv1alpha1.VersionTracker { return nil },
				xdssCacheFactory:      testCacheGenerator("test", "xxxx"),
			},
			want: false,
		},
		{
			name: "Revision tainted, status already up to date",
			args: args{
				envoyConfigRevisionFactory: func() *marin3rv1alpha1.EnvoyConfigRevision {
					return &marin3rv1alpha1.EnvoyConfigRevision{
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "xxxx",
							NodeID:  "test",
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Tainted: pointer.BoolPtr(true),
							Conditions: status.Conditions{
								{Type: marin3rv1alpha1.RevisionTaintedCondition, Status: corev1.ConditionTrue},
							},
						},
					}
				},
				versionTrackerFactory: func() *marin3rv1alpha1.VersionTracker { return nil },
				xdssCacheFactory:      testCacheGenerator("test", "xxxx"),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ecr := tt.args.envoyConfigRevisionFactory()
			if got := IsStatusReconciled(ecr, tt.args.versionTrackerFactory(), tt.args.xdssCacheFactory()); got != tt.want {
				t.Errorf("IsStatusReconciled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_calculateResourcesInSyncCondition(t *testing.T) {
	type args struct {
		envoyConfigRevisionFactory func() *marin3rv1alpha1.EnvoyConfigRevision
		xdssCacheFactory           func() xdss.Cache
	}
	tests := []struct {
		name string
		args args
		want corev1.ConditionStatus
	}{
		{
			name: "Returns condition true",
			args: args{
				envoyConfigRevisionFactory: func() *marin3rv1alpha1.EnvoyConfigRevision {
					return &marin3rv1alpha1.EnvoyConfigRevision{
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "xxxx",
							NodeID:  "test",
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.Conditions{
								{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionTrue},
							},
						},
					}
				},
				xdssCacheFactory: testCacheGenerator("test", "xxxx"),
			},
			want: corev1.ConditionTrue,
		},
		{
			name: "Returns condition false if snapshot not found for spec.nodeID",
			args: args{
				envoyConfigRevisionFactory: func() *marin3rv1alpha1.EnvoyConfigRevision {
					return &marin3rv1alpha1.EnvoyConfigRevision{
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "xxxx",
							NodeID:  "test",
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.Conditions{
								{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionTrue},
							},
						},
					}
				},
				xdssCacheFactory: func() xdss.Cache {
					cache := xdss_v3.NewCache(cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil))
					return cache
				},
			},
			want: corev1.ConditionFalse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateResourcesInSyncCondition(tt.args.envoyConfigRevisionFactory(), tt.args.xdssCacheFactory()); got.Status != tt.want {
				t.Errorf("calculateResourcesInSyncCondition() = %v, want %v", got.Status, tt.want)
			}
		})
	}
}
