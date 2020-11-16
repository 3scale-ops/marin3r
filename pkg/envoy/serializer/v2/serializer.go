package envoy

import (
	"bytes"
	"encoding/base64"
	"fmt"

	envoy "github.com/3scale/marin3r/pkg/envoy"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"

	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"

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

type JSON struct{}

func (s JSON) Marshal(res envoy.Resource) (string, error) {

	m := jsonpb.Marshaler{OrigName: true}
	json := bytes.NewBuffer([]byte{})
	err := m.Marshal(json, res)
	if err != nil {
		return "", err
	}
	return string(json.Bytes()), nil
}

func (s JSON) Unmarshal(str string, res envoy.Resource) error {

	var err error
	switch o := res.(type) {

	case *envoy_api_v2.ClusterLoadAssignment:
		err = jsonpb.Unmarshal(bytes.NewReader([]byte(str)), o)

	case *envoy_api_v2.Cluster:
		err = jsonpb.Unmarshal(bytes.NewReader([]byte(str)), o)

	case *envoy_api_v2.RouteConfiguration:
		err = jsonpb.Unmarshal(bytes.NewReader([]byte(str)), o)

	case *envoy_api_v2.Listener:
		err = jsonpb.Unmarshal(bytes.NewReader([]byte(str)), o)

	case *envoy_service_discovery_v2.Runtime:
		err = jsonpb.Unmarshal(bytes.NewReader([]byte(str)), o)

	case *envoy_api_v2_auth.Secret:
		err = jsonpb.Unmarshal(bytes.NewReader([]byte(str)), o)

	default:
		err = fmt.Errorf("Unknown resource type")
	}

	if err != nil {
		return fmt.Errorf("Error deserializing resource: '%s'", err)
	}
	return nil
}

type B64JSON struct{}

func (s B64JSON) Unmarshal(str string, res envoy.Resource) error {
	b, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return fmt.Errorf("Error decoding base64 string: '%s'", err)
	}

	js := JSON{}
	err = js.Unmarshal(string(b), res)
	if err != nil {
		return err
	}

	return nil
}

type YAML struct{}

func (s YAML) Unmarshal(str string, res envoy.Resource) error {
	b, err := yaml.YAMLToJSON([]byte(str))
	if err != nil {
		return fmt.Errorf("Error converting yaml to json: '%s'", err)
	}

	js := JSON{}
	err = js.Unmarshal(string(b), res)
	if err != nil {
		return err
	}

	return nil
}
