package e2e

import (
	"fmt"
	"time"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	"github.com/3scale-ops/marin3r/pkg/envoy/container/defaults"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	"github.com/3scale-ops/marin3r/pkg/util/pki"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

const (
	DeploymentLabelKey   string = "app"
	DeploymentLabelValue string = "nginx"
	PodLabelKey          string = "app"
	PodLabelValue        string = "testPod"
)

func GeneratePod(key types.NamespacedName, nodeID, envoyAPI, envoyVersion, discoveryService string) *corev1.Pod {

	initContainers := []corev1.Container{{
		Name:  "init-manager",
		Image: "quay.io/3scale/marin3r:test",
		Args: []string{
			"init-manager",
			"--api-version", envoyAPI,
			"--xdss-host", fmt.Sprintf("marin3r-%s.%s.svc", discoveryService, key.Namespace),
			"--envoy-image", fmt.Sprintf("%s:%s", defaults.ImageRepo, envoyVersion),
		},
		Env: []corev1.EnvVar{
			{Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
			{Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
			{Name: "HOST_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "config-volume",
			ReadOnly:  false,
			MountPath: "/etc/envoy/bootstrap",
		}},
	}}

	containers := []corev1.Container{{
		Name:    "envoy",
		Image:   fmt.Sprintf("%s:%s", defaults.ImageRepo, envoyVersion),
		Command: []string{"envoy"},
		Args: []string{
			"-c", "/etc/envoy/bootstrap/config.json",
			"--service-node", nodeID,
			"--service-cluster", nodeID,
			"--component-log-level", "config:debug",
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "tls-volume", ReadOnly: true, MountPath: "/etc/envoy/tls/client"},
			{Name: "config-volume", ReadOnly: true, MountPath: "/etc/envoy/bootstrap"},
		},
		ReadinessProbe: &corev1.Probe{
			Handler:             corev1.Handler{HTTPGet: &corev1.HTTPGetAction{Path: "/ready", Port: intstr.IntOrString{IntVal: 9901}}},
			InitialDelaySeconds: 15, TimeoutSeconds: 1, PeriodSeconds: 5, SuccessThreshold: 1, FailureThreshold: 1,
		},
	}}

	volumes := []corev1.Volume{
		{Name: "tls-volume", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "envoy-sidecar-client-cert"}}},
		{Name: "config-volume", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
			Labels:    map[string]string{PodLabelKey: PodLabelValue},
		},
		Spec: corev1.PodSpec{
			Volumes:        volumes,
			InitContainers: initContainers,
			Containers:     containers,
		},
	}
}

func GenerateDeployment(key types.NamespacedName) *appsv1.Deployment {
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
						DeploymentLabelKey: DeploymentLabelValue,
					},
					Annotations: map[string]string{},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "nginx",
						Image: "nginxdemos/hello:plain-text",
						Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 80}},
					}},
				},
			},
		},
	}
}

func GenerateHeadlessService(key types.NamespacedName) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "None",
			Ports:     []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromString("http")}},
			Selector: map[string]string{
				DeploymentLabelKey: DeploymentLabelValue,
			},
		},
	}
}

func GenerateDeploymentWithInjection(key types.NamespacedName, nodeID, envoyAPI, envoyVersion string, envoyPort uint32) *appsv1.Deployment {
	dep := GenerateDeployment(key)
	dep.Spec.Template.ObjectMeta.Labels["marin3r.3scale.net/status"] = "enabled"
	dep.Spec.Template.ObjectMeta.Annotations["marin3r.3scale.net/node-id"] = nodeID
	dep.Spec.Template.ObjectMeta.Annotations["marin3r.3scale.net/envoy-extra-args"] = "--component-log-level config:debug"
	dep.Spec.Template.ObjectMeta.Annotations["marin3r.3scale.net/ports"] = fmt.Sprintf("envoy-http:%v", envoyPort)
	dep.Spec.Template.ObjectMeta.Annotations["marin3r.3scale.net/envoy-api-version"] = envoyAPI
	dep.Spec.Template.ObjectMeta.Annotations["marin3r.3scale.net/envoy-image"] = fmt.Sprintf("%s:%s", defaults.ImageRepo, envoyVersion)
	dep.Spec.Template.ObjectMeta.Annotations["marin3r.3scale.net/init-manager.image"] = "quay.io/3scale/marin3r:test"

	return dep
}

func GenerateTLSSecret(k8skey types.NamespacedName, commonName, duration string) (*corev1.Secret, error) {

	tDuration, err := time.ParseDuration(duration)
	if err != nil {
		return nil, err
	}

	crt, key, err := pki.GenerateCertificate(nil, nil, commonName, tDuration, true, false, commonName)
	if err != nil {
		return nil, err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: k8skey.Name, Namespace: k8skey.Namespace},
		Type:       corev1.SecretTypeTLS,
		Data:       map[string][]byte{"tls.crt": crt, "tls.key": key},
	}
	return secret, err
}

