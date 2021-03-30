package revisions

import (
	"reflect"
	"testing"
	"time"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSortByPublication(t *testing.T) {
	now := time.Now()

	type args struct {
		currentResourcesVersion string
		list                    *marin3rv1alpha1.EnvoyConfigRevisionList
	}
	tests := []struct {
		name string
		args args
		want *marin3rv1alpha1.EnvoyConfigRevisionList
	}{
		{
			name: "Returns a sorted list of EnvoyConfigRevisions, using publication order",
			args: args{
				currentResourcesVersion: "2",
				list: &marin3rv1alpha1.EnvoyConfigRevisionList{
					Items: []marin3rv1alpha1.EnvoyConfigRevision{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "ecr1", Namespace: "default"},
							Spec:       marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "1"},
							Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
								LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.NewTime(now.Add(-4 * time.Second))),
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{Name: "ecr2", Namespace: "default"},
							Spec:       marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "2"},
							Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
								LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.NewTime(now.Add(-2 * time.Second))),
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{Name: "ecr3", Namespace: "default"},
							Spec:       marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "3"},
							Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
								LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.NewTime(now.Add(-1 * time.Second))),
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{Name: "ecr4", Namespace: "default",
								CreationTimestamp: metav1.NewTime(now.Add(-3 * time.Second)),
							},
							Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "4"}},
					},
				},
			},
			want: &marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "ecr1", Namespace: "default"},
						Spec:       marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "1"},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.NewTime(now.Add(-4 * time.Second))),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "ecr4", Namespace: "default",
							CreationTimestamp: metav1.NewTime(now.Add(-3 * time.Second)),
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "4"},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "ecr3", Namespace: "default"},
						Spec:       marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "3"},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.NewTime(now.Add(-1 * time.Second))),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "ecr2", Namespace: "default"},
						Spec:       marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "2"},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							LastPublishedAt: func(t metav1.Time) *metav1.Time { return &t }(metav1.NewTime(now.Add(-2 * time.Second))),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SortByPublication(tt.args.currentResourcesVersion, tt.args.list); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SortByPublication() = %v, want %v", got, tt.want)
			}
		})
	}
}
