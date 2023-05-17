package discover

import (
	"context"
	"net"
	"reflect"
	"testing"

	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_resources "github.com/3scale-ops/marin3r/pkg/envoy/resources"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/go-logr/logr"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEndpoints(t *testing.T) {
	type args struct {
		ctx           context.Context
		cl            client.Client
		namespace     string
		clusterName   string
		portName      string
		labelSelector *metav1.LabelSelector
		generator     envoy_resources.Generator
		log           logr.Logger
	}
	tests := []struct {
		name    string
		args    args
		want    envoy.Resource
		wantErr bool
	}{
		{
			name: "Produces a cluster load assignment (endpoint) envoy resource",
			args: args{
				ctx: context.TODO(),
				cl: fake.NewClientBuilder().WithObjects(
					&discoveryv1.EndpointSlice{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "ns",
							Labels: map[string]string{
								"key": "value",
							},
						},
						AddressType: discoveryv1.AddressTypeIPv4,
						Endpoints: []discoveryv1.Endpoint{
							{
								Addresses:  []string{"127.0.0.1"},
								Conditions: discoveryv1.EndpointConditions{Ready: pointer.New(true)},
							},
							{
								Addresses:  []string{"127.0.0.2"},
								Conditions: discoveryv1.EndpointConditions{Ready: pointer.New(true)},
							},
						},
						Ports: []discoveryv1.EndpointPort{
							{Name: pointer.New("port"), Port: pointer.New(int32(1001))},
						},
					},
				).Build(),
				namespace:     "ns",
				clusterName:   "cluster",
				portName:      "port",
				labelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"key": "value"}},
				generator:     envoy_resources.NewGenerator(envoy.APIv3),
				log:           ctrl.Log.WithName("test"),
			},
			want: &envoy_config_endpoint_v3.ClusterLoadAssignment{
				ClusterName: "cluster",
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
														PortValue: 1001,
													},
												},
											},
										},
									},
								},
								HealthStatus: envoy_config_core_v3.HealthStatus_HEALTHY,
							},
							{
								HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
									Endpoint: &envoy_config_endpoint_v3.Endpoint{
										Address: &envoy_config_core_v3.Address{
											Address: &envoy_config_core_v3.Address_SocketAddress{
												SocketAddress: &envoy_config_core_v3.SocketAddress{
													Address: "127.0.0.2",
													PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
														PortValue: 1001,
													},
												},
											},
										},
									},
								},
								HealthStatus: envoy_config_core_v3.HealthStatus_HEALTHY,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Error, no endpoints returned (port not matched)",
			args: args{
				ctx: context.TODO(),
				cl: fake.NewClientBuilder().WithObjects(
					&discoveryv1.EndpointSlice{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "ns",
							Labels: map[string]string{
								"key": "value",
							},
						},
						AddressType: discoveryv1.AddressTypeIPv4,
						Endpoints: []discoveryv1.Endpoint{
							{
								Addresses:  []string{"127.0.0.1"},
								Conditions: discoveryv1.EndpointConditions{Ready: pointer.New(true)},
							},
						},
						Ports: []discoveryv1.EndpointPort{
							{Name: pointer.New("port"), Port: pointer.New(int32(1001))},
						},
					},
				).Build(),
				namespace:     "ns",
				clusterName:   "cluster",
				portName:      "non-existent-port",
				labelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"key": "value"}},
				generator:     envoy_resources.NewGenerator(envoy.APIv3),
				log:           ctrl.Log.WithName("test"),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Error, no endpoints returned for given label selector",
			args: args{
				ctx: context.TODO(),
				cl: fake.NewClientBuilder().WithObjects(
					&discoveryv1.EndpointSlice{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "ns",
							Labels: map[string]string{
								"key": "xxxx",
							},
						},
						AddressType: discoveryv1.AddressTypeIPv4,
						Endpoints: []discoveryv1.Endpoint{
							{
								Addresses:  []string{"127.0.0.1"},
								Conditions: discoveryv1.EndpointConditions{Ready: pointer.New(true)},
							},
						},
						Ports: []discoveryv1.EndpointPort{
							{Name: pointer.New("port"), Port: pointer.New(int32(1001))},
						},
					},
				).Build(),
				namespace:     "ns",
				clusterName:   "cluster",
				portName:      "non-existent-port",
				labelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"key": "value"}},
				generator:     envoy_resources.NewGenerator(envoy.APIv3),
				log:           ctrl.Log.WithName("test"),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Endpoints(tt.args.ctx, tt.args.cl, tt.args.namespace, tt.args.clusterName, tt.args.portName, tt.args.labelSelector, tt.args.generator, tt.args.log)
			if (err != nil) != tt.wantErr {
				t.Errorf("Endpoints() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Endpoints() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_endpointSlices_to_UpstreamHosts(t *testing.T) {
	type args struct {
		esl      *discoveryv1.EndpointSliceList
		portName string
		log      logr.Logger
	}
	tests := []struct {
		name    string
		args    args
		want    []envoy.UpstreamHost
		wantErr bool
	}{
		{
			name: "Generated the list of upstream hosts",
			args: args{
				esl: &discoveryv1.EndpointSliceList{
					Items: []discoveryv1.EndpointSlice{
						{
							AddressType: discoveryv1.AddressTypeIPv4,
							Endpoints: []discoveryv1.Endpoint{
								{
									Addresses:  []string{"127.0.0.1"},
									Conditions: discoveryv1.EndpointConditions{Ready: pointer.New(true)},
								},
								{
									Addresses:  []string{"127.0.0.2"},
									Conditions: discoveryv1.EndpointConditions{Ready: pointer.New(true)},
								},
							},
							Ports: []discoveryv1.EndpointPort{
								{Name: pointer.New("port1"), Port: pointer.New(int32(1001))},
								{Name: pointer.New("port2"), Port: pointer.New(int32(1002))},
							},
						},
						{
							AddressType: discoveryv1.AddressTypeIPv4,
							Endpoints: []discoveryv1.Endpoint{
								{
									Addresses:  []string{"127.0.0.3"},
									Conditions: discoveryv1.EndpointConditions{Ready: pointer.New(true)},
								},
								{
									Addresses:  []string{"127.0.0.4"},
									Conditions: discoveryv1.EndpointConditions{Ready: pointer.New(false)},
								},
							},
							Ports: []discoveryv1.EndpointPort{
								{Name: pointer.New("port1"), Port: pointer.New(int32(1001))},
								{Name: pointer.New("port2"), Port: pointer.New(int32(1002))},
							},
						},
					},
				},
				portName: "port1",
				log:      ctrl.Log.WithName("test"),
			},
			want: []envoy.UpstreamHost{
				{
					IP:     net.ParseIP("127.0.0.1"),
					Port:   1001,
					Health: envoy.HealthStatus_HEALTHY,
				},
				{
					IP:     net.ParseIP("127.0.0.2"),
					Port:   1001,
					Health: envoy.HealthStatus_HEALTHY,
				},
				{
					IP:     net.ParseIP("127.0.0.3"),
					Port:   1001,
					Health: envoy.HealthStatus_HEALTHY,
				},
				{
					IP:     net.ParseIP("127.0.0.4"),
					Port:   1001,
					Health: envoy.HealthStatus_UNHEALTHY,
				},
			},
			wantErr: false,
		},
		{
			name: "Error, unsupported address type",
			args: args{
				esl: &discoveryv1.EndpointSliceList{
					Items: []discoveryv1.EndpointSlice{{AddressType: discoveryv1.AddressTypeFQDN}},
				},
				portName: "port1",
				log:      ctrl.Log.WithName("test"),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Error, port not found",
			args: args{
				esl: &discoveryv1.EndpointSliceList{
					Items: []discoveryv1.EndpointSlice{{
						Ports: []discoveryv1.EndpointPort{
							{Name: pointer.New("port1"), Port: pointer.New(int32(1001))},
						}}},
				},
				portName: "other-port",
				log:      ctrl.Log.WithName("test"),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Continues on invalid IP address",
			args: args{
				esl: &discoveryv1.EndpointSliceList{
					Items: []discoveryv1.EndpointSlice{{
						AddressType: discoveryv1.AddressTypeIPv4,
						Endpoints: []discoveryv1.Endpoint{
							{
								Addresses:  []string{"xxxx"},
								Conditions: discoveryv1.EndpointConditions{Ready: pointer.New(true)},
							},
							{
								Addresses:  []string{"127.0.0.2"},
								Conditions: discoveryv1.EndpointConditions{Ready: pointer.New(true)},
							},
						},
						Ports: []discoveryv1.EndpointPort{
							{Name: pointer.New("port1"), Port: pointer.New(int32(1001))},
						},
					}},
				},
				portName: "port1",
				log:      ctrl.Log.WithName("test"),
			},
			want: []envoy.UpstreamHost{{
				IP:     net.ParseIP("127.0.0.2"),
				Port:   1001,
				Health: envoy.HealthStatus_HEALTHY,
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := endpointSlices_to_UpstreamHosts(tt.args.esl, tt.args.portName, tt.args.log)
			if (err != nil) != tt.wantErr {
				t.Errorf("endpointSlices_to_UpstreamHosts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("endpointSlices_to_UpstreamHosts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_health(t *testing.T) {
	type args struct {
		ec discoveryv1.EndpointConditions
	}
	tests := []struct {
		name string
		args args
		want envoy.EndpointHealthStatus
	}{
		{
			name: "HEALTHY",
			args: args{
				ec: discoveryv1.EndpointConditions{
					Ready:       pointer.New(true),
					Serving:     pointer.New(true),
					Terminating: pointer.New(false),
				},
			},
			want: envoy.HealthStatus_HEALTHY,
		},
		{
			name: "HEALTHY",
			args: args{
				ec: discoveryv1.EndpointConditions{
					Ready:   pointer.New(true),
					Serving: pointer.New(true),
				},
			},
			want: envoy.HealthStatus_HEALTHY,
		},
		{
			name: "HEALTHY",
			args: args{
				ec: discoveryv1.EndpointConditions{
					Ready: pointer.New(true),
				},
			},
			want: envoy.HealthStatus_HEALTHY,
		},
		{
			name: "UNHEALTHY",
			args: args{
				ec: discoveryv1.EndpointConditions{
					Ready: pointer.New(false),
				},
			},
			want: envoy.HealthStatus_UNHEALTHY,
		},
		{
			name: "UNHEALTHY",
			args: args{
				ec: discoveryv1.EndpointConditions{
					Serving: pointer.New(false),
				},
			},
			want: envoy.HealthStatus_UNHEALTHY,
		},
		{
			name: "UNKNOWN",
			args: args{
				ec: discoveryv1.EndpointConditions{},
			},
			want: envoy.HealthStatus_UNKNOWN,
		},
		{
			name: "DRAINING",
			args: args{
				ec: discoveryv1.EndpointConditions{
					Terminating: pointer.New(true),
				},
			},
			want: envoy.HealthStatus_DRAINING,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := health(tt.args.ec); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("health() = %v, want %v", got, tt.want)
			}
		})
	}
}
