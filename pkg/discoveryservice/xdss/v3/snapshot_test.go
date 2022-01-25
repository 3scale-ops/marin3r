package discoveryservice

import (
	"testing"

	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_resources "github.com/3scale-ops/marin3r/pkg/envoy/resources"
	testutil "github.com/3scale-ops/marin3r/pkg/util/test"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

func TestSnapshot_SetResource(t *testing.T) {
	type fields struct {
		v3 *cache_v3.Snapshot
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
			fields: fields{v3: &cache_v3.Snapshot{
				Resources: [8]cache_v3.Resources{
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
				}}},
			args: args{name: "endpoint", res: &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}},
			wantSnap: Snapshot{v3: &cache_v3.Snapshot{
				Resources: [8]cache_v3.Resources{
					{Version: "845f965864", Items: map[string]cache_types.ResourceWithTTL{
						"endpoint": {Resource: &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
				}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Snapshot{
				v3: tt.fields.v3,
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
		v3 *cache_v3.Snapshot
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
			fields: fields{v3: &cache_v3.Snapshot{
				Resources: [8]cache_v3.Resources{
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{
						"endpoint": {Resource: &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
				}}},
			args: args{rType: envoy.Endpoint},
			want: map[string]envoy.Resource{
				"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Snapshot{
				v3: tt.fields.v3,
			}
			if got := s.GetResources(tt.args.rType); !envoy_resources.ResourcesEqual(got, tt.want) {
				t.Errorf("Snapshot.GetResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshot_GetVersion(t *testing.T) {
	type fields struct {
		v3 *cache_v3.Snapshot
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
			fields: fields{v3: &cache_v3.Snapshot{
				Resources: [8]cache_v3.Resources{
					{Version: "1", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "2", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "3", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "4", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "5", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "6", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "7", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "8", Items: map[string]cache_types.ResourceWithTTL{}},
				}}},
			args: args{envoy.Endpoint},
			want: "1",
		},
		{
			name: "Returns the snapshot's version for the given resource type",
			fields: fields{v3: &cache_v3.Snapshot{
				Resources: [8]cache_v3.Resources{
					{Version: "1", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "2", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "3", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "4", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "5", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "6", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "7", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "8", Items: map[string]cache_types.ResourceWithTTL{}},
				}}},
			args: args{envoy.Secret},
			want: "6",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Snapshot{
				v3: tt.fields.v3,
			}
			if got := s.GetVersion(tt.args.rType); got != tt.want {
				t.Errorf("Snapshot.GetVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshot_SetVersion(t *testing.T) {
	type fields struct {
		v3 *cache_v3.Snapshot
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
			fields: fields{v3: &cache_v3.Snapshot{
				Resources: [8]cache_v3.Resources{
					{Version: "1", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "2", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "3", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "4", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "5", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "6", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "7", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "8", Items: map[string]cache_types.ResourceWithTTL{}},
				}}},
			args: args{rType: envoy.Secret, version: "xxxx"},
			wantSnap: Snapshot{v3: &cache_v3.Snapshot{
				Resources: [8]cache_v3.Resources{
					{Version: "1", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "2", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "3", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "4", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "5", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "7", Items: map[string]cache_types.ResourceWithTTL{}},
					{Version: "8", Items: map[string]cache_types.ResourceWithTTL{}},
				}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Snapshot{
				v3: tt.fields.v3,
			}
			s.SetVersion(tt.args.rType, tt.args.version)
			if !testutil.SnapshotsAreEqual(s, tt.wantSnap) {
				t.Errorf("Snapshot.SetVersion() = %v, want %v", s, tt.wantSnap)
			}
		})
	}
}

func Test_v3CacheResources(t *testing.T) {
	type args struct {
		rType envoy.Type
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Returns the internal resource type for the v3 secret",
			args: args{rType: envoy.Secret},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v3CacheResources(tt.args.rType); got != tt.want {
				t.Errorf("v3CacheResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshot_recalculateVersion(t *testing.T) {
	type fields struct {
		v3 *cache_v3.Snapshot
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
			name: "computes the hash of endpoints",
			fields: fields{
				v3: &cache_v3.Snapshot{
					Resources: [8]cache_v3.Resources{
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{
							"endpoint": {Resource: &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					},
				},
			},
			args: args{rType: envoy.Endpoint},
			want: "845f965864",
		},
		{
			name: "computes the hash of clusters",
			fields: fields{
				v3: &cache_v3.Snapshot{
					Resources: [8]cache_v3.Resources{
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "", Items: map[string]cache_types.ResourceWithTTL{
							"cluster": {Resource: &envoy_config_cluster_v3.Cluster{Name: "cluster"}},
						}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					},
				},
			},
			args: args{
				rType: envoy.Cluster,
			},
			want: "568989d74c",
		},
		{
			name: "computes the hash of secrets",
			fields: fields{
				v3: &cache_v3.Snapshot{
					Resources: [8]cache_v3.Resources{
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "56c6b8dc45", Items: map[string]cache_types.ResourceWithTTL{
							"secret": {Resource: &envoy_extensions_transport_sockets_tls_v3.Secret{
								Name: "secret",
								Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
									TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
										PrivateKey: &envoy_config_core_v3.DataSource{
											Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("key")},
										},
										CertificateChain: &envoy_config_core_v3.DataSource{
											Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("cert")},
										}}}}}}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
						{Version: "xxxx", Items: map[string]cache_types.ResourceWithTTL{}},
					},
				},
			},
			args: args{rType: envoy.Secret},
			want: "56c6b8dc45",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Snapshot{
				v3: tt.fields.v3,
			}
			if got := s.recalculateVersion(tt.args.rType); got != tt.want {
				t.Errorf("Snapshot.recalculateVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
