package reconcilers

import (
	"testing"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
)

func TestIsStatusReconciled(t *testing.T) {
	type args struct {
		publishedVersion   string
		envoyConfigFactory func() *marin3rv1alpha1.EnvoyConfig
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStatusReconciled(tt.args.envoyConfigFactory(), tt.args.publishedVersion); got != tt.want {
				t.Errorf("IsStatusReconciled() = %v, want %v", got, tt.want)
			}
		})
	}
}
