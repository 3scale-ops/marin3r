package envoy

import (
	"testing"
	"time"

	envoy "github.com/3scale/marin3r/pkg/envoy"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	_struct "github.com/golang/protobuf/ptypes/struct"

	// This is the list of imports so all proto types are registered.
	// Generated with the following command in go-control-plane@v0.9.7
	//
	// for proto in $(find envoy -name '*.pb.go' | grep v2 | grep -v v2alpha); do echo "_ \"github.com/envoyproxy/go-control-plane/$(dirname $proto)\""; done | sort | uniq
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/ratelimit"
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/fault/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/buffer/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/compressor/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/cors/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/csrf/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/dynamo/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/ext_authz/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/fault/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/grpc_http1_bridge/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/grpc_web/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/gzip/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/header_to_metadata/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/health_check/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/ip_tagging/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/lua/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/on_demand/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/rate_limit/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/rbac/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/router/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/squash/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/transcoder/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/listener/http_inspector/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/listener/original_dst/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/listener/proxy_protocol/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/listener/tls_inspector/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/client_ssl_auth/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/direct_response/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/echo/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/ext_authz/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/mongo_proxy/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/rate_limit/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/rbac/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/redis_proxy/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/sni_cluster/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/health_checker/redis/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/listener/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/metrics/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/ratelimit/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/retry/omit_canary_hosts/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/retry/omit_host_metadata/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/retry/previous_hosts/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/trace/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/transport_socket/raw_buffer/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/metrics/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/status/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/trace/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/type/metadata/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/type/tracing/v2"
)

