package discoveryservice

import (
	"bytes"
	"context"
	"fmt"
	"time"

	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/webhook"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	envoy_config_bootstrap_v2 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileEnabledNamespaces is in charge of keep the resources that envoy sidecars require available in all
// the active namespaces:
//     - a Secret holding a client certificate for mTLS with the DiscoveryService
//     - a ConfigMap with the envoy bootstrap configuration to allow envoy sidecars to talk to the DiscoveryService
//     - keeps the Namespace object marked as owned by the marin3r instance
func (r *ReconcileDiscoveryService) reconcileEnabledNamespaces(ctx context.Context) (reconcile.Result, error) {
	var err error
	// Reconcile each namespace in the list of enabled namespaces
	for _, ns := range r.ds.Spec.EnabledNamespaces {
		err = r.reconcileEnabledNamespace(ctx, ns)
		// Keep going even if an error is returned
	}

	if err != nil {
		// TODO: this will surface just the last error, change it so if several errors
		// occur in different namespaces all of them are reported to the caller
		return reconcile.Result{}, fmt.Errorf("Failed reconciling enabled namespaces: %s", err)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileDiscoveryService) reconcileEnabledNamespace(ctx context.Context, namespace string) error {

	r.logger.V(1).Info("Reconciling enabled Namespace", "Namespace", namespace)

	ns := &corev1.Namespace{}
	err := r.client.Get(ctx, types.NamespacedName{Name: namespace}, ns)

	if err != nil {
		// Namespace should exist
		return err
	}

	owner, err := isOwner(r.ds, ns)
	if err != nil {
		return err
	}

	if !owner || !hasEnabledLabel(ns) {

		patch := client.MergeFrom(ns.DeepCopy())

		// Init label's map
		if ns.GetLabels() == nil {
			ns.SetLabels(map[string]string{})
		}

		// Set namespace labels
		ns.ObjectMeta.Labels[operatorv1alpha1.DiscoveryServiceEnabledKey] = operatorv1alpha1.DiscoveryServiceEnabledValue
		ns.ObjectMeta.Labels[operatorv1alpha1.DiscoveryServiceLabelKey] = r.ds.GetName()

		if err := r.client.Patch(ctx, ns, patch); err != nil {
			return err
		}
		r.logger.Info("Patched Namespace", "Namespace", namespace)
	}

	if err := r.reconcileClientCertificate(ctx, namespace); err != nil {
		return err
	}

	if err := r.reconcileBootstrapConfigMap(ctx, namespace); err != nil {
		return err
	}

	return nil
}

func isOwner(owner metav1.Object, object metav1.Object) (bool, error) {

	value, ok := object.GetLabels()[operatorv1alpha1.DiscoveryServiceLabelKey]
	if ok {
		if value == owner.GetName() {
			return true, nil
		}
		return false, fmt.Errorf("Namespace already onwed by %s", value)
	}

	return false, nil
}

func hasEnabledLabel(object metav1.Object) bool {

	value, ok := object.GetLabels()[operatorv1alpha1.DiscoveryServiceEnabledKey]
	if ok && value == operatorv1alpha1.DiscoveryServiceEnabledValue {
		return true
	}

	return false
}

func (r *ReconcileDiscoveryService) reconcileClientCertificate(ctx context.Context, namespace string) error {
	r.logger.V(1).Info("Reconciling client certificate", "Namespace", namespace)
	existent := &operatorv1alpha1.DiscoveryServiceCertificate{}
	err := r.client.Get(ctx, types.NamespacedName{Name: webhook.DefaultClientCertificate, Namespace: namespace}, existent)

	if err != nil {
		if errors.IsNotFound(err) {
			existent = r.getClientCertObject(namespace)
			if err := controllerutil.SetControllerReference(r.ds, existent, r.scheme); err != nil {
				return err
			}
			if err := r.client.Create(ctx, existent); err != nil {
				return err
			}
			r.logger.Info("Created client certificate", "Namespace", namespace)
			return nil
		}
		return err
	}

	// Certificates are not currently reconciled

	return nil
}

func (r *ReconcileDiscoveryService) reconcileBootstrapConfigMap(ctx context.Context, namespace string) error {
	r.logger.V(1).Info("Reconciling bootstrap ConfigMap", "Namespace", namespace)
	existent := &corev1.ConfigMap{}
	err := r.client.Get(ctx, types.NamespacedName{Name: webhook.DefaultBootstrapConfigMap, Namespace: namespace}, existent)

	if err != nil {
		if errors.IsNotFound(err) {
			existent, err := r.getBootstrapConfigMapObject(namespace)
			if err != nil {
				return err
			}
			if err := controllerutil.SetControllerReference(r.ds, existent, r.scheme); err != nil {
				return err
			}
			if err := r.client.Create(ctx, existent); err != nil {
				return err
			}
			r.logger.Info("Created bootstrap ConfigMap", "Namespace", namespace)
			return nil
		}
		return err
	}

	// Bootstrap ConfigMap are not currently reconciled

	return nil
}

func (r *ReconcileDiscoveryService) getClientCertObject(namespace string) *operatorv1alpha1.DiscoveryServiceCertificate {
	return &operatorv1alpha1.DiscoveryServiceCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhook.DefaultClientCertificate,
			Namespace: namespace,
		},
		Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
			CommonName: OwnedObjectName(r.ds),
			ValidFor:   clientValidFor,
			Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
				CertManager: &operatorv1alpha1.CertManagerConfig{
					ClusterIssuer: r.getClusterIssuerName(),
				},
			},
			SecretRef: corev1.SecretReference{
				Name:      webhook.DefaultClientCertificate,
				Namespace: namespace,
			},
		},
	}
}

