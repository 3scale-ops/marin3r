package reconcilers

import (
	"context"
	"testing"
	"time"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	"github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/stats"
	xdss_v3 "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/v3"
	"github.com/3scale-ops/marin3r/pkg/envoy"
	"github.com/davecgh/go-spew/spew"
	resource_v3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestIsStatusReconciled(t *testing.T) {
	type args struct {
		envoyConfigRevisionFactory func() *marin3rv1alpha1.EnvoyConfigRevision
		versionTrackerFactory      func() *marin3rv1alpha1.VersionTracker
		xdssCacheFactory           func() xdss.Cache
		dStats                     func() *stats.Stats
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
							Conditions: []metav1.Condition{
								{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: metav1.ConditionTrue},
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
				xdssCacheFactory: func() xdss.Cache {
					cache := xdss_v3.NewCache()
					cache.SetSnapshot(context.TODO(), "test", cache.NewSnapshot())
					return cache
				},
				dStats: stats.New,
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
							Published: pointer.Bool(true),
							ProvidesVersions: &marin3rv1alpha1.VersionTracker{
								Endpoints: "a",
								Clusters:  "b",
								Routes:    "c",
								Listeners: "d",
								Secrets:   "e",
								Runtimes:  "f",
							},
							LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.Now()),
							Conditions: []metav1.Condition{
								{
									Type:   marin3rv1alpha1.RevisionPublishedCondition,
									Status: metav1.ConditionTrue,
								},
								{
									Type:    marin3rv1alpha1.ResourcesInSyncCondition,
									Status:  metav1.ConditionTrue,
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
				xdssCacheFactory: func() xdss.Cache {
					cache := xdss_v3.NewCache()
					cache.SetSnapshot(context.TODO(), "test", cache.NewSnapshot())
					return cache
				},
				dStats: stats.New,
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
							Published:       pointer.Bool(true),
							LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.Now()),
							Conditions: []metav1.Condition{
								{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: metav1.ConditionFalse},
								{Type: marin3rv1alpha1.ResourcesInSyncCondition, Status: metav1.ConditionTrue},
							},
						},
					}
				},
				versionTrackerFactory: func() *marin3rv1alpha1.VersionTracker { return nil },
				xdssCacheFactory: func() xdss.Cache {
					cache := xdss_v3.NewCache()
					cache.SetSnapshot(context.TODO(), "test", cache.NewSnapshot())
					return cache
				},
				dStats: stats.New,
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
							Published:       pointer.Bool(false),
							LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.Now()),
							Conditions: []metav1.Condition{
								{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: metav1.ConditionFalse},
							},
						},
					}
				},
				versionTrackerFactory: func() *marin3rv1alpha1.VersionTracker { return nil },
				xdssCacheFactory: func() xdss.Cache {
					cache := xdss_v3.NewCache()
					cache.SetSnapshot(context.TODO(), "test", cache.NewSnapshot())
					return cache
				},
				dStats: stats.New,
			},
			want: true,
		},
		{
			name: "Reported failed, needs tainted condition",
			args: args{
				envoyConfigRevisionFactory: func() *marin3rv1alpha1.EnvoyConfigRevision {
					return &marin3rv1alpha1.EnvoyConfigRevision{
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version:  "xxxx",
							NodeID:   "test",
							EnvoyAPI: pointer.String(envoy.APIv3.String()),
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							ProvidesVersions: &marin3rv1alpha1.VersionTracker{Endpoints: "aaaa"},
						},
					}
				},
				xdssCacheFactory: func() xdss.Cache {
					cache := xdss_v3.NewCache()
					cache.SetSnapshot(context.TODO(), "test", cache.NewSnapshot())
					return cache
				},
				versionTrackerFactory: func() *marin3rv1alpha1.VersionTracker { return &marin3rv1alpha1.VersionTracker{Endpoints: "aaaa"} },
				dStats: func() *stats.Stats {
					return stats.NewWithItems(map[string]cache.Item{
						"test:" + resource_v3.EndpointType + ":*:pod-aaaa:request_counter:stream_1": {Object: int64(2), Expiration: int64(0)},
						"test:" + resource_v3.EndpointType + ":aaaa:pod-aaaa:nack_counter":          {Object: int64(1), Expiration: int64(0)},
					}, time.Now())
				},
			},
			want: false,
		},
		{
			name: "Reported failed, status already up to date",
			args: args{
				envoyConfigRevisionFactory: func() *marin3rv1alpha1.EnvoyConfigRevision {
					return &marin3rv1alpha1.EnvoyConfigRevision{
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version:  "xxxx",
							NodeID:   "test",
							EnvoyAPI: pointer.String(envoy.APIv3.String()),
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							ProvidesVersions: &marin3rv1alpha1.VersionTracker{Endpoints: "aaaa"},
							Tainted:          pointer.Bool(true),
							Conditions: []metav1.Condition{
								{
									Type:    marin3rv1alpha1.RevisionTaintedCondition,
									Status:  metav1.ConditionTrue,
									Reason:  "ResourcesFailing",
									Message: "EnvoyConfigRevision resources are being rejected by more than 100% of the Envoy clients",
								},
							},
						},
					}
				},
				xdssCacheFactory: func() xdss.Cache {
					cache := xdss_v3.NewCache()
					cache.SetSnapshot(context.TODO(), "test", cache.NewSnapshot())
					return cache
				},
				versionTrackerFactory: func() *marin3rv1alpha1.VersionTracker { return &marin3rv1alpha1.VersionTracker{Endpoints: "aaaa"} },
				dStats: func() *stats.Stats {
					return stats.NewWithItems(map[string]cache.Item{
						"test:" + resource_v3.EndpointType + ":*:pod-aaaa:request_counter:stream_1": {Object: int64(2), Expiration: int64(0)},
						"test:" + resource_v3.EndpointType + ":aaaa:pod-aaaa:nack_counter":          {Object: int64(1), Expiration: int64(0)},
					}, time.Now())
				},
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
							Conditions: []metav1.Condition{
								{Type: marin3rv1alpha1.RevisionTaintedCondition, Status: metav1.ConditionTrue},
							},
						},
					}
				},
				versionTrackerFactory: func() *marin3rv1alpha1.VersionTracker { return nil },
				xdssCacheFactory: func() xdss.Cache {
					cache := xdss_v3.NewCache()
					cache.SetSnapshot(context.TODO(), "test", cache.NewSnapshot())
					return cache
				},
				dStats: stats.New,
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
							Tainted: pointer.Bool(true),
							Conditions: []metav1.Condition{
								{Type: marin3rv1alpha1.RevisionTaintedCondition, Status: metav1.ConditionTrue},
							},
						},
					}
				},
				versionTrackerFactory: func() *marin3rv1alpha1.VersionTracker { return nil },
				xdssCacheFactory: func() xdss.Cache {
					cache := xdss_v3.NewCache()
					cache.SetSnapshot(context.TODO(), "test", cache.NewSnapshot())
					return cache
				},
				dStats: stats.New,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ecr := tt.args.envoyConfigRevisionFactory()
			if got := IsStatusReconciled(ecr, tt.args.versionTrackerFactory(), tt.args.xdssCacheFactory(), tt.args.dStats()); got != tt.want {
				spew.Dump(ecr.Status)
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
		want metav1.ConditionStatus
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
							Conditions: []metav1.Condition{
								{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: metav1.ConditionTrue},
							},
						},
					}
				},
				xdssCacheFactory: func() xdss.Cache {
					cache := xdss_v3.NewCache()
					cache.SetSnapshot(context.TODO(), "test", cache.NewSnapshot())
					return cache
				},
			},
			want: metav1.ConditionTrue,
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
							Conditions: []metav1.Condition{
								{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: metav1.ConditionTrue},
							},
						},
					}
				},
				xdssCacheFactory: func() xdss.Cache {
					cache := xdss_v3.NewCache()
					return cache
				},
			},
			want: metav1.ConditionFalse,
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