func GenerateEnvoyConfig(key types.NamespacedName, nodeID string, envoyAPI envoy.APIVersion,
	endpointsGenFn, clustersGenFn, routesGenFn, listenersGenFn func() map[string]envoy.Resource, secrets map[string]string) *marin3rv1alpha1.EnvoyConfig {
	m := envoy_serializer.NewResourceMarshaller(envoy_serializer.JSON, envoyAPI)

	return &marin3rv1alpha1.EnvoyConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: marin3rv1alpha1.EnvoyConfigSpec{
			EnvoyAPI: pointer.StringPtr(envoyAPI.String()),
			NodeID:   nodeID,
			EnvoyResources: &marin3rv1alpha1.EnvoyResources{
				Endpoints: func() []marin3rv1alpha1.EnvoyResource {
					endpoints := []marin3rv1alpha1.EnvoyResource{}
					for name, resource := range endpointsGenFn() {
						json, err := m.Marshal(resource)
						if err != nil {
							panic(err)
						}
						endpoints = append(endpoints, marin3rv1alpha1.EnvoyResource{Name: name, Value: json})
					}
					return endpoints
				}(),
				Clusters: func() []marin3rv1alpha1.EnvoyResource {
					clusters := []marin3rv1alpha1.EnvoyResource{}
					for name, resource := range clustersGenFn() {
						json, err := m.Marshal(resource)
						if err != nil {
							panic(err)
						}
						clusters = append(clusters, marin3rv1alpha1.EnvoyResource{Name: name, Value: json})
					}
					return clusters
				}(),
				Routes: func() []marin3rv1alpha1.EnvoyResource {
					routes := []marin3rv1alpha1.EnvoyResource{}
					for name, resource := range routesGenFn() {
						json, err := m.Marshal(resource)
						if err != nil {
							panic(err)
						}
						routes = append(routes, marin3rv1alpha1.EnvoyResource{Name: name, Value: json})
					}
					return routes
				}(),
				Listeners: func() []marin3rv1alpha1.EnvoyResource {
					listeners := []marin3rv1alpha1.EnvoyResource{}
					for name, resource := range listenersGenFn() {
						json, err := m.Marshal(resource)
						if err != nil {
							panic(err)
						}
						listeners = append(listeners, marin3rv1alpha1.EnvoyResource{Name: name, Value: json})
					}
					return listeners
				}(),
				Secrets: func() []marin3rv1alpha1.EnvoySecretResource {
					s := []marin3rv1alpha1.EnvoySecretResource{}
					for k, v := range secrets {
						s = append(s, marin3rv1alpha1.EnvoySecretResource{
							Name: k,
							Ref: &corev1.SecretReference{
								Name:      v,
								Namespace: key.Namespace,
							},
						})
					}
					return s
				}(),
			},
		},
	}

}

func GetAddressV3(host string, port uint32) *envoy_config_core_v3.Address {
	return &envoy_config_core_v3.Address{
		Address: &envoy_config_core_v3.Address_SocketAddress{
			SocketAddress: &envoy_config_core_v3.SocketAddress{
				Address: host,
				PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
					PortValue: port,
				}}}}
}

func TransportSocketV3(secretName string) *envoy_config_core_v3.TransportSocket {
	return &envoy_config_core_v3.TransportSocket{
		Name: "envoy.transport_sockets.tls",
		ConfigType: &envoy_config_core_v3.TransportSocket_TypedConfig{
			TypedConfig: func() *anypb.Any {
				any, err := anypb.New(&envoy_extensions_transport_sockets_tls_v3.DownstreamTlsContext{
					CommonTlsContext: &envoy_extensions_transport_sockets_tls_v3.CommonTlsContext{
						TlsCertificateSdsSecretConfigs: []*envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig{
							{
								Name: secretName,
								SdsConfig: &envoy_config_core_v3.ConfigSource{
									ConfigSourceSpecifier: &envoy_config_core_v3.ConfigSource_Ads{
										Ads: &envoy_config_core_v3.AggregatedConfigSource{},
									},
									ResourceApiVersion: envoy_config_core_v3.ApiVersion_V3,
								},
							},
						},
					},
				})
				if err != nil {
					panic(err)
				}
				return any
			}(),
		},
	}
}

func HTTPListenerWithRdsV3(listenerName, routeName string, address, transportSocket proto.Message) (string, *envoy_config_listener_v3.Listener) {
	return listenerName, &envoy_config_listener_v3.Listener{
		Name:    listenerName,
		Address: address.(*envoy_config_core_v3.Address),
		FilterChains: []*envoy_config_listener_v3.FilterChain{{
			Filters: []*envoy_config_listener_v3.Filter{{
				Name: "envoy.filters.network.http_connection_manager",
				ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
					TypedConfig: func() *anypb.Any {
						any, err := anypb.New(
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
			TransportSocket: func() *envoy_config_core_v3.TransportSocket {
				if transportSocket != nil {
					return transportSocket.(*envoy_config_core_v3.TransportSocket)
				}
				return nil
			}(),
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

func ClusterWithEdsV3(clusterName string) (string, *envoy_config_cluster_v3.Cluster) {
	return clusterName, &envoy_config_cluster_v3.Cluster{
		Name:           clusterName,
		ConnectTimeout: durationpb.New(10 * time.Millisecond),
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

func ClusterWithStrictDNSV3(clusterName, host string, port uint32) (string, *envoy_config_cluster_v3.Cluster) {
	return clusterName, &envoy_config_cluster_v3.Cluster{
		Name:           clusterName,
		ConnectTimeout: durationpb.New(10 * time.Millisecond),
		ClusterDiscoveryType: &envoy_config_cluster_v3.Cluster_Type{
			Type: envoy_config_cluster_v3.Cluster_STRICT_DNS,
		},
		LbPolicy: envoy_config_cluster_v3.Cluster_ROUND_ROBIN,
		LoadAssignment: &envoy_config_endpoint_v3.ClusterLoadAssignment{
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
		},
	}
}
