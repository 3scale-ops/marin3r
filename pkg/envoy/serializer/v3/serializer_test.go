package envoy

import (
	"testing"
	"time"

	envoy "github.com/3scale/marin3r/pkg/envoy"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_service_runtime_v3 "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	_struct "github.com/golang/protobuf/ptypes/struct"

	// This is the list of imports so all proto types are registered.
	// Generated with the following command in go-control-plane@v0.9.7
	//
	// for proto in $(find envoy -name '*.pb.go' | grep v3 | grep -v v3alpha); do echo "_ \"github.com/envoyproxy/go-control-plane/$(dirname $proto)\""; done | sort | uniq
	_ "github.com/envoyproxy/go-control-plane/envoy/admin/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/grpc_credential/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/metrics/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/overload/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/ratelimit/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/tap/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/trace/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/data/cluster/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/data/core/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/data/dns/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/data/tap/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/grpc/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/clusters/aggregate/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/clusters/dynamic_forward_proxy/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/clusters/redis/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/common/dynamic_forward_proxy/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/common/tap/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/common/fault/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/adaptive_concurrency/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/aws_lambda/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/aws_request_signing/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/buffer/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/compressor/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/csrf/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/dynamic_forward_proxy/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/dynamo/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ext_authz/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/fault/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_http1_bridge/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_http1_reverse_bridge/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_json_transcoder/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_stats/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_web/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/gzip/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/header_to_metadata/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ip_tagging/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/lua/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/on_demand/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/original_src/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ratelimit/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/squash/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/tap/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/http_inspector/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/original_dst/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/original_src/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/proxy_protocol/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/tls_inspector/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/client_ssl_auth/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/direct_response/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/dubbo_proxy/router/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/dubbo_proxy/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/echo/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/ext_authz/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/kafka_broker/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/local_ratelimit/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/mongo_proxy/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/mysql_proxy/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/ratelimit/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/rbac/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/redis_proxy/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/sni_cluster/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/thrift_proxy/filters/ratelimit/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/thrift_proxy/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/zookeeper_proxy/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/retry/host/omit_host_metadata/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/retry/priority/previous_priorities/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/alts/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/raw_buffer/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tap/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/wasm/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/event_reporting/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/health/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/metrics/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/status/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/tap/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/service/trace/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/type/metadata/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/type/tracing/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/type/v3"
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
	listener *envoy_config_listener_v3.Listener = &envoy_config_listener_v3.Listener{
		Name: "listener1",
		Address: &envoy_config_core_v3.Address{
			Address: &envoy_config_core_v3.Address_SocketAddress{
				SocketAddress: &envoy_config_core_v3.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
						PortValue: 8443,
					}}}}}

	endpointJSON string                                          = `{"cluster_name":"cluster1","endpoints":[{"lb_endpoints":[{"endpoint":{"address":{"socket_address":{"address":"127.0.0.1","port_value":8080}}}}]}]}`
	endpoint     *envoy_config_endpoint_v3.ClusterLoadAssignment = &envoy_config_endpoint_v3.ClusterLoadAssignment{
		ClusterName: "cluster1",
		Endpoints: []*envoy_config_endpoint_v3.LocalityLbEndpoints{
			{
				LbEndpoints: []*envoy_config_endpoint_v3.LbEndpoint{
					{
						HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
							Endpoint: &envoy_config_endpoint_v3.Endpoint{
								Address: &envoy_config_core_v3.Address{
									Address: &envoy_config_core_v3.Address_SocketAddress{
										SocketAddress: &envoy_config_core_v3.SocketAddress{
											Address: "127.0.0.1",
											PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
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
	cluster *envoy_config_cluster_v3.Cluster = &envoy_config_cluster_v3.Cluster{
		Name:           "cluster1",
		ConnectTimeout: ptypes.DurationProto(2 * time.Second),
		ClusterDiscoveryType: &envoy_config_cluster_v3.Cluster_Type{
			Type: envoy_config_cluster_v3.Cluster_STRICT_DNS,
		},
		LbPolicy: envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
		LoadAssignment: &envoy_config_endpoint_v3.ClusterLoadAssignment{
			ClusterName: "cluster1",
		},
	}

	secretJSON string                                            = `{"name":"cert1","tls_certificate":{"certificate_chain":{"inline_bytes":"eHh4eA=="},"private_key":{"inline_bytes":"eHh4eA=="}}}`
	secret     *envoy_extensions_transport_sockets_tls_v3.Secret = &envoy_extensions_transport_sockets_tls_v3.Secret{
		Name: "cert1",
		Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
			TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
				PrivateKey: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("xxxx")},
				},
				CertificateChain: &envoy_config_core_v3.DataSource{
					Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("xxxx")},
				}}}}

	routeJSON string                                    = `{"name":"route1","virtual_hosts":[{"name":"vhost","domains":["*"],"routes":[{"match":{"prefix":"/"},"direct_response":{"status":200}}]}]}`
	route     *envoy_config_route_v3.RouteConfiguration = &envoy_config_route_v3.RouteConfiguration{
		Name: "route1",
		VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
			Name:    "vhost",
			Domains: []string{"*"},
			Routes: []*envoy_config_route_v3.Route{{
				Match: &envoy_config_route_v3.RouteMatch{
					PathSpecifier: &envoy_config_route_v3.RouteMatch_Prefix{Prefix: "/"}},
				Action: &envoy_config_route_v3.Route_DirectResponse{
					DirectResponse: &envoy_config_route_v3.DirectResponseAction{Status: 200}},
			}},
		}},
	}

	runtimeJSON string                            = `{"name":"runtime1","layer":{"static_layer_0":"value"}}`
	runtime     *envoy_service_runtime_v3.Runtime = &envoy_service_runtime_v3.Runtime{
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
			args:    args{str: endpointJSON, res: &envoy_config_endpoint_v3.ClusterLoadAssignment{}},
			want:    endpoint,
			wantErr: false,
		},
		{
			name:    "Deserialize listener from json",
			s:       JSON{},
			args:    args{str: listenerJSON, res: &envoy_config_listener_v3.Listener{}},
			want:    listener,
			wantErr: false,
		},
		{
			name:    "Deserialize cluster from json",
			s:       JSON{},
			args:    args{str: clusterJSON, res: &envoy_config_cluster_v3.Cluster{}},
			want:    cluster,
			wantErr: false,
		},
		{
			name:    "Deserialize secret from json",
			s:       JSON{},
			args:    args{str: secretJSON, res: &envoy_extensions_transport_sockets_tls_v3.Secret{}},
			want:    secret,
			wantErr: false,
		},
		{
			name:    "Deserialize route from json",
			s:       JSON{},
			args:    args{str: routeJSON, res: &envoy_config_route_v3.RouteConfiguration{}},
			want:    route,
			wantErr: false,
		},
		{
			name:    "Deserialize runtime from json",
			s:       JSON{},
			args:    args{str: runtimeJSON, res: &envoy_service_runtime_v3.Runtime{}},
			want:    runtime,
			wantErr: false,
		},
		{
			name:    "Error deserializing resource",
			s:       JSON{},
			args:    args{str: `{"this_is": "wrong"}`, res: &envoy_config_route_v3.RouteConfiguration{}},
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
			args:    args{str: listenerB64JSON, res: &envoy_config_listener_v3.Listener{}},
			want:    listener,
			wantErr: false,
		},
		{
			name:    "Deserialize cluster from yaml",
			s:       B64JSON{},
			args:    args{str: clusterB64JSON, res: &envoy_config_cluster_v3.Cluster{}},
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
			args:    args{str: listenerYAML, res: &envoy_config_listener_v3.Listener{}},
			want:    listener,
			wantErr: false,
		},
		{
			name:    "Deserialize cluster from yaml",
			s:       YAML{},
			args:    args{str: clusterYAML, res: &envoy_config_cluster_v3.Cluster{}},
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
