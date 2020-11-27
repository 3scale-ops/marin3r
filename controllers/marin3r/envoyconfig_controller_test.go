package controllers

import (
	"context"
	"testing"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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
		ec      *marin3rv1alpha1.EnvoyConfig
		ecrList *marin3rv1alpha1.EnvoyConfigRevisionList
		want    string
		wantErr bool
	}{
		{
			name: "Returns the desiredVersion on seeing a new version",
			ec: &marin3rv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: marin3rv1alpha1.EnvoyConfigSpec{
					NodeID:         "node1",
					EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
				},
				Status: marin3rv1alpha1.EnvoyConfigStatus{
					ConfigRevisions: []marin3rv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
					},
				},
			},
			ecrList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "xxx",
							EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
						},
					},
				},
			},
			want:    "xxx",
			wantErr: false,
		},
		{
			name: "Returns the highest index untainted revision of the ConfigRevision list",
			ec: &marin3rv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: marin3rv1alpha1.EnvoyConfigSpec{
					NodeID:         "node1",
					EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
				},
				Status: marin3rv1alpha1.EnvoyConfigStatus{
					ConfigRevisions: []marin3rv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
						{Version: "zzz", Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
					},
				},
			},
			ecrList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "xxx",
							EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr2",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "zzz",
							EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
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
			ec: &marin3rv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: marin3rv1alpha1.EnvoyConfigSpec{
					NodeID:         "node1",
					EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
				},
				Status: marin3rv1alpha1.EnvoyConfigStatus{
					ConfigRevisions: []marin3rv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
						{Version: "zzz", Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
					},
				},
			},
			ecrList: &marin3rv1alpha1.EnvoyConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []marin3rv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "xxx",
							EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
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
						Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "zzz",
							EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
						},
						Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   marin3rv1alpha1.RevisionTaintedCondition,
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
				Client: fake.NewFakeClient(tt.ec, tt.ecrList),
				Scheme: s,
				Log:    ctrl.Log.WithName("test"),
			}
			got, err := r.getVersionToPublish(context.TODO(), tt.ec, nil)
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
