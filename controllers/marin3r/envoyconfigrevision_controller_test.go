package controllers

import (
	"context"
	"testing"

	"github.com/3scale-ops/basereconciler/reconciler"
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	xdss_v3 "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/v3"
	"github.com/3scale-ops/marin3r/pkg/envoy"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnvoyConfigRevisionReconciler_taintSelf(t *testing.T) {

	err := marin3rv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("Taints the ecr object", func(t *testing.T) {
		ecr := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "default"},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				NodeID:    "node1",
				Version:   "bbbb",
				Resources: []marin3rv1alpha1.Resource{},
			},
		}
		r := &EnvoyConfigRevisionReconciler{
			Reconciler: &reconciler.Reconciler{
				Client: fake.NewClientBuilder().WithObjects(ecr).Build(),
				Scheme: scheme.Scheme,
				Log:    ctrl.Log.WithName("test"),
			},
			XdsCache: xdss_v3.NewCache(),
		}
		if err := r.taintSelf(context.TODO(), ecr, "test", "test", r.Log); err != nil {
			t.Errorf("EnvoyConfigRevisionReconciler.taintSelf() error = %v", err)
		}
		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ecr", Namespace: "default"}, ecr)
		if !meta.IsStatusConditionTrue(ecr.Status.Conditions, marin3rv1alpha1.RevisionTaintedCondition) {
			t.Errorf("EnvoyConfigRevisionReconciler.taintSelf() ecr is not tainted")
		}
	})
}

func Test_filterByAPIVersion(t *testing.T) {
	type args struct {
		obj     runtime.Object
		version envoy.APIVersion
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "V3 EnvoyConfigRevision with V3 controller returns true",
			args: args{
				obj: &marin3rv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "xx", Namespace: "xx"},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						EnvoyAPI: pointer.New(envoy.APIv3),
					},
				},
				version: envoy.APIv3,
			},
			want: true,
		},
		{
			name: "XX EnvoyConfigRevision with V3 controller returns false",
			args: args{
				obj: &marin3rv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "xx", Namespace: "xx"},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						EnvoyAPI: pointer.New(envoy.APIVersion("XX")),
					},
				},
				version: envoy.APIv3,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterByAPIVersion(tt.args.obj, tt.args.version); got != tt.want {
				t.Errorf("filterByAPIVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