var (
	listenerJSON    string = `{"name":"listener1","address":{"socket_address":{"address":"0.0.0.0","port_value":8443}}}`
	listenerB64JSON string = "eyJuYW1lIjoibGlzdGVuZXIxIiwiYWRkcmVzcyI6eyJzb2NrZXRfYWRkcmVzcyI6eyJhZGRyZXNzIjoiMC4wLjAuMCIsInBvcnRfdmFsdWUiOjg0NDN9fX0K"
	listenerYAML    string = `
        name: listener1
        address:
          socket_address:
            address: 0.0.0.0
            port_value: 8443
        `
	listener *envoy_api_v2.Listener = &envoy_api_v2.Listener{
		Name: "listener1",
		Address: &envoy_api_v2_core.Address{
			Address: &envoy_api_v2_core.Address_SocketAddress{
				SocketAddress: &envoy_api_v2_core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
						PortValue: 8443,
					}}}}}

	endpointJSON string                              = `{"cluster_name":"cluster1","endpoints":[{"lb_endpoints":[{"endpoint":{"address":{"socket_address":{"address":"127.0.0.1","port_value":8080}}}}]}]}`
	endpoint     *envoy_api_v2.ClusterLoadAssignment = &envoy_api_v2.ClusterLoadAssignment{
		ClusterName: "cluster1",
		Endpoints: []*envoy_api_v2_endpoint.LocalityLbEndpoints{
			{
				LbEndpoints: []*envoy_api_v2_endpoint.LbEndpoint{
					{
						HostIdentifier: &envoy_api_v2_endpoint.LbEndpoint_Endpoint{
							Endpoint: &envoy_api_v2_endpoint.Endpoint{
								Address: &envoy_api_v2_core.Address{
									Address: &envoy_api_v2_core.Address_SocketAddress{
										SocketAddress: &envoy_api_v2_core.SocketAddress{
											Address: "127.0.0.1",
											PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
												PortValue: 8080,
											}}}}}}}}}}}

	clusterJSON    string = `{"name":"cluster1","type":"STRICT_DNS","connect_timeout":"2s","load_assignment":{"cluster_name":"cluster1"}}`
	clusterB64JSON string = "eyJuYW1lIjoiY2x1c3RlcjEiLCJ0eXBlIjoiU1RSSUNUX0ROUyIsImNvbm5lY3RfdGltZW91dCI6IjJzIiwibG9hZF9hc3NpZ25tZW50Ijp7ImNsdXN0ZXJfbmFtZSI6ImNsdXN0ZXIxIn19Cg=="
	clusterYAML    string = `
        name: cluster1
        connect_timeout: 2s
        type: STRICT_DNS
        lb_policy: ROUND_ROBIN
        load_assignment:
          cluster_name: cluster1
        `
	cluster *envoy_api_v2.Cluster = &envoy_api_v2.Cluster{
		Name:           "cluster1",
		ConnectTimeout: ptypes.DurationProto(2 * time.Second),
		ClusterDiscoveryType: &envoy_api_v2.Cluster_Type{
			Type: envoy_api_v2.Cluster_STRICT_DNS,
		},
		LbPolicy: envoy_api_v2.Cluster_ROUND_ROBIN,
		LoadAssignment: &envoy_api_v2.ClusterLoadAssignment{
			ClusterName: "cluster1",
		},
	}

	secretJSON string                    = `{"name":"cert1","tls_certificate":{"certificate_chain":{"inline_bytes":"eHh4eA=="},"private_key":{"inline_bytes":"eHh4eA=="}}}`
	secret     *envoy_api_v2_auth.Secret = &envoy_api_v2_auth.Secret{
		Name: "cert1",
		Type: &envoy_api_v2_auth.Secret_TlsCertificate{
			TlsCertificate: &envoy_api_v2_auth.TlsCertificate{
				PrivateKey: &envoy_api_v2_core.DataSource{
					Specifier: &envoy_api_v2_core.DataSource_InlineBytes{InlineBytes: []byte("xxxx")},
				},
				CertificateChain: &envoy_api_v2_core.DataSource{
					Specifier: &envoy_api_v2_core.DataSource_InlineBytes{InlineBytes: []byte("xxxx")},
				}}}}

	routeJSON string                           = `{"name":"route1","virtual_hosts":[{"name":"vhost","domains":["*"],"routes":[{"match":{"prefix":"/"},"direct_response":{"status":200}}]}]}`
	route     *envoy_api_v2.RouteConfiguration = &envoy_api_v2.RouteConfiguration{
		Name: "route1",
		VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
			Name:    "vhost",
			Domains: []string{"*"},
			Routes: []*envoy_api_v2_route.Route{{
				Match: &envoy_api_v2_route.RouteMatch{
					PathSpecifier: &envoy_api_v2_route.RouteMatch_Prefix{Prefix: "/"}},
				Action: &envoy_api_v2_route.Route_DirectResponse{
					DirectResponse: &envoy_api_v2_route.DirectResponseAction{Status: 200}},
			}},
		}},
	}

	runtimeJSON string                              = `{"name":"runtime1","layer":{"static_layer_0":"value"}}`
	runtime     *envoy_service_discovery_v2.Runtime = &envoy_service_discovery_v2.Runtime{
		Name: "runtime1",
		// See https://www.envoyproxy.io/docs/envoy/latest/configuration/operations/runtime
		Layer: &_struct.Struct{
			Fields: map[string]*_struct.Value{
				"static_layer_0": {Kind: &_struct.Value_StringValue{StringValue: "value"}},
			}}}
)

