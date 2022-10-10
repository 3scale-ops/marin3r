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
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

func TestSnapshot_SetResources(t *testing.T) {
	type args struct {
		rType     envoy.Type
		resources []envoy.Resource
	}
	tests := []struct {
		name     string
		snapshot xdss.Snapshot
		args     args
		want     xdss.Snapshot
	}{
		{
			name:     "Writes resources in the snapshot",
			snapshot: NewSnapshot(),
			args: args{
				rType: envoy.Endpoint,
				resources: []envoy.Resource{
					&envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
				},
			},
			want: func() xdss.Snapshot {
				s := NewSnapshot()
				s.v3.Resources[cache_types.Endpoint] = cache_v3.NewResources("845f965864", []cache_types.Resource{
					&envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
				})
				return s
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.snapshot
			if got := s.SetResources(tt.args.rType, tt.args.resources); !testutil.SnapshotsAreEqual(got, tt.want) {
				t.Errorf("Snapshot.SetResources() = %v, want %v", got.(Snapshot).v3.Resources, tt.want.(Snapshot).v3.Resources)
			}
		})
	}
}

func TestSnapshot_GetResources(t *testing.T) {
	type args struct {
		rType envoy.Type
	}
	tests := []struct {
		name     string
		snapshot xdss.Snapshot
		args     args
		want     map[string]envoy.Resource
	}{
		{
			name: "Returns a map with the snapshot resources",
			snapshot: NewSnapshot().SetResources(envoy.Endpoint, []envoy.Resource{
				&envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
			}),
			args: args{rType: envoy.Endpoint},
			want: map[string]envoy.Resource{
				"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.snapshot
			if got := s.GetResources(tt.args.rType); !envoy_resources.ResourcesEqual(got, tt.want) {
				t.Errorf("Snapshot.GetResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshot_GetVersion(t *testing.T) {
	type args struct {
		rType envoy.Type
	}
	tests := []struct {
		name     string
		snapshot xdss.Snapshot
		args     args
		want     string
	}{
		{
			name: "Returns the snapshot's version for the given resource type",
			snapshot: NewSnapshot().SetResources(envoy.Endpoint, []envoy.Resource{
				&envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
			}),
			args: args{envoy.Endpoint},
			want: "845f965864",
		},
		{
			name: "Returns the snapshot's version for the given resource type",
			snapshot: NewSnapshot().SetResources(envoy.Route, []envoy.Resource{
				&envoy_config_route_v3.RouteConfiguration{Name: "route"},
			}),
			args: args{envoy.Route},
			want: "6645547657",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.snapshot
			if got := s.GetVersion(tt.args.rType); got != tt.want {
				t.Errorf("Snapshot.GetVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSnapshot_SetVersion(t *testing.T) {
	type args struct {
		rType   envoy.Type
		version string
	}
	tests := []struct {
		name     string
		snapshot xdss.Snapshot
		args     args
	}{
		{
			name: "Writes the version for the given resource type",
			snapshot: NewSnapshot().SetResources(envoy.Route, []envoy.Resource{
				&envoy_config_route_v3.RouteConfiguration{Name: "listener"},
			}),
			args: args{rType: envoy.Secret, version: "xxxx"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.snapshot
			s.SetVersion(tt.args.rType, tt.args.version)
			if s.GetVersion(tt.args.rType) != tt.args.version {
				t.Errorf("Snapshot.SetVersion() = %v, want %v", s.GetVersion(tt.args.rType), tt.args.version)
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
			want: int(cache_types.Secret),
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
	type args struct {
		rType envoy.Type
	}
	tests := []struct {
		name     string
		snapshot Snapshot
		args     args
		want     string
	}{
		{
			name: "computes the hash of endpoints",
			// "endpoint": {Resource: &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}}},
			snapshot: func() Snapshot {
				s := NewSnapshot()
				s.v3.Resources[cache_types.Endpoint] = cache_v3.NewResources("xxxx", []cache_types.Resource{
					&envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
				})
				return s
			}(),
			args: args{rType: envoy.Endpoint},
			want: "845f965864",
		},
		{
			name: "computes the hash of clusters",
			snapshot: func() Snapshot {
				s := NewSnapshot()
				s.v3.Resources[cache_types.Cluster] = cache_v3.NewResources("xxxx", []cache_types.Resource{
					&envoy_config_cluster_v3.Cluster{Name: "cluster"},
				})
				return s
			}(),
			args: args{
				rType: envoy.Cluster,
			},
			want: "568989d74c",
		},
		{
			name: "computes the hash of secrets",
			snapshot: func() Snapshot {
				s := NewSnapshot()
				s.v3.Resources[cache_types.Secret] = cache_v3.NewResources("xxxx", []cache_types.Resource{
					&envoy_extensions_transport_sockets_tls_v3.Secret{
						Name: "secret",
						Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
							TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
								PrivateKey: &envoy_config_core_v3.DataSource{
									Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("key")},
								},
								CertificateChain: &envoy_config_core_v3.DataSource{
									Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("cert")},
								}}}},
				})
				return s
			}(),
			args: args{rType: envoy.Secret},
			want: "56c6b8dc45",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.snapshot
			if got := s.recalculateVersion(tt.args.rType); got != tt.want {
				t.Errorf("Snapshot.recalculateVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
