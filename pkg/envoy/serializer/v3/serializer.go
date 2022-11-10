package envoy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	_ "github.com/3scale-ops/marin3r/pkg/envoy/protos/v3"
	"github.com/ghodss/yaml"
	"google.golang.org/protobuf/encoding/protojson"
)

type JSON struct{}

func (s JSON) Marshal(res envoy.Resource) (string, error) {

	opts := protojson.MarshalOptions{UseProtoNames: true, Indent: ""}
	data, err := opts.Marshal(res)
	if err != nil {
		return "", err
	}

	// The output of jsonpb.Marshal is not stable so we need
	// to use the json package to produce stable json output
	// See https://github.com/golang/protobuf/issues/1082
	data2, err := json.Marshal(json.RawMessage(data))
	if err != nil {
		return "", err
	}

	return string(data2), nil
}

func (s JSON) Unmarshal(str string, res envoy.Resource) error {
	if res == nil {
		return fmt.Errorf("resource cannot be nil")
	}

	err := protojson.Unmarshal([]byte(str), res)
	if err != nil {
		return fmt.Errorf("error deserializing resource: '%s'", err)
	}
	return nil
}

type B64JSON struct{}

func (s B64JSON) Unmarshal(str string, res envoy.Resource) error {
	b, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return fmt.Errorf("error decoding base64 string: '%s'", err)
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
		return fmt.Errorf("error converting yaml to json: '%s'", err)
	}

	js := JSON{}
	err = js.Unmarshal(string(b), res)
	if err != nil {
		return err
	}

	return nil
}
