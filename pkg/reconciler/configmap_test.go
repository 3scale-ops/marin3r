// Copyright 2020 rvazquez@redhat.com
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package reconciler

import (
	"bytes"
	"io"
	"reflect"
	"testing"
	"time"

	envoy_api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/roivaz/marin3r/pkg/cache"
	"github.com/roivaz/marin3r/pkg/util"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func fakeClientWithSecrets() *util.K8s {
	return util.FakeClusterClient(
		&corev1.SecretList{
			TypeMeta: metav1.TypeMeta{Kind: "SecretList", APIVersion: "v1"},
			Items: []corev1.Secret{
				{
					TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "secret1",
						Namespace:   "default",
						Annotations: map[string]string{certificateAnnotation: "cert1"},
					},
					Data: map[string][]byte{
						"tls.crt": []byte("xxxx"),
						"tls.key": []byte("xxxx"),
					},
				},
				{
					TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "secret2",
						Namespace:   "default",
						Annotations: map[string]string{certificateAnnotation: "cert2"},
					},
					Data: map[string][]byte{
						"tls.crt": []byte("yyyy"),
						"tls.key": []byte("yyyy"),
					},
				},
			},
		},
	)
}

func TestNewConfigMapReconcileJob(t *testing.T) {
	type args struct {
		nodeID    string
		eventType EventType
		configMap *corev1.ConfigMap
	}
	tests := []struct {
		name string
		args args
		want *ConfigMapReconcileJob
	}{
		{
			"Creates new job from 'Add' event",
			args{
				"node1",
				Add,
				&corev1.ConfigMap{
					TypeMeta:   v1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
					ObjectMeta: v1.ObjectMeta{Name: "cm"},
					Data:       map[string]string{"config.yaml": "content"},
				},
			},
			&ConfigMapReconcileJob{
				eventType: Add,
				nodeID:    "node1",
				configMap: &corev1.ConfigMap{
					TypeMeta:   v1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
					ObjectMeta: v1.ObjectMeta{Name: "cm"},
					Data:       map[string]string{"config.yaml": "content"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewConfigMapReconcileJob(tt.args.nodeID, tt.args.eventType, tt.args.configMap); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConfigMapReconcileJob() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigMapReconcileJob_Push(t *testing.T) {
	type args struct {
		queue chan ReconcileJob
	}
	tests := []struct {
		name string
		job  ConfigMapReconcileJob
		args args
	}{
		{
			"Pushes a job to the queue",
			ConfigMapReconcileJob{
				eventType: Update,
				nodeID:    "node1",
				configMap: &corev1.ConfigMap{
					TypeMeta: v1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
					ObjectMeta: v1.ObjectMeta{
						Name:        "cm",
						Annotations: map[string]string{"marin3r.3scale.net/node-id": "node1"},
					},
					Data: map[string]string{"config.yaml": "content"},
				},
			},
			args{make(chan ReconcileJob)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			received := false
			go func() {
				<-tt.args.queue
				received = true
			}()
			tt.job.Push(tt.args.queue)
			if !received {
				t.Error("ConfigMapReconcileJob.Push(): Job not received")
			}
		})
	}
}

func TestConfigMapReconcileJob_process(t *testing.T) {
	type args struct {
		c         cache.Cache
		clientset *util.K8s
		namespace string
		logger    *zap.SugaredLogger
	}
	type resource struct {
		name  string
		rtype xds_cache.ResponseType
		value xds_cache.Resource
	}
	type want struct {
		nodeIDs   []string
		resources []resource
	}
	tests := []struct {
		name    string
		job     ConfigMapReconcileJob
		args    args
		want    want
		wantErr bool
	}{
		{
			name: "Processes a ConfigMapReconcile job and generates expected resources in the cache",
			job: ConfigMapReconcileJob{
				Add,
				"node1",
				&corev1.ConfigMap{
					TypeMeta: v1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
					ObjectMeta: v1.ObjectMeta{
						Name:        "cm",
						Annotations: map[string]string{"marin3r.3scale.net/node-id": "node1"},
					},
					Data: map[string]string{
						"config.yaml": `
                            clusters:
                              - name: cluster1
                                connect_timeout: 2s
                                type: STRICT_DNS
                                lb_policy: ROUND_ROBIN
                                load_assignment:
                                  cluster_name: cluster1
                                  endpoints:
                                      - lb_endpoints:
                                          - endpoint:
                                              address:
                                                socket_address:
                                                    address: 127.0.0.1
                                                    port_value: 8080
                            listeners:
                              - name: listener1
                                address:
                                  socket_address:
                                    address: 0.0.0.0
                                    port_value: 8443
                            `,
					},
				},
			},
			args: args{
				func() cache.Cache { c := cache.NewCache(); c.NewNodeCache("node1"); return c }(),
				&util.K8s{},
				"default",
				func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
			want: want{
				[]string{"node1"},
				[]resource{
					{
						name:  "cluster1",
						rtype: cache.Cluster,
						value: &envoy_api.Cluster{
							Name:           "cluster1",
							ConnectTimeout: ptypes.DurationProto(2 * time.Second),
							ClusterDiscoveryType: &envoy_api.Cluster_Type{
								Type: envoy_api.Cluster_STRICT_DNS,
							},
							LbPolicy: envoy_api.Cluster_ROUND_ROBIN,
							LoadAssignment: &envoy_api.ClusterLoadAssignment{
								ClusterName: "cluster1",
								Endpoints: []*envoy_api_endpoint.LocalityLbEndpoints{
									{
										LbEndpoints: []*envoy_api_endpoint.LbEndpoint{
											{
												HostIdentifier: &envoy_api_endpoint.LbEndpoint_Endpoint{
													Endpoint: &envoy_api_endpoint.Endpoint{
														Address: &envoy_api_core.Address{
															Address: &envoy_api_core.Address_SocketAddress{
																SocketAddress: &envoy_api_core.SocketAddress{
																	Address: "127.0.0.1",
																	PortSpecifier: &envoy_api_core.SocketAddress_PortValue{
																		PortValue: 8080,
																	}}}}}}}}}}}},
					},
					{
						name:  "listener1",
						rtype: cache.Listener,
						value: &envoy_api.Listener{
							Name: "listener1",
							Address: &envoy_api_core.Address{
								Address: &envoy_api_core.Address_SocketAddress{
									SocketAddress: &envoy_api_core.SocketAddress{
										Address: "0.0.0.0",
										PortSpecifier: &envoy_api_core.SocketAddress_PortValue{
											PortValue: 8443,
										}}}}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Processes a ConfigMapReconcile job, creates a new cache for the node and generates expected resources in the cache",
			job: ConfigMapReconcileJob{
				Update,
				"node1",
				&corev1.ConfigMap{
					TypeMeta: v1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
					ObjectMeta: v1.ObjectMeta{
						Name:        "cm",
						Annotations: map[string]string{"marin3r.3scale.net/node-id": "node1"},
					},
					Data: map[string]string{
						"config.yaml": `
                            listeners:
                              - name: listener1
                                address:
                                  socket_address:
                                    address: 0.0.0.0
                                    port_value: 8443
                            `,
					},
				},
			},
			args: args{
				cache.NewCache(),
				fakeClientWithSecrets(),
				"default",
				func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
			want: want{
				[]string{"node1"},
				[]resource{
					{
						name:  "listener1",
						rtype: cache.Listener,
						value: &envoy_api.Listener{
							Name: "listener1",
							Address: &envoy_api_core.Address{
								Address: &envoy_api_core.Address_SocketAddress{
									SocketAddress: &envoy_api_core.SocketAddress{
										Address: "0.0.0.0",
										PortSpecifier: &envoy_api_core.SocketAddress_PortValue{
											PortValue: 8443,
										}}}}},
					},
					{
						"cert1",
						cache.Secret,
						&envoy_api_auth.Secret{
							Name: "cert1",
							Type: &envoy_api_auth.Secret_TlsCertificate{
								TlsCertificate: &envoy_api_auth.TlsCertificate{
									PrivateKey: &envoy_api_core.DataSource{
										Specifier: &envoy_api_core.DataSource_InlineBytes{InlineBytes: []byte("xxxx")},
									},
									CertificateChain: &envoy_api_core.DataSource{
										Specifier: &envoy_api_core.DataSource_InlineBytes{InlineBytes: []byte("xxxx")},
									}}}},
					},
					{
						"cert2",
						cache.Secret,
						&envoy_api_auth.Secret{
							Name: "cert2",
							Type: &envoy_api_auth.Secret_TlsCertificate{
								TlsCertificate: &envoy_api_auth.TlsCertificate{
									PrivateKey: &envoy_api_core.DataSource{
										Specifier: &envoy_api_core.DataSource_InlineBytes{InlineBytes: []byte("yyyy")},
									},
									CertificateChain: &envoy_api_core.DataSource{
										Specifier: &envoy_api_core.DataSource_InlineBytes{InlineBytes: []byte("yyyy")},
									}}}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Resources are not written to the cache if an error occurs in the node's cache bootstap process",
			job: ConfigMapReconcileJob{
				Add,
				"node1",
				&corev1.ConfigMap{
					TypeMeta: v1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
					ObjectMeta: v1.ObjectMeta{
						Name:        "cm",
						Annotations: map[string]string{"marin3r.3scale.net/node-id": "node1"},
					},
					Data: map[string]string{
						"config.yaml": `
                            listeners:
                              - name: listener1
                                address:
                                  socket_address:
                                    address: 0.0.0.0
                                    port_value: 8443
                            `,
					},
				},
			},
			args: args{
				cache.NewCache(),
				util.FakeErrorClusterClient(),
				"default",
				func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
			want: want{
				[]string{},
				[]resource{},
			},
			wantErr: true,
		},
		{
			name: "Resources are not written to the cache if an error occurs parsing the envoy config",
			job: ConfigMapReconcileJob{
				Add,
				"node1",
				&corev1.ConfigMap{
					TypeMeta: v1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
					ObjectMeta: v1.ObjectMeta{
						Name:        "cm",
						Annotations: map[string]string{"marin3r.3scale.net/node-id": "node1"},
					},
					Data: map[string]string{
						"config.yaml": `
                            listeners:
                              - name: listener1
                                address:
                                  wrong_field:
                                    address: 0.0.0.0
                                    port_value: 8443
                            `,
					},
				},
			},
			args: args{
				func() cache.Cache { c := cache.NewCache(); c.NewNodeCache("node1"); return c }(),
				&util.K8s{},
				"default",
				func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
			want: want{
				[]string{},
				[]resource{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := tt.job.process(tt.args.c, tt.args.clientset, tt.args.namespace, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigMapReconcileJob.process() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want.nodeIDs) {
				t.Errorf("ConfigMapReconcileJob.process() = %v, want %v", got, tt.want.nodeIDs)
			}

			// DeepEqual is not working for comparisons so we serialize
			// to json and then compare the resulting json strings
			marshaler := jsonpb.Marshaler{}
			for _, r := range tt.want.resources {

				var got bytes.Buffer
				_ = marshaler.Marshal(io.Writer(&got), tt.args.c.GetResource(tt.job.nodeID, r.name, r.rtype))
				t.Log(string(got.Bytes()))
				var res bytes.Buffer
				_ = marshaler.Marshal(io.Writer(&res), r.value)
				t.Log(string(got.Bytes()))

				if string(got.Bytes()) != string(res.Bytes()) {
					t.Errorf("ConfigMapReconcileJob.process() = '%v', want '%v'", string(got.Bytes()), string(res.Bytes()))
				}
			}
		})
	}
}

func Test_syncNodeSecrets(t *testing.T) {
	type args struct {
		client    *util.K8s
		namespace string
		nodeID    string
		c         cache.Cache
	}
	type resource struct {
		name  string
		rtype xds_cache.ResponseType
		value xds_cache.Resource
	}
	tests := []struct {
		name    string
		args    args
		want    []resource
		wantErr bool
	}{
		{
			"Populates the cache of secrets for the node",
			args{
				fakeClientWithSecrets(),
				"default",
				"node1",
				func() cache.Cache { c := cache.NewCache(); c.NewNodeCache("node1"); return c }(),
			},
			[]resource{
				{
					"cert1",
					cache.Secret,
					&envoy_api_auth.Secret{
						Name: "cert1",
						Type: &envoy_api_auth.Secret_TlsCertificate{
							TlsCertificate: &envoy_api_auth.TlsCertificate{
								PrivateKey: &envoy_api_core.DataSource{
									Specifier: &envoy_api_core.DataSource_InlineBytes{InlineBytes: []byte("xxxx")},
								},
								CertificateChain: &envoy_api_core.DataSource{
									Specifier: &envoy_api_core.DataSource_InlineBytes{InlineBytes: []byte("xxxx")},
								}}}},
				},
				{
					"cert2",
					cache.Secret,
					&envoy_api_auth.Secret{
						Name: "cert2",
						Type: &envoy_api_auth.Secret_TlsCertificate{
							TlsCertificate: &envoy_api_auth.TlsCertificate{
								PrivateKey: &envoy_api_core.DataSource{
									Specifier: &envoy_api_core.DataSource_InlineBytes{InlineBytes: []byte("yyyy")},
								},
								CertificateChain: &envoy_api_core.DataSource{
									Specifier: &envoy_api_core.DataSource_InlineBytes{InlineBytes: []byte("yyyy")},
								}}}},
				},
			},
			false,
		},
		{
			"Client error when listing secrets",
			args{
				util.FakeErrorClusterClient(),
				"default",
				"node1",
				cache.Cache{},
			},
			[]resource{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := syncNodeSecrets(tt.args.client, tt.args.namespace, tt.args.nodeID, tt.args.c); (err != nil) != tt.wantErr {
				t.Errorf("syncNodeSecrets() error = %v, wantErr %v", err, tt.wantErr)
			}

			// DeepEqual is not working for comparisons so we serialize
			// to json and then compare the resulting json strings
			marshaler := jsonpb.Marshaler{}
			for _, r := range tt.want {

				var got bytes.Buffer
				_ = marshaler.Marshal(io.Writer(&got), tt.args.c.GetResource(tt.args.nodeID, r.name, r.rtype))
				t.Log(string(got.Bytes()))
				var res bytes.Buffer
				_ = marshaler.Marshal(io.Writer(&res), r.value)
				t.Log(string(got.Bytes()))

				if string(got.Bytes()) != string(res.Bytes()) {
					t.Errorf("syncNodeSecrets() = '%v', want '%v'", string(got.Bytes()), string(res.Bytes()))
				}
			}
		})
	}
}
