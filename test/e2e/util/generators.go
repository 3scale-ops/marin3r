package e2e

import (
	"fmt"
	"time"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	envoy "github.com/3scale/marin3r/pkg/envoy"
	envoy_serializer "github.com/3scale/marin3r/pkg/envoy/serializer"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	envoy_api_v2_listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	http_connection_manager_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

const (
	DeploymentLabelKey   string = "app"
	DeploymentLabelValue string = "nginx"
	PodPort              uint32 = 8080
)

func GenerateDeploymentWithInjection(key types.NamespacedName, nodeID, envoyAPI, envoyVersion string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{DeploymentLabelKey: DeploymentLabelValue},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						DeploymentLabelKey:          DeploymentLabelValue,
						"marin3r.3scale.net/status": "enabled",
					},
					Annotations: map[string]string{
						"marin3r.3scale.net/node-id":           nodeID,
						"marin3r.3scale.net/envoy-extra-args":  "--component-log-level config:debug",
						"marin3r.3scale.net/ports":             "envoy-http:8080",
						"marin3r.3scale.net/envoy-api-version": envoyAPI,
						"marin3r.3scale.net/envoy-image":       fmt.Sprintf("envoyproxy/envoy:%s", envoyVersion),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "nginx",
						Image: "nginxdemos/hello:plain-text",
						Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}},
					}},
				},
			},
		},
	}
}

func GenerateEnvoyConfig(key types.NamespacedName, nodeID string, envoyAPI envoy.APIVersion,
	endpointsGenFn, clustersGenFn, routesGenFn, listenersGenFn func() map[string]envoy.Resource) *envoyv1alpha1.EnvoyConfig {
	m := envoy_serializer.NewResourceMarshaller(envoy_serializer.JSON, envoyAPI)

	return &envoyv1alpha1.EnvoyConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: envoyv1alpha1.EnvoyConfigSpec{
			EnvoyAPI: pointer.StringPtr(envoyAPI.String()),
			NodeID:   nodeID,
			EnvoyResources: &envoyv1alpha1.EnvoyResources{
				Endpoints: func() []envoyv1alpha1.EnvoyResource {
					endpoints := []envoyv1alpha1.EnvoyResource{}
					for name, resource := range endpointsGenFn() {
						json, err := m.Marshal(resource)
						if err != nil {
							panic(err)
						}
						endpoints = append(endpoints, envoyv1alpha1.EnvoyResource{Name: name, Value: json})
					}
					return endpoints
				}(),
				Clusters: func() []envoyv1alpha1.EnvoyResource {
					clusters := []envoyv1alpha1.EnvoyResource{}
					for name, resource := range clustersGenFn() {
						json, err := m.Marshal(resource)
						if err != nil {
							panic(err)
						}
						clusters = append(clusters, envoyv1alpha1.EnvoyResource{Name: name, Value: json})
					}
					return clusters
				}(),
				Routes: func() []envoyv1alpha1.EnvoyResource {
					routes := []envoyv1alpha1.EnvoyResource{}
					for name, resource := range routesGenFn() {
						json, err := m.Marshal(resource)
						if err != nil {
							panic(err)
						}
						routes = append(routes, envoyv1alpha1.EnvoyResource{Name: name, Value: json})
					}
					return routes
				}(),
				Listeners: func() []envoyv1alpha1.EnvoyResource {
					listeners := []envoyv1alpha1.EnvoyResource{}
					for name, resource := range listenersGenFn() {
						json, err := m.Marshal(resource)
						if err != nil {
							panic(err)
						}
						listeners = append(listeners, envoyv1alpha1.EnvoyResource{Name: name, Value: json})
					}
					return listeners
				}(),
			},
		},
	}

}

