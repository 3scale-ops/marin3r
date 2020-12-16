package reconcilers

import (
	"reflect"
	"testing"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsStatusReconciled(t *testing.T) {
	type args struct {
		ec               *marin3rv1alpha1.EnvoyConfig
		cacheState       string
		publishedVersion string
		list             *marin3rv1alpha1.EnvoyConfigRevisionList
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Status already up to date, returns true",
			args: args{
				ec: &marin3rv1alpha1.EnvoyConfig{
					Status: marin3rv1alpha1.EnvoyConfigStatus{
						DesiredVersion:   "6ddbcdf795",
						PublishedVersion: "6ddbcdf795",
						CacheState:       marin3rv1alpha1.InSyncState,
						ConfigRevisions: []marin3rv1alpha1.ConfigRevisionRef{
							{Version: "1", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "test"}},
							{Version: "2", Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "test"}},
						},
						Conditions: status.Conditions{
							{Type: marin3rv1alpha1.CacheOutOfSyncCondition, Status: corev1.ConditionFalse},
							{Type: marin3rv1alpha1.RollbackFailedCondition, Status: corev1.ConditionFalse},
						},
					},
				},
				cacheState:       marin3rv1alpha1.InSyncState,
				publishedVersion: "6ddbcdf795",
				list: &marin3rv1alpha1.EnvoyConfigRevisionList{
					Items: []marin3rv1alpha1.EnvoyConfigRevision{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "ecr1", Namespace: "test"},
							Spec:       marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "1"},
						},
						{
							ObjectMeta: metav1.ObjectMeta{Name: "ecr2", Namespace: "test"},
							Spec:       marin3rv1alpha1.EnvoyConfigRevisionSpec{Version: "2"},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "RollbackFailedCondition needs to be inactive, returns false",
			args: args{
				ec: &marin3rv1alpha1.EnvoyConfig{
					Status: marin3rv1alpha1.EnvoyConfigStatus{
						DesiredVersion:   "6ddbcdf795",
						PublishedVersion: "6ddbcdf795",
						CacheState:       marin3rv1alpha1.InSyncState,
						ConfigRevisions:  []marin3rv1alpha1.ConfigRevisionRef{},
						Conditions: status.Conditions{
							{Type: marin3rv1alpha1.CacheOutOfSyncCondition, Status: corev1.ConditionFalse},
							{Type: marin3rv1alpha1.RollbackFailedCondition, Status: corev1.ConditionTrue},
						},
					},
				},
				cacheState:       marin3rv1alpha1.InSyncState,
				publishedVersion: "6ddbcdf795",
				list:             &marin3rv1alpha1.EnvoyConfigRevisionList{},
			},
			want: false,
		},
		{
			name: "CacheOutOfSyncCondition needs to be inactive, returns false",
			args: args{
				ec: &marin3rv1alpha1.EnvoyConfig{
					Status: marin3rv1alpha1.EnvoyConfigStatus{
						DesiredVersion:   "6ddbcdf795",
						PublishedVersion: "6ddbcdf795",
						CacheState:       marin3rv1alpha1.InSyncState,
						ConfigRevisions:  []marin3rv1alpha1.ConfigRevisionRef{},
						Conditions: status.Conditions{
							{Type: marin3rv1alpha1.CacheOutOfSyncCondition, Status: corev1.ConditionTrue},
							{Type: marin3rv1alpha1.RollbackFailedCondition, Status: corev1.ConditionFalse},
						},
					},
				},
				cacheState:       marin3rv1alpha1.InSyncState,
				publishedVersion: "6ddbcdf795",
				list:             &marin3rv1alpha1.EnvoyConfigRevisionList{},
			},
			want: false,
		},
		{
			name: "CacheOutOfSyncCondition and RollbackFailedCondition need to be active, returns false",
			args: args{
				ec: &marin3rv1alpha1.EnvoyConfig{
					Status: marin3rv1alpha1.EnvoyConfigStatus{
						DesiredVersion:   "6ddbcdf795",
						PublishedVersion: "6ddbcdf795",
						CacheState:       marin3rv1alpha1.RollbackFailedState,
						ConfigRevisions:  []marin3rv1alpha1.ConfigRevisionRef{},
						Conditions: status.Conditions{
							{Type: marin3rv1alpha1.CacheOutOfSyncCondition, Status: corev1.ConditionFalse},
							{Type: marin3rv1alpha1.RollbackFailedCondition, Status: corev1.ConditionFalse},
						},
					},
				},
				cacheState:       marin3rv1alpha1.InSyncState,
				publishedVersion: "6ddbcdf795",
				list:             &marin3rv1alpha1.EnvoyConfigRevisionList{},
			},
			want: false,
		},
		{
			name: "DesiredVersion needs update, return false",
			args: args{
				ec: &marin3rv1alpha1.EnvoyConfig{
					Status: marin3rv1alpha1.EnvoyConfigStatus{
						DesiredVersion:   "xxxx",
						PublishedVersion: "6ddbcdf795",
						CacheState:       marin3rv1alpha1.InSyncState,
						ConfigRevisions:  []marin3rv1alpha1.ConfigRevisionRef{},
						Conditions: status.Conditions{
							{Type: marin3rv1alpha1.CacheOutOfSyncCondition, Status: corev1.ConditionFalse},
							{Type: marin3rv1alpha1.RollbackFailedCondition, Status: corev1.ConditionFalse},
						},
					},
				},
				cacheState:       marin3rv1alpha1.InSyncState,
				publishedVersion: "6ddbcdf795",
				list:             &marin3rv1alpha1.EnvoyConfigRevisionList{},
			},
			want: false,
		},
		{
			name: "PublishedVersion needs update, return false",
			args: args{
				ec: &marin3rv1alpha1.EnvoyConfig{
					Status: marin3rv1alpha1.EnvoyConfigStatus{
						DesiredVersion:   "6ddbcdf795",
						PublishedVersion: "xxxx",
						CacheState:       marin3rv1alpha1.InSyncState,
						ConfigRevisions:  []marin3rv1alpha1.ConfigRevisionRef{},
						Conditions: status.Conditions{
							{Type: marin3rv1alpha1.CacheOutOfSyncCondition, Status: corev1.ConditionFalse},
							{Type: marin3rv1alpha1.RollbackFailedCondition, Status: corev1.ConditionFalse},
						},
					},
				},
				cacheState:       marin3rv1alpha1.InSyncState,
				publishedVersion: "6ddbcdf795",
				list:             &marin3rv1alpha1.EnvoyConfigRevisionList{},
			},
			want: false,
		},
		{
			name: "CacheState needs update, return false",
			args: args{
				ec: &marin3rv1alpha1.EnvoyConfig{
					Status: marin3rv1alpha1.EnvoyConfigStatus{
						DesiredVersion:   "6ddbcdf795",
						PublishedVersion: "xxxx",
						CacheState:       marin3rv1alpha1.InSyncState,
						ConfigRevisions:  []marin3rv1alpha1.ConfigRevisionRef{},
						Conditions: status.Conditions{
							{Type: marin3rv1alpha1.CacheOutOfSyncCondition, Status: corev1.ConditionFalse},
							{Type: marin3rv1alpha1.RollbackFailedCondition, Status: corev1.ConditionFalse},
						},
					},
				},
				cacheState:       marin3rv1alpha1.RollbackState,
				publishedVersion: "xxxx",
				list:             &marin3rv1alpha1.EnvoyConfigRevisionList{},
			},
			want: false,
		},
		{
			name: "Status empty, return false",
			args: args{
				ec: &marin3rv1alpha1.EnvoyConfig{
					Status: marin3rv1alpha1.EnvoyConfigStatus{},
				},
				cacheState:       marin3rv1alpha1.RollbackState,
				publishedVersion: "xxxx",
				list:             &marin3rv1alpha1.EnvoyConfigRevisionList{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStatusReconciled(tt.args.ec, tt.args.cacheState, tt.args.publishedVersion, tt.args.list); got != tt.want {
				t.Errorf("IsStatusReconciled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateRevisionList(t *testing.T) {
	type args struct {
		list *marin3rv1alpha1.EnvoyConfigRevisionList
	}
	tests := []struct {
		name string
		args args
		want []marin3rv1alpha1.ConfigRevisionRef
	}{
		{
			name: "Generates a list of revision references",
			args: args{list: &marin3rv1alpha1.EnvoyConfigRevisionList{
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "test",
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr2",
							Namespace: "test",
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							Version: "2",
						},
					},
				},
			}},
			want: []marin3rv1alpha1.ConfigRevisionRef{
				{
					Version: "1",
					Ref: corev1.ObjectReference{
						Name:      "ecr1",
						Namespace: "test",
					},
				},
				{
					Version: "2",
					Ref: corev1.ObjectReference{
						Name:      "ecr2",
						Namespace: "test",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateRevisionList(tt.args.list); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateRevisionList() = %v, want %v", got, tt.want)
			}
		})
	}
}
