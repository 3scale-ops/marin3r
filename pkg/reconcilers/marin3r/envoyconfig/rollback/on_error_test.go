package rollback

import (
	"testing"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	"github.com/3scale-ops/marin3r/pkg/reconcilers/marin3r/envoyconfig/filters"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
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

func TestOnError(t *testing.T) {
	type args struct {
		nodeID   string
		version  string
		msg      string
		envoyAPI envoy.APIVersion
	}
	tests := []struct {
		name    string
		cl      client.Client
		args    args
		wantErr bool
	}{
		{
			name: "Returns a function that does not return error when called",
			cl: fake.NewFakeClientWithScheme(s,
				&marin3rv1alpha1.EnvoyConfigRevision{
					TypeMeta: metav1.TypeMeta{Kind: "EnvoyConfigRevision", APIVersion: "v1alpha1"},
					ObjectMeta: metav1.ObjectMeta{
						Name: "ecr1", Namespace: "test",
						Labels: map[string]string{
							filters.NodeIDTag:   "node",
							filters.EnvoyAPITag: envoy.APIv3.String(),
							filters.VersionTag:  "xxxx",
						},
					},
				}),
			args:    args{"node", "xxxx", "test", envoy.APIv3},
			wantErr: false,
		},
		{
			name:    "Returns a function that does returns an error when called",
			cl:      fake.NewFakeClientWithScheme(s),
			args:    args{"node", "xxxx", "test", envoy.APIv3},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			fn := OnError(tt.cl)
			err := fn(tt.args.nodeID, tt.args.version, tt.args.msg, tt.args.envoyAPI)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}
}
