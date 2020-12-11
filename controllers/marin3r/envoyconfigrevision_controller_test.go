package controllers

import (
	"context"
	"testing"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	xdss_v2 "github.com/3scale/marin3r/pkg/discoveryservice/xdss/v2"
	"github.com/3scale/marin3r/pkg/envoy"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnvoyConfigRevisionReconciler_taintSelf(t *testing.T) {

	t.Run("Taints the ecr object", func(t *testing.T) {
		ecr := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: "default"},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				NodeID:         "node1",
				Version:        "bbbb",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
		}
		r := &EnvoyConfigRevisionReconciler{
			Client:   fake.NewFakeClient(ecr),
			Scheme:   s,
			XdsCache: xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)),
			Log:      ctrl.Log.WithName("test"),
		}
		if err := r.taintSelf(context.TODO(), ecr, "test", "test", r.Log); err != nil {
			t.Errorf("EnvoyConfigRevisionReconciler.taintSelf() error = %v", err)
		}
		r.Client.Get(context.TODO(), types.NamespacedName{Name: "ecr", Namespace: "default"}, ecr)
		if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionTaintedCondition) {
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
			name: "V2 EnvoyConfigRevision with V2 controller returns true",
			args: args{
				obj: &marin3rv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "xx", Namespace: "xx"},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						EnvoyAPI: pointer.StringPtr(string(envoy.APIv2)),
					},
				},
				version: envoy.APIv2,
			},
			want: true,
		},
		{
			name: "V3 EnvoyConfigRevision with V3 controller returns true",
			args: args{
				obj: &marin3rv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "xx", Namespace: "xx"},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						EnvoyAPI: pointer.StringPtr(string(envoy.APIv3)),
					},
				},
				version: envoy.APIv3,
			},
			want: true,
		},
		{
			name: "V2 EnvoyConfigRevision with V3 controller returns false",
			args: args{
				obj: &marin3rv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "xx", Namespace: "xx"},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						EnvoyAPI: pointer.StringPtr(string(envoy.APIv2)),
					},
				},
				version: envoy.APIv3,
			},
			want: false,
		},
		{
			name: "V3 EnvoyConfigRevision with V2 controller returns false",
			args: args{
				obj: &marin3rv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{Name: "xx", Namespace: "xx"},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						EnvoyAPI: pointer.StringPtr(string(envoy.APIv2)),
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