func Test_calculateRevisionTaintedCondition(t *testing.T) {
	type args struct {
		ecr        *marin3rv1alpha1.EnvoyConfigRevision
		vt         *marin3rv1alpha1.VersionTracker
		dStats     *stats.Stats
		thresshold float64
	}
	tests := []struct {
		name string
		args args
		want corev1.ConditionStatus
	}{
		{
			name: "All endpoints fail, return taint",
			args: args{
				ecr: &marin3rv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "test"},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:   "node",
						EnvoyAPI: pointer.String(envoy.APIv3.String()),
					},
				},
				vt: &marin3rv1alpha1.VersionTracker{
					Endpoints: "xxxx",
					Clusters:  "",
					Routes:    "",
					Listeners: "",
					Secrets:   "",
					Runtimes:  "",
				},
				dStats: stats.NewWithItems(map[string]cache.Item{
					"node:" + resource_v3.EndpointType + ":*:pod-bbbb:request_counter:stream_2": {Object: int64(5), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":*:pod-cccc:request_counter:stream_3": {Object: int64(1), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":*:pod-dddd:request_counter:stream_4": {Object: int64(1), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":*:pod-aaaa:request_counter:stream_1": {Object: int64(2), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":xxxx:pod-aaaa:nack_counter":          {Object: int64(1), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":xxxx:pod-bbbb:nack_counter":          {Object: int64(10), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":xxxx:pod-cccc:nack_counter":          {Object: int64(10), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":xxxx:pod-dddd:nack_counter":          {Object: int64(10), Expiration: int64(0)},
				}, time.Now()),
				thresshold: 1,
			},
			want: corev1.ConditionTrue,
		},
		{
			name: "Half of endpoints fail, return taint",
			args: args{
				ecr: &marin3rv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "test"},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:   "node",
						EnvoyAPI: pointer.String(envoy.APIv3.String()),
					},
				},
				vt: &marin3rv1alpha1.VersionTracker{
					Endpoints: "xxxx",
					Clusters:  "",
					Routes:    "",
					Listeners: "",
					Secrets:   "",
					Runtimes:  "",
				},
				dStats: stats.NewWithItems(map[string]cache.Item{
					"node:" + resource_v3.EndpointType + ":*:pod-bbbb:request_counter:stream_2": {Object: int64(5), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":*:pod-cccc:request_counter:stream_3": {Object: int64(1), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":*:pod-dddd:request_counter:stream_4": {Object: int64(1), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":*:pod-aaaa:request_counter:stream_1": {Object: int64(2), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":xxxx:pod-aaaa:nack_counter":          {Object: int64(1), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":xxxx:pod-bbbb:nack_counter":          {Object: int64(10), Expiration: int64(0)},
				}, time.Now()),
				thresshold: 0.5,
			},
			want: corev1.ConditionTrue,
		},
		{
			name: "Less than half of endpoints fail, return nil",
			args: args{
				ecr: &marin3rv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "test"},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:   "node",
						EnvoyAPI: pointer.String(envoy.APIv3.String()),
					},
				},
				vt: &marin3rv1alpha1.VersionTracker{
					Endpoints: "xxxx",
					Clusters:  "",
					Routes:    "",
					Listeners: "",
					Secrets:   "",
					Runtimes:  "",
				},
				dStats: stats.NewWithItems(map[string]cache.Item{
					"node:" + resource_v3.EndpointType + ":*:pod-bbbb:request_counter:stream_2": {Object: int64(5), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":*:pod-cccc:request_counter:stream_3": {Object: int64(1), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":*:pod-dddd:request_counter:stream_4": {Object: int64(1), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":*:pod-aaaa:request_counter:stream_1": {Object: int64(2), Expiration: int64(0)},
					"node:" + resource_v3.EndpointType + ":xxxx:pod-aaaa:nack_counter":          {Object: int64(1), Expiration: int64(0)},
				}, time.Now()),
				thresshold: 0.5,
			},
			want: corev1.ConditionFalse,
		},
		{
			name: "No data, return nil",
			args: args{
				ecr: &marin3rv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "test"},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						NodeID:   "node",
						EnvoyAPI: pointer.String(envoy.APIv3.String()),
					},
				}, vt: &marin3rv1alpha1.VersionTracker{},
				dStats:     stats.NewWithItems(map[string]cache.Item{}, time.Now()),
				thresshold: 1,
			},
			want: corev1.ConditionFalse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateRevisionTaintedCondition(tt.args.ecr, tt.args.vt, tt.args.dStats, tt.args.thresshold)
			if tt.want == corev1.ConditionFalse && got != nil {
				t.Errorf("calculateRevisionTaintedCondition() = %v, want %v", got, tt.want)
				return
			}
			if tt.want == corev1.ConditionTrue && got == nil {
				t.Errorf("calculateRevisionTaintedCondition() = %v, want %v", got, tt.want)
				return
			}
		})
	}
}
