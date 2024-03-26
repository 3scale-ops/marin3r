package v1alpha1

import (
	"fmt"

	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

// Blueprint is an enum of the supported blueprints for
// generated resources
type Blueprint string

const (
	// TlsCertificate
	TlsCertificate Blueprint = "tlsCertificate"
	// TlsValidationContext
	TlsValidationContext Blueprint = "validationContext"
)

const defaultBlueprint Blueprint = TlsCertificate

// Resource holds serialized representation of an envoy
// resource
type Resource struct {
	// Type is the type url for the protobuf message
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Enum=listener;route;scopedRoute;cluster;endpoint;secret;runtime;extensionConfig;
	Type envoy.Type `json:"type"`
	// Value is the protobufer message that configures the resource. The proto
	// must match the envoy configuration API v3 specification for the given resource
	// type (https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#resource-types)
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Value *runtime.RawExtension `json:"value,omitempty"`
	// The name of a Kubernetes Secret of type "kubernetes.io/tls"
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	GenerateFromTlsSecret *string `json:"generateFromTlsSecret,omitempty"`
	// The name of a Kubernetes Secret of type "Opaque". It will generate an
	// envoy "generic secret" proto.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	GenerateFromOpaqueSecret *SecretKeySelector `json:"generateFromOpaqueSecret,omitempty"`
	// Specifies a label selector to watch for EndpointSlices that will
	// be used to generate the endpoint resource
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	GenerateFromEndpointSlices *GenerateFromEndpointSlices `json:"generateFromEndpointSlices,omitempty"`
	// Blueprint specifies a template to generate a configuration proto. It is currently
	// only supported to generate secret configuration resources from k8s Secrets
	// +kubebuilder:validation:Enum=tlsCertificate;validationContext;
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Blueprint *Blueprint `json:"blueprint,omitempty"`
}

func (r *Resource) GetBlueprint() Blueprint {
	if r.Blueprint != nil {
		return *r.Blueprint
	}
	return defaultBlueprint
}

func (r *Resource) SecretRef() (string, error) {
	if r.Type != envoy.Secret {
		return "", fmt.Errorf("not a secret type")
	}
	if r.GenerateFromOpaqueSecret != nil {
		return r.GenerateFromOpaqueSecret.Name, nil
	} else if r.GenerateFromTlsSecret != nil {
		return *r.GenerateFromTlsSecret, nil
	}
	return "", fmt.Errorf("secret reference not set")
}

type SecretKeySelector struct {
	// The name of the secret in the pod's namespace to select from.
	Name string `json:"name"`
	// The key of the secret to select from.  Must be a valid secret key.
	Key string `json:"key"`
	// A unique name to refer to the name:key combination
	Alias string `json:"alias"`
}

type GenerateFromEndpointSlices struct {
	Selector    *metav1.LabelSelector `json:"selector"`
	ClusterName string                `json:"clusterName"`
	TargetPort  string                `json:"targetPort"`
}

// EnvoyResources holds each envoy api resource type
type EnvoyResources struct {
	// Endpoints is a list of the envoy ClusterLoadAssignment resource type.
	// API V3 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/endpoint/v3/endpoint.proto
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Endpoints []EnvoyResource `json:"endpoints,omitempty"`
	// Clusters is a list of the envoy Cluster resource type.
	// API V3 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/cluster/v3/cluster.proto
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Clusters []EnvoyResource `json:"clusters,omitempty"`
	// Routes is a list of the envoy Route resource type.
	// API V3 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/route/v3/route.proto
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Routes []EnvoyResource `json:"routes,omitempty"`
	// ScopedRoutes is a list of the envoy ScopeRoute resource type.
	// API V3 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/route/v3/scoped_route.proto
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ScopedRoutes []EnvoyResource `json:"scopedRoutes,omitempty"`
	// Listeners is a list of the envoy Listener resource type.
	// API V3 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/listener/v3/listener.proto
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Listeners []EnvoyResource `json:"listeners,omitempty"`
	// Runtimes is a list of the envoy Runtime resource type.
	// API V3 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/runtime/v3/rtds.proto
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Runtimes []EnvoyResource `json:"runtimes,omitempty"`
	// Secrets is a list of references to Kubernetes Secret objects.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Secrets []EnvoySecretResource `json:"secrets,omitempty"`
	// ExtensionConfigs is a list of the envoy ExtensionConfig resource type
	// API V3 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/extension.proto
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ExtensionConfigs []EnvoyResource `json:"extensionConfigs,omitempty"`
}

func (in *EnvoyResources) Resources(serialization envoy_serializer.Serialization) ([]Resource, error) {
	errList := []error{}
	resources := []Resource{}

	for _, deprecatedResource := range in.Endpoints {
		resource, err := (&deprecatedResource).Resource(envoy.Endpoint, serialization)
		if err != nil {
			errList = append(errList, err)
		}
		resources = append(resources, resource)
	}
	for _, deprecatedResource := range in.Clusters {
		r, err := (&deprecatedResource).Resource(envoy.Cluster, serialization)
		if err != nil {
			errList = append(errList, err)
		}
		resources = append(resources, r)
	}
	for _, deprecatedResource := range in.Routes {
		r, err := (&deprecatedResource).Resource(envoy.Route, serialization)
		if err != nil {
			errList = append(errList, err)
		}
		resources = append(resources, r)
	}
	for _, deprecatedResource := range in.ScopedRoutes {
		r, err := (&deprecatedResource).Resource(envoy.ScopedRoute, serialization)
		if err != nil {
			errList = append(errList, err)
		}
		resources = append(resources, r)
	}
	for _, deprecatedResource := range in.Listeners {
		r, err := (&deprecatedResource).Resource(envoy.Listener, serialization)
		if err != nil {
			errList = append(errList, err)
		}
		resources = append(resources, r)
	}
	for _, deprecatedResource := range in.Runtimes {
		r, err := (&deprecatedResource).Resource(envoy.Runtime, serialization)
		if err != nil {
			errList = append(errList, err)
		}
		resources = append(resources, r)
	}
	for _, deprecatedResource := range in.ExtensionConfigs {
		r, err := (&deprecatedResource).Resource(envoy.ExtensionConfig, serialization)
		if err != nil {
			errList = append(errList, err)
		}
		resources = append(resources, r)
	}
	for _, deprecatedResource := range in.Secrets {
		r := Resource{
			Type:                  envoy.Secret,
			GenerateFromTlsSecret: &deprecatedResource.Name,
			Blueprint:             pointer.New(TlsCertificate),
		}
		resources = append(resources, r)
	}

	if len(errList) > 0 {
		return nil, NewMultiError(errList)
	}

	return resources, nil
}

// EnvoyResource holds serialized representation of an envoy
// resource
type EnvoyResource struct {
	// Name of the envoy resource.
	// DEPRECATED: this field has no effect and will be removed in an
	// upcoming release. The name of the resources for discovery purposes
	// is included in the resource itself. Refer to the envoy API reference
	// to check how the name is specified for each resource type.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Name *string `json:"name"`
	// Value is the serialized representation of the envoy resource
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Value string `json:"value"`
}

// Transforms from the deprecated EnvoyResource struct to Resource
func (res *EnvoyResource) Resource(rType envoy.Type, serialization envoy_serializer.Serialization) (Resource, error) {
	var err error
	var b []byte

	if serialization == envoy_serializer.YAML {
		b, err = yaml.YAMLToJSON([]byte(res.Value))
		if err != nil {
			return Resource{}, err
		}
	} else {
		b = []byte(res.Value)
	}

	return Resource{
		Type: rType,
		Value: &runtime.RawExtension{
			Raw: b,
		},
	}, nil
}

// EnvoySecretResource holds a reference to a k8s Secret from where
// to take a secret from. Only Secrets within the same namespace can
// be referred.
type EnvoySecretResource struct {
	// Name of the envoy tslCerticate secret resource. The certificate will be fetched
	// from a Kubernetes Secrets of type 'kubernetes.io/tls' with this same name.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Name string `json:"name"`
	// DEPRECATED: this field is deprecated and it's value will be ignored. The 'name' of the
	// Kubernetes Secret must match the 'name' field.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:SecretReference"
	// +optional
	Ref *corev1.SecretReference `json:"ref,omitempty"`
}

func (esr *EnvoySecretResource) GetSecretKey(namespace string) types.NamespacedName {
	if esr.Ref != nil {
		return types.NamespacedName{Name: esr.Ref.Name, Namespace: namespace}
	}
	return types.NamespacedName{Name: esr.Name, Namespace: namespace}
}

func (esr *EnvoySecretResource) Validate(namespace string) error {
	if esr.Ref != nil {
		if esr.Ref.Name == "" {
			return fmt.Errorf("'%T.ref.name' cannot be empty", esr)
		}
		if esr.Ref.Namespace != "" && esr.Ref.Namespace != namespace {
			return fmt.Errorf("only Secrets from the same namespace '%s' can be referred", namespace)
		}
	}
	return nil
}
