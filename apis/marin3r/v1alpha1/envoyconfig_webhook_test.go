/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"testing"

	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestEnvoyConfig_ValidateResources(t *testing.T) {
	tests := []struct {
		name    string
		r       *EnvoyConfig
		wantErr bool
	}{
		{
			name: "Succeeds: type cluster",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type: "cluster",
						Value: &runtime.RawExtension{
							Raw: []byte(`{"name": "cluster"}`),
						},
					}},
				},
			},
			wantErr: false,
		},
		{
			name: "Fails: incorrect timeout",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type: "cluster",
						Value: &runtime.RawExtension{
							Raw: []byte(`{"name":"cluster1","type":"STRICT_DNS","connect_timeout":"xx"}`),
						},
					}},
				},
			}, wantErr: true,
		},
		{
			name: "Fails: missing resource value",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type: "cluster",
					}},
				},
			}, wantErr: true,
		},
		{
			name: "Fails: blueprint cannot be used for cluster",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type: "cluster",
						Value: &runtime.RawExtension{
							Raw: []byte(`{"name": "cluster"}`),
						},
						Blueprint: new(Blueprint),
					}},
				},
			}, wantErr: true,
		},
		{
			name: "Fails: generateFromEndpointSlice cannot be used for cluster",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type: "cluster",
						Value: &runtime.RawExtension{
							Raw: []byte(`{"name": "cluster"}`),
						},
						GenerateFromEndpointSlices: &GenerateFromEndpointSlices{},
					}},
				},
			}, wantErr: true,
		},
		{
			name: "Fails: generateFromTlsSecret cannot be used for cluster",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type: "cluster",
						Value: &runtime.RawExtension{
							Raw: []byte(`{"name": "cluster"}`),
						},
						GenerateFromTlsSecret: new(string),
					}},
				},
			}, wantErr: true,
		},
		{
			name: "Succeeds: type secret",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type:                  "secret",
						GenerateFromTlsSecret: new(string),
					}},
				},
			}, wantErr: false,
		},
		{
			name: "Fails: generateFromTlsSecret' cannot be empty for secret",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type: "secret",
					}},
				},
			}, wantErr: true,
		},
		{
			name: "Fails: value cannot be used for secret",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type:  "secret",
						Value: &runtime.RawExtension{},
					}},
				},
			}, wantErr: true,
		},
		{
			name: "Fails: generateFromEndpointSlice can only be used for endpoints",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type:                       "secret",
						GenerateFromEndpointSlices: &GenerateFromEndpointSlices{},
					}},
				},
			}, wantErr: true,
		},
		{
			name: "Succeeds: type endpoint",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type: "endpoint",
						GenerateFromEndpointSlices: &GenerateFromEndpointSlices{
							Selector:    &metav1.LabelSelector{MatchLabels: map[string]string{"label": "value"}},
							ClusterName: "test",
							TargetPort:  "port",
						},
					}},
				},
			}, wantErr: false,
		},
		{
			name: "Fails: one of value/generateFromEndpointSlice for endpoint",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type: "endpoint",
						GenerateFromEndpointSlices: &GenerateFromEndpointSlices{
							Selector:    &metav1.LabelSelector{MatchLabels: map[string]string{"label": "value"}},
							ClusterName: "test",
							TargetPort:  "port",
						}, Value: &runtime.RawExtension{},
					}},
				},
			}, wantErr: true,
		},
		{
			name: "Fails: missing value/generateFromEndpointSlice for endpoint",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type: "endpoint",
					}},
				},
			}, wantErr: true,
		},
		{
			name: "Fails: generateFromTlsSecret not allowed for endpoint",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type:                  "endpoint",
						GenerateFromTlsSecret: new(string),
					}},
				},
			}, wantErr: true,
		},
		{
			name: "Fails: blueprint not allowed for endpoint",
			r: &EnvoyConfig{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type:      "endpoint",
						Blueprint: new(Blueprint),
					}},
				},
			}, wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.r.ValidateResources(); (err != nil) != tt.wantErr {
				t.Errorf("EnvoyConfig.ValidateResources() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvoyConfig_ValidateEnvoyResources(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       EnvoyConfigSpec
		Status     EnvoyConfigStatus
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "fails for an EnvoyConfig with a syntax error in one of the envoy resources (from json)",
			fields: fields{
				Spec: EnvoyConfigSpec{
					NodeID:        "test",
					Serialization: pointer.New(envoy_serializer.JSON),
					EnvoyAPI:      pointer.New(envoy.APIv3),
					EnvoyResources: &EnvoyResources{
						Clusters: []EnvoyResource{{
							Name: pointer.New("cluster"),
							// the connect_timeout value unit is wrong
							Value: `{"name":"cluster1","type":"STRICT_DNS","connect_timeout":"2xs","load_assignment":{"cluster_name":"cluster1"}}`,
						}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "fails for an EnvoyConfig with a syntax error in one of the envoy resources (from yaml)",
			fields: fields{
				Spec: EnvoyConfigSpec{
					NodeID:        "test",
					Serialization: pointer.New(envoy_serializer.YAML),
					EnvoyAPI:      pointer.New(envoy.APIv3),
					EnvoyResources: &EnvoyResources{
						Listeners: []EnvoyResource{{
							Name: pointer.New("test"),
							// the "port" property should be "port_value"
							Value: `
                              name: listener1
                              address:
                                socket_address:
                                  address: 0.0.0.0
                                  port: 8443
                            `,
						}},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EnvoyConfig{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if err := r.ValidateEnvoyResources(); (err != nil) != tt.wantErr {
				t.Errorf("EnvoyConfig.ValidateEnvoyResources() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvoyConfig_Validate(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       EnvoyConfigSpec
		Status     EnvoyConfigStatus
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Ok, using spec.EnvoyResources",
			fields: fields{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					EnvoyResources: &EnvoyResources{
						Clusters: []EnvoyResource{{
							Name:  pointer.New("cluster"),
							Value: `{"name":"cluster1","type":"STRICT_DNS","connect_timeout":"2s","load_assignment":{"cluster_name":"cluster1"}}`,
						}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Ok, using spec.Resources",
			fields: fields{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
					Resources: []Resource{{
						Type: "cluster",
						Value: &runtime.RawExtension{
							Raw: []byte(`{"name":"cluster1","type":"STRICT_DNS","connect_timeout":"2s","load_assignment":{"cluster_name":"cluster1"}}`),
						},
					}},
				},
			},
			wantErr: false,
		},
		{
			name: "Fail, cannot use EnvoyResources and Resources both",
			fields: fields{
				Spec: EnvoyConfigSpec{
					NodeID:         "test",
					Resources:      []Resource{},
					EnvoyResources: &EnvoyResources{},
				},
			},
			wantErr: true,
		},
		{
			name: "Fail, must use one of EnvoyResources, Resources",
			fields: fields{
				Spec: EnvoyConfigSpec{
					NodeID: "test",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EnvoyConfig{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("EnvoyConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
