package reconcilers

import (
	"testing"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
)

func TestIsStatusReconciled(t *testing.T) {
	type args struct {
		ec               *marin3rv1alpha1.EnvoyConfig
		reconcileStatus  string
		publishedVersion string
		list             *marin3rv1alpha1.EnvoyConfigRevisionList
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStatusReconciled(tt.args.ec, tt.args.reconcileStatus, tt.args.publishedVersion, tt.args.list); got != tt.want {
				t.Errorf("IsStatusReconciled() = %v, want %v", got, tt.want)
			}
		})
	}
}
