package envoy

import (
	"reflect"
	"testing"
	"time"

	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoyapi_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoyapi_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoyapi_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	envoyapi_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoyapi_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/protobuf/ptypes"
	_struct "github.com/golang/protobuf/ptypes/struct"

	_ "github.com/cncf/udpa/go/udpa/annotations"
	_ "github.com/envoyproxy/go-control-plane/envoy/annotations"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/cluster/redis"
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
	_ "github.com/envoyproxy/go-control-plane/envoy/config/retry/previous_priorities"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/trace/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/transport_socket/raw_buffer/v2"
	xds_cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/golang/protobuf/proto"
)

var (
	listenerJSON    string = `{"name":"listener1","address":{"socketAddress":{"address":"0.0.0.0","portValue":8443}}}`
	listenerB64JSON string = "eyJuYW1lIjoibGlzdGVuZXIxIiwiYWRkcmVzcyI6eyJzb2NrZXRBZGRyZXNzIjp7ImFkZHJlc3MiOiIwLjAuMC4wIiwicG9ydFZhbHVlIjo4NDQzfX19Cg=="
	listenerYAML    string = `
        name: listener1
        address:
          socket_address:
            address: 0.0.0.0
            port_value: 8443
        `
	listener *envoyapi.Listener = &envoyapi.Listener{
		Name: "listener1",
		Address: &envoyapi_core.Address{
			Address: &envoyapi_core.Address_SocketAddress{
				SocketAddress: &envoyapi_core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &envoyapi_core.SocketAddress_PortValue{
						PortValue: 8443,
					}}}}}

	endpointJSON string                          = `{"clusterName":"cluster1","endpoints":[{"lbEndpoints":[{"endpoint":{"address":{"socketAddress":{"address":"127.0.0.1","portValue":8080}}}}]}]}`
	endpoint     *envoyapi.ClusterLoadAssignment = &envoyapi.ClusterLoadAssignment{
		ClusterName: "cluster1",
		Endpoints: []*envoyapi_endpoint.LocalityLbEndpoints{
			{
				LbEndpoints: []*envoyapi_endpoint.LbEndpoint{
					{
						HostIdentifier: &envoyapi_endpoint.LbEndpoint_Endpoint{
							Endpoint: &envoyapi_endpoint.Endpoint{
								Address: &envoyapi_core.Address{
									Address: &envoyapi_core.Address_SocketAddress{
										SocketAddress: &envoyapi_core.SocketAddress{
											Address: "127.0.0.1",
											PortSpecifier: &envoyapi_core.SocketAddress_PortValue{
												PortValue: 8080,
											}}}}}}}}}}}

	clusterJSON    string = `{"name":"cluster1","type":"STRICT_DNS","connectTimeout":"2s","loadAssignment":{"clusterName":"cluster1"}}`
	clusterB64JSON string = "eyJuYW1lIjoiY2x1c3RlcjEiLCJ0eXBlIjoiU1RSSUNUX0ROUyIsImNvbm5lY3RUaW1lb3V0IjoiMnMiLCJsb2FkQXNzaWdubWVudCI6eyJjbHVzdGVyTmFtZSI6ImNsdXN0ZXIxIn19Cg=="
	clusterYAML    string = `
        name: cluster1
        connect_timeout: 2s
        type: STRICT_DNS
        lb_policy: ROUND_ROBIN
        load_assignment:
          cluster_name: cluster1
        `
	cluster *envoyapi.Cluster = &envoyapi.Cluster{
		Name:           "cluster1",
		ConnectTimeout: ptypes.DurationProto(2 * time.Second),
		ClusterDiscoveryType: &envoyapi.Cluster_Type{
			Type: envoyapi.Cluster_STRICT_DNS,
		},
		LbPolicy: envoyapi.Cluster_ROUND_ROBIN,
		LoadAssignment: &envoyapi.ClusterLoadAssignment{
			ClusterName: "cluster1",
		},
	}

	secretJSON string                = `{"name":"cert1","tlsCertificate":{"certificateChain":{"inlineBytes":"eHh4eA=="},"privateKey":{"inlineBytes":"eHh4eA=="}}}`
	secret     *envoyapi_auth.Secret = &envoyapi_auth.Secret{
		Name: "cert1",
		Type: &envoyapi_auth.Secret_TlsCertificate{
			TlsCertificate: &envoyapi_auth.TlsCertificate{
				PrivateKey: &envoyapi_core.DataSource{
					Specifier: &envoyapi_core.DataSource_InlineBytes{InlineBytes: []byte("xxxx")},
				},
				CertificateChain: &envoyapi_core.DataSource{
					Specifier: &envoyapi_core.DataSource_InlineBytes{InlineBytes: []byte("xxxx")},
				}}}}

	routeJSON string                = `{"name":"route1","match":{"prefix":"/"},"directResponse":{"status":200}}`
	route     *envoyapi_route.Route = &envoyapi_route.Route{
		Name: "route1",
		Match: &envoyapi_route.RouteMatch{
			PathSpecifier: &envoyapi_route.RouteMatch_Prefix{Prefix: "/"}},
		Action: &envoyapi_route.Route_DirectResponse{
			DirectResponse: &envoyapi_route.DirectResponseAction{Status: 200}},
	}

	runtimeJSON string                      = `{"name":"runtime1","layer":{"static_layer_0":"value"}}`
	runtime     *envoyapi_discovery.Runtime = &envoyapi_discovery.Runtime{
		Name: "runtime1",
		// See https://www.envoyproxy.io/docs/envoy/latest/configuration/operations/runtime
		Layer: &_struct.Struct{
			Fields: map[string]*_struct.Value{
				"static_layer_0": {Kind: &_struct.Value_StringValue{StringValue: "value"}},
			}}}
)