func (r *ReconcileDiscoveryService) getBootstrapConfigMapObject(namespace string) (*corev1.ConfigMap, error) {

	config, err := getEnvoyBootstrapConfig(getDiscoveryServiceHost(r.ds), getDiscoveryServicePort())
	if err != nil {
		return nil, err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhook.DefaultBootstrapConfigMap,
			Namespace: namespace,
		},
		Data: map[string]string{
			"config.json": config,
		},
	}

	return cm, nil
}

func getEnvoyBootstrapConfig(host string, port uint32) (string, error) {

	tlsContext := &envoy_api_v2_auth.UpstreamTlsContext{
		CommonTlsContext: &envoy_api_v2_auth.CommonTlsContext{
			TlsCertificates: []*envoy_api_v2_auth.TlsCertificate{
				{
					CertificateChain: &envoy_api_v2_core.DataSource{
						Specifier: &envoy_api_v2_core.DataSource_Filename{
							Filename: fmt.Sprintf("%s/%s", webhook.DefaultEnvoyTLSBasePath, "tls.crt"),
						},
					},
					PrivateKey: &envoy_api_v2_core.DataSource{
						Specifier: &envoy_api_v2_core.DataSource_Filename{
							Filename: fmt.Sprintf("%s/%s", webhook.DefaultEnvoyTLSBasePath, "tls.key"),
						}},
				},
			},
		},
	}

	serializedTLSContext, err := proto.Marshal(tlsContext)
	if err != nil {
		return "", err
	}

	cfg := &envoy_config_bootstrap_v2.Bootstrap{
		DynamicResources: &envoy_config_bootstrap_v2.Bootstrap_DynamicResources{
			AdsConfig: &envoy_api_v2_core.ApiConfigSource{
				ApiType:             envoy_api_v2_core.ApiConfigSource_GRPC,
				TransportApiVersion: envoy_api_v2_core.ApiVersion_V2,
				GrpcServices: []*envoy_api_v2_core.GrpcService{
					{
						TargetSpecifier: &envoy_api_v2_core.GrpcService_EnvoyGrpc_{
							EnvoyGrpc: &envoy_api_v2_core.GrpcService_EnvoyGrpc{
								ClusterName: "ads_cluster",
							},
						},
					},
				},
			},
			CdsConfig: &envoy_api_v2_core.ConfigSource{
				ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
					Ads: &envoy_api_v2_core.AggregatedConfigSource{},
				},
			},
			LdsConfig: &envoy_api_v2_core.ConfigSource{
				ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
					Ads: &envoy_api_v2_core.AggregatedConfigSource{},
				},
			},
		},
		StaticResources: &envoy_config_bootstrap_v2.Bootstrap_StaticResources{
			Clusters: []*envoy_api_v2.Cluster{
				{
					Name:           "ads_cluster",
					ConnectTimeout: ptypes.DurationProto(1 * time.Second),
					ClusterDiscoveryType: &envoy_api_v2.Cluster_Type{
						Type: envoy_api_v2.Cluster_STRICT_DNS,
					},
					Http2ProtocolOptions: &envoy_api_v2_core.Http2ProtocolOptions{},
					LoadAssignment: &envoy_api_v2.ClusterLoadAssignment{
						ClusterName: "ads_cluster",
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
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					TransportSocket: &envoy_api_v2_core.TransportSocket{
						Name: wellknown.TransportSocketTls,
						ConfigType: &envoy_api_v2_core.TransportSocket_TypedConfig{
							TypedConfig: &any.Any{
								TypeUrl: "type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext",
								Value:   serializedTLSContext,
							},
						},
					},
				},
			},
		},
	}

	m := jsonpb.Marshaler{}

	json := bytes.NewBuffer([]byte{})
	err = m.Marshal(json, cfg)
	if err != nil {
		return "", err
	}

	// yaml, err := yaml.JSONToYAML(json.Bytes())
	// if err != nil {
	// 	return "", err
	// }

	return string(json.Bytes()), nil
}

func getDiscoveryServiceHost(ds *operatorv1alpha1.DiscoveryService) string {
	return fmt.Sprintf("%s.%s.%s", OwnedObjectName(ds), OwnedObjectNamespace(ds), "svc")
}

func getDiscoveryServicePort() uint32 {
	return uint32(18000)
}
