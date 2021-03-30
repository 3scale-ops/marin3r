package filters

import (
	"reflect"
	"testing"

	"github.com/3scale-ops/marin3r/pkg/envoy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestVersionFilter_ApplyToLabelSelector(t *testing.T) {
	tests := []struct {
		name     string
		selector client.MatchingLabels
		filter   *VersionFilter
		want     client.MatchingLabels
	}{
		{
			name:     "Applies VersionFilter to selector",
			filter:   &VersionFilter{Value: "xxxx"},
			selector: client.MatchingLabels{},
			want:     client.MatchingLabels{VersionTag: "xxxx"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.filter.ApplyToLabelSelector(tt.selector)
			if !reflect.DeepEqual(tt.selector, tt.want) {
				t.Errorf("VersionFilter_ApplyToLabelSelector() = %v, want %v", tt.selector, tt.want)
			}
		})
	}
}

func TestByVersion(t *testing.T) {
	type args struct {
		version string
	}
	tests := []struct {
		name string
		args args
		want RevisionFilter
	}{
		{
			name: "Returns a VersionFilter",
			args: args{version: "xxxx"},
			want: &VersionFilter{Value: "xxxx"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ByVersion(tt.args.version); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ByVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeIDFilter_ApplyToLabelSelector(t *testing.T) {

	tests := []struct {
		name     string
		filter   *NodeIDFilter
		selector client.MatchingLabels
		want     client.MatchingLabels
	}{
		{
			name:     "Applies NodeIDFilter to selector",
			filter:   &NodeIDFilter{Value: "xxxx"},
			selector: client.MatchingLabels{},
			want:     client.MatchingLabels{NodeIDTag: "xxxx"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.filter.ApplyToLabelSelector(tt.selector)
			if !reflect.DeepEqual(tt.selector, tt.want) {
				t.Errorf("NodeIDFilter_ApplyToLabelSelector() = %v, want %v", tt.selector, tt.want)
			}
		})
	}
}

func TestByNodeID(t *testing.T) {
	type args struct {
		nodeID string
	}
	tests := []struct {
		name string
		args args
		want RevisionFilter
	}{
		{
			name: "Returns a NodeIDFilter",
			args: args{nodeID: "xxxx"},
			want: &NodeIDFilter{Value: "xxxx"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ByNodeID(tt.args.nodeID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ByNodeID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvoyAPIFilter_ApplyToLabelSelector(t *testing.T) {
	tests := []struct {
		name     string
		selector client.MatchingLabels
		filter   *EnvoyAPIFilter
		want     client.MatchingLabels
	}{
		{
			name:     "Applies VersionFilter to selector",
			filter:   &EnvoyAPIFilter{Value: "v3"},
			selector: client.MatchingLabels{},
			want:     client.MatchingLabels{EnvoyAPITag: "v3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.filter.ApplyToLabelSelector(tt.selector)
			if !reflect.DeepEqual(tt.selector, tt.want) {
				t.Errorf("EnvoyAPIFilter_ApplyToLabelSelector() = %v, want %v", tt.selector, tt.want)
			}
		})
	}
}

func TestByEnvoyAPI(t *testing.T) {
	type args struct {
		envoyAPI envoy.APIVersion
	}
	tests := []struct {
		name string
		args args
		want RevisionFilter
	}{
		{
			name: "Returns a EnvoyAPIFilter",
			args: args{envoyAPI: envoy.APIv3},
			want: &EnvoyAPIFilter{Value: "v3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ByEnvoyAPI(tt.args.envoyAPI); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ByEnvoyAPI() = %v, want %v", got, tt.want)
			}
		})
	}
}