func TestYAMLtoResources(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *Resources
		wantErr bool
	}{
		{
			name: "Loads yaml into evnoy resources",
			args: args{data: []byte(`
                clusters:
                - name: cluster1
                  connect_timeout: 2s
                  type: STRICT_DNS
                  lb_policy: ROUND_ROBIN
                  load_assignment:
                    cluster_name: cluster1
                    endpoints: []

                listeners:
                - name: listener1
                  address:
                    socket_address:
                      address: 0.0.0.0
                      port_value: 8443
                `,
			)},
			want: &Resources{
				Clusters:  []*envoyapi.Cluster{cluster},
				Listeners: []*envoyapi.Listener{listener},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := YAMLtoResources(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("YAMLtoResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !proto.Equal(got.Clusters[0], tt.want.Clusters[0]) {
				t.Errorf("YAMLtoResources() = %v, want %v", got.Clusters[0], tt.want.Clusters[0])
			}
			if !proto.Equal(got.Listeners[0], tt.want.Listeners[0]) {
				t.Errorf("YAMLtoResources() = %v, want %v", got.Listeners[0], tt.want.Listeners[0])
			}
		})
	}
}

func TestResourcesToJSON(t *testing.T) {
	type args struct {
		pb proto.Message
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResourcesToJSON(tt.args.pb)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResourcesToJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResourcesToJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSON_Marshal(t *testing.T) {
	type args struct {
		res xds_cache_types.Resource
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
		res xds_cache_types.Resource
	}
	tests := []struct {
		name    string
		s       JSON
		args    args
		want    xds_cache_types.Resource
		wantErr bool
	}{
		{
			name:    "Deserialize endpoint from json",
			s:       JSON{},
			args:    args{str: endpointJSON, res: &envoyapi.ClusterLoadAssignment{}},
			want:    endpoint,
			wantErr: false,
		},
		{
			name:    "Deserialize listener from json",
			s:       JSON{},
			args:    args{str: listenerJSON, res: &envoyapi.Listener{}},
			want:    listener,
			wantErr: false,
		},
		{
			name:    "Deserialize cluster from json",
			s:       JSON{},
			args:    args{str: clusterJSON, res: &envoyapi.Cluster{}},
			want:    cluster,
			wantErr: false,
		},
		{
			name:    "Deserialize secret from json",
			s:       JSON{},
			args:    args{str: secretJSON, res: &envoyapi_auth.Secret{}},
			want:    secret,
			wantErr: false,
		},
		{
			name:    "Deserialize route from json",
			s:       JSON{},
			args:    args{str: routeJSON, res: &envoyapi_route.Route{}},
			want:    route,
			wantErr: false,
		},
		{
			name:    "Deserialize runtime from json",
			s:       JSON{},
			args:    args{str: runtimeJSON, res: &envoyapi_discovery.Runtime{}},
			want:    runtime,
			wantErr: false,
		},
		{
			name:    "Error deserializing resource",
			s:       JSON{},
			args:    args{str: `{"this_is": "wrong"}`, res: &envoyapi_route.Route{}},
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
		res xds_cache_types.Resource
	}
	tests := []struct {
		name    string
		s       B64JSON
		args    args
		want    xds_cache_types.Resource
		wantErr bool
	}{
		{
			name:    "Deserialize listener from yaml",
			s:       B64JSON{},
			args:    args{str: listenerB64JSON, res: &envoyapi.Listener{}},
			want:    listener,
			wantErr: false,
		},
		{
			name:    "Deserialize cluster from yaml",
			s:       B64JSON{},
			args:    args{str: clusterB64JSON, res: &envoyapi.Cluster{}},
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
		res xds_cache_types.Resource
	}
	tests := []struct {
		name    string
		s       YAML
		args    args
		want    xds_cache_types.Resource
		wantErr bool
	}{
		{
			name:    "Deserialize listener from yaml",
			s:       YAML{},
			args:    args{str: listenerYAML, res: &envoyapi.Listener{}},
			want:    listener,
			wantErr: false,
		},
		{
			name:    "Deserialize cluster from yaml",
			s:       YAML{},
			args:    args{str: clusterYAML, res: &envoyapi.Cluster{}},
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
