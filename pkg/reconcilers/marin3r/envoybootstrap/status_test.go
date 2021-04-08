package reconcilers

import (
	"testing"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	"k8s.io/utils/pointer"
)

func TestIsStatusReconciled(t *testing.T) {
	type args struct {
		eb           *marin3rv1alpha1.EnvoyBootstrap
		configHashV2 string
		configHashV3 string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Returns true, status up to date",
			args: args{
				eb: &marin3rv1alpha1.EnvoyBootstrap{
					Status: marin3rv1alpha1.EnvoyBootstrapStatus{
						ConfigHashV2: pointer.StringPtr("xxxx"),
						ConfigHashV3: pointer.StringPtr("xxxx"),
					},
				},
				configHashV2: "xxxx",
				configHashV3: "xxxx",
			},
			want: true,
		},
		{
			name: "Returns false, configHashV2 needs update",
			args: args{
				eb: &marin3rv1alpha1.EnvoyBootstrap{
					Status: marin3rv1alpha1.EnvoyBootstrapStatus{
						ConfigHashV2: pointer.StringPtr("yyyy"),
						ConfigHashV3: pointer.StringPtr("xxxx"),
					},
				},
				configHashV2: "xxxx",
				configHashV3: "xxxx",
			},
			want: false,
		},
		{
			name: "Returns false, certificateHash needs update",
			args: args{
				eb: &marin3rv1alpha1.EnvoyBootstrap{
					Status: marin3rv1alpha1.EnvoyBootstrapStatus{
						ConfigHashV2: pointer.StringPtr("xxxx"),
						ConfigHashV3: pointer.StringPtr("yyyy"),
					},
				},
				configHashV2: "xxxx",
				configHashV3: "xxxx",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStatusReconciled(tt.args.eb, tt.args.configHashV2, tt.args.configHashV3); got != tt.want {
				t.Errorf("IsStatusReconciled() = %v, want %v", got, tt.want)
			}
		})
	}
}