func TestJSON_Marshal(t *testing.T) {
	type args struct {
		res envoy.Resource
	}
	tests := []struct {
		name    string
		s       JSON
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "Serialize listener to json",
			s:       JSON{},
			args:    args{res: listener},
			want:    listenerJSON,
			wantErr: false,
		},
		{
			name:    "Serialize endpoint to json",
			s:       JSON{},
			args:    args{res: endpoint},
			want:    endpointJSON,
			wantErr: false,
		},
		{
			name:    "Serialize cluster to json",
			s:       JSON{},
			args:    args{res: cluster},
			want:    clusterJSON,
			wantErr: false,
		},
		{
			name:    "Serialize secret to json",
			s:       JSON{},
			args:    args{res: secret},
			want:    secretJSON,
			wantErr: false,
		},
		{
			name:    "Serialize route to json",
			s:       JSON{},
			args:    args{res: route},
			want:    routeJSON,
			wantErr: false,
		},
		{
			name:    "Serialize runtime to json",
			s:       JSON{},
			args:    args{res: runtime},
			want:    runtimeJSON,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.Marshal(tt.args.res)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSON.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("JSON.Marshal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSON_Unmarshal(t *testing.T) {
	type args struct {
		str string
		res envoy.Resource
	}
	tests := []struct {
		name    string
		s       JSON
		args    args
		want    envoy.Resource
		wantErr bool
	}{
		{
			name:    "Deserialize endpoint from json",
			s:       JSON{},
			args:    args{str: endpointJSON, res: &envoy_api_v2.ClusterLoadAssignment{}},
			want:    endpoint,
			wantErr: false,
		},
		{
			name:    "Deserialize listener from json",
			s:       JSON{},
			args:    args{str: listenerJSON, res: &envoy_api_v2.Listener{}},
			want:    listener,
			wantErr: false,
		},
		{
			name:    "Deserialize cluster from json",
			s:       JSON{},
			args:    args{str: clusterJSON, res: &envoy_api_v2.Cluster{}},
			want:    cluster,
			wantErr: false,
		},
		{
			name:    "Deserialize secret from json",
			s:       JSON{},
			args:    args{str: secretJSON, res: &envoy_api_v2_auth.Secret{}},
			want:    secret,
			wantErr: false,
		},
		{
			name:    "Deserialize route from json",
			s:       JSON{},
			args:    args{str: routeJSON, res: &envoy_api_v2.RouteConfiguration{}},
			want:    route,
			wantErr: false,
		},
		{
			name:    "Deserialize runtime from json",
			s:       JSON{},
			args:    args{str: runtimeJSON, res: &envoy_service_discovery_v2.Runtime{}},
			want:    runtime,
			wantErr: false,
		},
		{
			name:    "Error deserializing resource",
			s:       JSON{},
			args:    args{str: `{"this_is": "wrong"}`, res: &envoy_api_v2.RouteConfiguration{}},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Error deserializing resource: unknown type",
			s:       JSON{},
			args:    args{str: `{"this_is": "wrong"}`, res: nil},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.Unmarshal(tt.args.str, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("JSON.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !proto.Equal(tt.args.res, tt.want) {
				t.Errorf("JSON.Unmarshal() = %v, want %v", tt.args.res, tt.want)
			}
		})
	}
}

func TestB64JSON_Unmarshal(t *testing.T) {
	type args struct {
		str string
		res envoy.Resource
	}
	tests := []struct {
		name    string
		s       B64JSON
		args    args
		want    envoy.Resource
		wantErr bool
	}{
		{
			name:    "Deserialize listener from yaml",
			s:       B64JSON{},
			args:    args{str: listenerB64JSON, res: &envoy_api_v2.Listener{}},
			want:    listener,
			wantErr: false,
		},
		{
			name:    "Deserialize cluster from yaml",
			s:       B64JSON{},
			args:    args{str: clusterB64JSON, res: &envoy_api_v2.Cluster{}},
			want:    cluster,
			wantErr: false,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.Unmarshal(tt.args.str, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("B64JSON.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !proto.Equal(tt.args.res, tt.want) {
				t.Errorf("JSON.Unmarshal() = %v, want %v", tt.args.res, tt.want)
			}
		})
	}
}

func TestYAML_Unmarshal(t *testing.T) {
	type args struct {
		str string
		res envoy.Resource
	}
	tests := []struct {
		name    string
		s       YAML
		args    args
		want    envoy.Resource
		wantErr bool
	}{
		{
			name:    "Deserialize listener from yaml",
			s:       YAML{},
			args:    args{str: listenerYAML, res: &envoy_api_v2.Listener{}},
			want:    listener,
			wantErr: false,
		},
		{
			name:    "Deserialize cluster from yaml",
			s:       YAML{},
			args:    args{str: clusterYAML, res: &envoy_api_v2.Cluster{}},
			want:    cluster,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.Unmarshal(tt.args.str, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("YAML.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !proto.Equal(tt.args.res, tt.want) {
				t.Errorf("JSON.Unmarshal() = %v, want %v", tt.args.res, tt.want)
			}
		})
	}
}
