package reconcilers

import (
	"context"
	"reflect"
	"testing"
	"time"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/envoy"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewBootstrapConfigReconciler(t *testing.T) {
	type args struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		scheme *runtime.Scheme
		eb     *envoyv1alpha1.EnvoyBootstrap
	}
	tests := []struct {
		name string
		args args
		want BootstrapConfigReconciler
	}{
		{
			name: "Returns a reconciler",
			args: args{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClient(),
				scheme: &runtime.Scheme{},
				eb:     &envoyv1alpha1.EnvoyBootstrap{},
			},
			want: BootstrapConfigReconciler{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClient(),
				scheme: &runtime.Scheme{},
				eb:     &envoyv1alpha1.EnvoyBootstrap{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewBootstrapConfigReconciler(tt.args.ctx, tt.args.logger, tt.args.client, tt.args.scheme, tt.args.eb); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBootstrapConfigReconciler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBootstrapConfigReconciler_Reconcile(t *testing.T) {
	var s *runtime.Scheme = scheme.Scheme
	s.AddKnownTypes(envoyv1alpha1.GroupVersion, &envoyv1alpha1.EnvoyBootstrap{})
	s.AddKnownTypes(operatorv1alpha1.GroupVersion, &operatorv1alpha1.DiscoveryService{})

	type args struct {
		envoyAPI envoy.APIVersion
	}
	tests := []struct {
		name    string
		r       *BootstrapConfigReconciler
		args    args
		want    ctrl.Result
		wantErr bool
		wantCM  *corev1.ConfigMap
	}{
		{
			name: "Creates a ConfigMap for v2",
			r: &BootstrapConfigReconciler{
				ctx:    context.TODO(),
				logger: ctrl.Log.WithName("test"),
				client: fake.NewFakeClientWithScheme(
					s,
					&operatorv1alpha1.DiscoveryService{
						ObjectMeta: v1.ObjectMeta{Name: "ds"},
						Spec: operatorv1alpha1.DiscoveryServiceSpec{
							DiscoveryServiceNamespace: "default",
							EnabledNamespaces:         []string{"default"},
							Image:                     "xxx",
							Debug:                     false,
						},
					},
				),
				scheme: s,
				eb: &envoyv1alpha1.EnvoyBootstrap{
					ObjectMeta: v1.ObjectMeta{Name: "eb", Namespace: "default"},
					Spec: envoyv1alpha1.EnvoyBootstrapSpec{
						DiscoveryService: "ds",
						ClientCertificate: &envoyv1alpha1.ClientCertificate{
							Directory:  "/tls",
							SecretName: "client-certificate",
							Duration: metav1.Duration{
								Duration: func() time.Duration { d, _ := time.ParseDuration("24h"); return d }(),
							},
						},
						EnvoyStaticConfig: &envoyv1alpha1.EnvoyStaticConfig{
							ConfigMapNameV2:       "cm-v2",
							ConfigMapNameV3:       "cm-v3",
							ConfigFile:            "config.json",
							ResourcesDir:          "/resdir",
							RtdsLayerResourceName: "runtime",
							AdminBindAddress:      "127.0.0.1:1000",
							AdminAccessLogPath:    "/dev/null",
						},
					},
				},
			},
			args:    args{envoyAPI: envoy.APIv2},
			want:    ctrl.Result{},
			wantErr: false,
			wantCM: &corev1.ConfigMap{
				ObjectMeta: v1.ObjectMeta{Name: "cm-v2", Namespace: "default"},
				Data: map[string]string{
					"config.json":                     `{"static_resources":{"clusters":[{"name":"xds_cluster","type":"STRICT_DNS","connect_timeout":"1s","load_assignment":{"cluster_name":"xds_cluster","endpoints":[{"lb_endpoints":[{"endpoint":{"address":{"socket_address":{"address":"discoveryservice-ds.default.svc","port_value":18000}}}}]}]},"http2_protocol_options":{},"transport_socket":{"name":"envoy.transport_sockets.tls","typed_config":{"@type":"type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext","common_tls_context":{"tls_certificate_sds_secret_configs":[{"sds_config":{"path":"/resdir/tls_certificate_sds_secret.json"}}]}}}}]},"dynamic_resources":{"lds_config":{"ads":{},"resource_api_version":"V2"},"cds_config":{"ads":{},"resource_api_version":"V2"},"ads_config":{"api_type":"GRPC","transport_api_version":"V2","grpc_services":[{"envoy_grpc":{"cluster_name":"xds_cluster"}}]}},"layered_runtime":{"layers":[{"name":"runtime","rtds_layer":{"name":"runtime","rtds_config":{"ads":{},"resource_api_version":"V2"}}}]},"admin":{"access_log_path":"/dev/null","address":{"socket_address":{"address":"127.0.0.1","port_value":1000}}}}`,
					"tls_certificate_sds_secret.json": `{"resources":[{"@type":"type.googleapis.com/envoy.api.v2.auth.Secret","tls_certificate":{"certificate_chain":{"filename":"/tls/tls.key"},"private_key":{"filename":"/tls/tls.crt"}}}]}`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.Reconcile(tt.args.envoyAPI)
			if (err != nil) != tt.wantErr {
				t.Errorf("BootstrapConfigReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BootstrapConfigReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
			gotCM := &corev1.ConfigMap{}
			_ = tt.r.client.Get(tt.r.ctx, types.NamespacedName{Name: tt.wantCM.GetName(), Namespace: tt.wantCM.GetNamespace()}, gotCM)
			if !tt.wantErr && !equality.Semantic.DeepEqual(tt.wantCM.Data, gotCM.Data) {
				t.Errorf("BootstrapConfigReconciler.Reconcile() ConfigMap.Data = %v, want %v", gotCM.Data, tt.wantCM.Data)
				return
			}
		})
	}
}

func Test_parseBindAddress(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   uint32
		wantErr bool
	}{
		{
			name:    "Correcly parses an address",
			args:    args{address: "127.0.0.1:1000"},
			want:    "127.0.0.1",
			want1:   1000,
			wantErr: false,
		},
		{
			name:    "Invalid IP address",
			args:    args{address: "127.x.x.x:1000"},
			want:    "",
			want1:   0,
			wantErr: true,
		},
		{
			name:    "Invalid port",
			args:    args{address: "127.0.0.1:invalid"},
			want:    "",
			want1:   0,
			wantErr: true,
		},
		{
			name:    "Invalid format",
			args:    args{address: "127.0.0.1-1000"},
			want:    "",
			want1:   0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseBindAddress(tt.args.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBindAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseBindAddress() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("parseBindAddress() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