func HTTPListenerWithRdsV2(listenerName, routeName string) (string, *envoy_api_v2.Listener) {
	return listenerName, &envoy_api_v2.Listener{
		Name: listenerName,
		Address: &envoy_api_v2_core.Address{
			Address: &envoy_api_v2_core.Address_SocketAddress{
				SocketAddress: &envoy_api_v2_core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
						PortValue: PodPort,
					}}}},
		FilterChains: []*envoy_api_v2_listener.FilterChain{{
			Filters: []*envoy_api_v2_listener.Filter{{
				Name: "envoy.http_connection_manager",
				ConfigType: &envoy_api_v2_listener.Filter_TypedConfig{
					TypedConfig: func() *any.Any {
						any, err := ptypes.MarshalAny(
							&http_connection_manager_v2.HttpConnectionManager{
								StatPrefix: listenerName,
								RouteSpecifier: &http_connection_manager_v2.HttpConnectionManager_Rds{
									Rds: &http_connection_manager_v2.Rds{
										ConfigSource: &envoy_api_v2_core.ConfigSource{
											ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
												Ads: &envoy_api_v2_core.AggregatedConfigSource{},
											},
											ResourceApiVersion: envoy_api_v2_core.ApiVersion_V2,
										},
										RouteConfigName: routeName,
									},
								},
								HttpFilters: []*http_connection_manager_v2.HttpFilter{{Name: "envoy.filters.http.router"}},
							})
						if err != nil {
							panic(err)
						}
						return any
					}(),
				},
			}},
		}},
	}
}

func HTTPListenerWithRdsV3(listenerName, routeName string) (string, *envoy_config_listener_v3.Listener) {
	return listenerName, &envoy_config_listener_v3.Listener{
		Name: listenerName,
		Address: &envoy_config_core_v3.Address{
			Address: &envoy_config_core_v3.Address_SocketAddress{
				SocketAddress: &envoy_config_core_v3.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
						PortValue: PodPort,
					}}}},
		FilterChains: []*envoy_config_listener_v3.FilterChain{{
			Filters: []*envoy_config_listener_v3.Filter{{
				Name: "envoy.filters.network.http_connection_manager",
				ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
					TypedConfig: func() *any.Any {
						any, err := ptypes.MarshalAny(
							&http_connection_manager_v3.HttpConnectionManager{
								StatPrefix: listenerName,
								RouteSpecifier: &http_connection_manager_v3.HttpConnectionManager_Rds{
									Rds: &http_connection_manager_v3.Rds{
										ConfigSource: &envoy_config_core_v3.ConfigSource{
											ConfigSourceSpecifier: &envoy_config_core_v3.ConfigSource_Ads{
												Ads: &envoy_config_core_v3.AggregatedConfigSource{},
											},
											ResourceApiVersion: envoy_config_core_v3.ApiVersion_V3,
										},
										RouteConfigName: routeName,
									},
								},
								HttpFilters: []*http_connection_manager_v3.HttpFilter{{Name: "envoy.filters.http.router"}},
							})
						if err != nil {
							panic(err)
						}
						return any
					}(),
				},
			}},
		}},
	}
}

func ProxyPassRouteV2(routeName, clusterName string) (string, *envoy_api_v2.RouteConfiguration) {
	return routeName, &envoy_api_v2.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
			Name:    routeName,
			Domains: []string{"*"},
			Routes: []*envoy_api_v2_route.Route{{
				Match: &envoy_api_v2_route.RouteMatch{
					PathSpecifier: &envoy_api_v2_route.RouteMatch_Prefix{Prefix: "/"}},
				Action: &envoy_api_v2_route.Route_Route{
					Route: &envoy_api_v2_route.RouteAction{
						ClusterSpecifier: &envoy_api_v2_route.RouteAction_Cluster{Cluster: clusterName},
					},
				},
			}},
		}},
	}
}

func ProxyPassRouteV3(routeName, clusterName string) (string, *envoy_config_route_v3.RouteConfiguration) {
	return routeName, &envoy_config_route_v3.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
			Name:    routeName,
			Domains: []string{"*"},
			Routes: []*envoy_config_route_v3.Route{{
				Match: &envoy_config_route_v3.RouteMatch{
					PathSpecifier: &envoy_config_route_v3.RouteMatch_Prefix{Prefix: "/"}},
				Action: &envoy_config_route_v3.Route_Route{
					Route: &envoy_config_route_v3.RouteAction{
						ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{Cluster: clusterName},
					},
				},
			}},
		}},
	}
}

func DirectResponseRouteV2(routeName, msg string) (string, *envoy_api_v2.RouteConfiguration) {
	return routeName, &envoy_api_v2.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []*envoy_api_v2_route.VirtualHost{{
			Name:    routeName,
			Domains: []string{"*"},
			Routes: []*envoy_api_v2_route.Route{{
				Match: &envoy_api_v2_route.RouteMatch{
					PathSpecifier: &envoy_api_v2_route.RouteMatch_Prefix{Prefix: "/"}},
				Action: &envoy_api_v2_route.Route_DirectResponse{
					DirectResponse: &envoy_api_v2_route.DirectResponseAction{
						Status: 200,
						Body: &envoy_api_v2_core.DataSource{
							Specifier: &envoy_api_v2_core.DataSource_InlineString{InlineString: msg},
						},
					}},
			}},
		}},
	}
}

