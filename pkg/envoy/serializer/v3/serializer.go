package envoy

import (
	"bytes"
	"encoding/base64"
	"fmt"

	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_service_runtime_v3 "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"

	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"

	// This is the list of imports so all proto types are registered.
	// Generated with the following command in go-control-plane@v0.9.7
	//
	// for proto in $(find envoy -name '*.pb.go' | grep v3 | grep -v v3alpha); do echo "_ \"github.com/envoyproxy/go-control-plane/$(dirname $proto)\""; done | sort | uniq
	_ "github.com/3scale-ops/marin3r/pkg/envoy/protos/v3"
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

	case *envoy_config_endpoint_v3.ClusterLoadAssignment:
		err = jsonpb.Unmarshal(bytes.NewReader([]byte(str)), o)

	case *envoy_config_cluster_v3.Cluster:
		err = jsonpb.Unmarshal(bytes.NewReader([]byte(str)), o)

	case *envoy_config_route_v3.RouteConfiguration:
		err = jsonpb.Unmarshal(bytes.NewReader([]byte(str)), o)

	case *envoy_config_listener_v3.Listener:
		err = jsonpb.Unmarshal(bytes.NewReader([]byte(str)), o)

	case *envoy_service_runtime_v3.Runtime:
		err = jsonpb.Unmarshal(bytes.NewReader([]byte(str)), o)

	case *envoy_extensions_transport_sockets_tls_v3.Secret:
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
