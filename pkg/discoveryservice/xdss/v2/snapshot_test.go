package discoveryservice

import (
	"testing"

	xdss "github.com/3scale/marin3r/pkg/discoveryservice/xdss"
	envoy "github.com/3scale/marin3r/pkg/envoy"
	envoy_resources "github.com/3scale/marin3r/pkg/envoy/resources"
	testutil "github.com/3scale/marin3r/pkg/util/test"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

func TestSnapshot_Consistent(t *testing.T) {
	type fields struct {
		v2 *cache_v2.Snapshot
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Snapshot{
				v2: tt.fields.v2,
			}
			if err := s.Consistent(); (err != nil) != tt.wantErr {
				t.Errorf("Snapshot.Consistent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSnapshot_SetResource(t *testing.T) {
	type fields struct {
		v2 *cache_v2.Snapshot
	}
	type args struct {
		name string
		res  envoy.Resource
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantSnap xdss.Snapshot
	}{
		{
			name: "Writes resource in the snapshot",
			fields: fields{v2: &cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
				}}},
			args: args{name: "endpoint", res: &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint"}},
			wantSnap: Snapshot{v2: &cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"endpoint": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint"}}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
				}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Snapshot{
				v2: tt.fields.v2,
			}
			s.SetResource(tt.args.name, tt.args.res)
			if !testutil.SnapshotsAreEqual(s, tt.wantSnap) {
				t.Errorf("Snapshot.SetResource() = %v, want %v", s, tt.wantSnap)
			}
		})
	}
}

func TestSnapshot_GetResources(t *testing.T) {
	type fields struct {
		v2 *cache_v2.Snapshot
	}
	type args struct {
		rType envoy.Type
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]envoy.Resource
	}{
		{
			name: "Returns a map with the snapshot resources",
			fields: fields{v2: &cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"endpoint": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint"}}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
				}}},
			args: args{rType: envoy.Endpoint},
			want: map[string]envoy.Resource{
				"endpoint": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Snapshot{
				v2: tt.fields.v2,
			}
			if got := s.GetResources(tt.args.rType); !envoy_resources.ResourcesEqual(got, tt.want) {
				t.Errorf("Snapshot.GetResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshot_GetVersion(t *testing.T) {
	type fields struct {
		v2 *cache_v2.Snapshot
	}
	type args struct {
		rType envoy.Type
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "Returns the snapshot's version for the given resource type",
			fields: fields{v2: &cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "1", Items: map[string]cache_types.Resource{}},
					{Version: "2", Items: map[string]cache_types.Resource{}},
					{Version: "3", Items: map[string]cache_types.Resource{}},
					{Version: "4", Items: map[string]cache_types.Resource{}},
					{Version: "5", Items: map[string]cache_types.Resource{}},
					{Version: "6", Items: map[string]cache_types.Resource{}},
				}}},
			args: args{envoy.Endpoint},
			want: "1",
		},
		{
			name: "Returns the snapshot's version for the given resource type",
			fields: fields{v2: &cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "1", Items: map[string]cache_types.Resource{}},
					{Version: "2", Items: map[string]cache_types.Resource{}},
					{Version: "3", Items: map[string]cache_types.Resource{}},
					{Version: "4", Items: map[string]cache_types.Resource{}},
					{Version: "5", Items: map[string]cache_types.Resource{}},
					{Version: "6", Items: map[string]cache_types.Resource{}},
				}}},
			args: args{envoy.Secret},
			want: "5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Snapshot{
				v2: tt.fields.v2,
			}
			if got := s.GetVersion(tt.args.rType); got != tt.want {
				t.Errorf("Snapshot.GetVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshot_SetVersion(t *testing.T) {
	type fields struct {
		v2 *cache_v2.Snapshot
	}
	type args struct {
		rType   envoy.Type
		version string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantSnap xdss.Snapshot
	}{
		{
			name: "Writes the version for the given resource type",
			fields: fields{v2: &cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "1", Items: map[string]cache_types.Resource{}},
					{Version: "2", Items: map[string]cache_types.Resource{}},
					{Version: "3", Items: map[string]cache_types.Resource{}},
					{Version: "4", Items: map[string]cache_types.Resource{}},
					{Version: "5", Items: map[string]cache_types.Resource{}},
					{Version: "6", Items: map[string]cache_types.Resource{}},
				}}},
			args: args{rType: envoy.Secret, version: "xxxx"},
			wantSnap: Snapshot{v2: &cache_v2.Snapshot{
				Resources: [6]cache_v2.Resources{
					{Version: "1", Items: map[string]cache_types.Resource{}},
					{Version: "2", Items: map[string]cache_types.Resource{}},
					{Version: "3", Items: map[string]cache_types.Resource{}},
					{Version: "4", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "6", Items: map[string]cache_types.Resource{}},
				}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Snapshot{
				v2: tt.fields.v2,
			}
			s.SetVersion(tt.args.rType, tt.args.version)
			if !testutil.SnapshotsAreEqual(s, tt.wantSnap) {
				t.Errorf("Snapshot.SetVersion() = %v, want %v", s, tt.wantSnap)
			}
		})
	}
}

func Test_v2CacheResources(t *testing.T) {
	type args struct {
		rType envoy.Type
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Returns the internal resource type for the v2 snapshot",
			args: args{rType: envoy.Secret},
			want: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v2CacheResources(tt.args.rType); got != tt.want {
				t.Errorf("v2CacheResources() = %v, want %v", got, tt.want)
			}
		})
	}
}