func DirectResponseRouteV3(routeName, msg string) (string, *envoy_config_route_v3.RouteConfiguration) {
	return routeName, &envoy_config_route_v3.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []*envoy_config_route_v3.VirtualHost{{
			Name:    routeName,
			Domains: []string{"*"},
			Routes: []*envoy_config_route_v3.Route{{
				Match: &envoy_config_route_v3.RouteMatch{
					PathSpecifier: &envoy_config_route_v3.RouteMatch_Prefix{Prefix: "/"}},
				Action: &envoy_config_route_v3.Route_DirectResponse{
					DirectResponse: &envoy_config_route_v3.DirectResponseAction{
						Status: 200,
						Body: &envoy_config_core_v3.DataSource{
							Specifier: &envoy_config_core_v3.DataSource_InlineString{InlineString: msg},
						},
					}},
			}},
		}},
	}
}

func EndpointV2(clusterName, host string, port uint32) (string, *envoy_api_v2.ClusterLoadAssignment) {
	return clusterName, &envoy_api_v2.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*envoy_api_v2_endpoint.LocalityLbEndpoints{
			{
				LbEndpoints: []*envoy_api_v2_endpoint.LbEndpoint{
					{
						HostIdentifier: &envoy_api_v2_endpoint.LbEndpoint_Endpoint{
							Endpoint: &envoy_api_v2_endpoint.Endpoint{
								Address: &envoy_api_v2_core.Address{
									Address: &envoy_api_v2_core.Address_SocketAddress{
										SocketAddress: &envoy_api_v2_core.SocketAddress{
											Address: host,
											PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
												PortValue: port,
											}}}}}}}}}},
	}
}

func EndpointV3(clusterName, host string, port uint32) (string, *envoy_config_endpoint_v3.ClusterLoadAssignment) {
	return clusterName, &envoy_config_endpoint_v3.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*envoy_config_endpoint_v3.LocalityLbEndpoints{
			{
				LbEndpoints: []*envoy_config_endpoint_v3.LbEndpoint{
					{
						HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
							Endpoint: &envoy_config_endpoint_v3.Endpoint{
								Address: &envoy_config_core_v3.Address{
									Address: &envoy_config_core_v3.Address_SocketAddress{
										SocketAddress: &envoy_config_core_v3.SocketAddress{
											Address: host,
											PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
												PortValue: port,
											}}}}}}}}}},
	}
}

func ClusterWithEdsV2(clusterName string) (string, *envoy_api_v2.Cluster) {
	return clusterName, &envoy_api_v2.Cluster{
		Name:           clusterName,
		ConnectTimeout: ptypes.DurationProto(10 * time.Millisecond),
		ClusterDiscoveryType: &envoy_api_v2.Cluster_Type{
			Type: envoy_api_v2.Cluster_EDS,
		},
		EdsClusterConfig: &envoy_api_v2.Cluster_EdsClusterConfig{
			EdsConfig: &envoy_api_v2_core.ConfigSource{
				ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
					Ads: &envoy_api_v2_core.AggregatedConfigSource{},
				},
				ResourceApiVersion: envoy_api_v2_core.ApiVersion_V2,
			}},
		LbPolicy: envoy_api_v2.Cluster_ROUND_ROBIN,
	}
}

func ClusterWithEdsV3(clusterName string) (string, *envoy_config_cluster_v3.Cluster) {
	return clusterName, &envoy_config_cluster_v3.Cluster{
		Name:           clusterName,
		ConnectTimeout: ptypes.DurationProto(10 * time.Millisecond),
		ClusterDiscoveryType: &envoy_config_cluster_v3.Cluster_Type{
			Type: envoy_config_cluster_v3.Cluster_EDS,
		},
		LbPolicy: envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
		EdsClusterConfig: &envoy_config_cluster_v3.Cluster_EdsClusterConfig{
			EdsConfig: &envoy_config_core_v3.ConfigSource{
				ConfigSourceSpecifier: &envoy_config_core_v3.ConfigSource_Ads{
					Ads: &envoy_config_core_v3.AggregatedConfigSource{},
				},
				ResourceApiVersion: envoy_config_core_v3.ApiVersion_V3,
			}},
	}
}
