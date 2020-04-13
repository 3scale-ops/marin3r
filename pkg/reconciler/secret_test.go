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
	"sort"
	"sync"
	"testing"

	envoy_api_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/golang/protobuf/jsonpb"
	"github.com/roivaz/marin3r/pkg/cache"
	"github.com/roivaz/marin3r/pkg/util"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewSecretReconcileJob(t *testing.T) {
	type args struct {
		cn        string
		eventType EventType
		secret    corev1.Secret
	}
	tests := []struct {
		name string
		args args
		want *SecretReconcileJob
	}{
		{
			"Creates new job from 'Add' event",
			args{
				"common-name",
				Add,
				corev1.Secret{
					TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "secret"},
					Data: map[string][]byte{
						"tls.crt": []byte("xxxx"),
						"tls.key": []byte("yyyy"),
					}},
			},
			&SecretReconcileJob{
				eventType: Add,
				cn:        "common-name",
				secret: corev1.Secret{
					TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "secret"},
					Data: map[string][]byte{
						"tls.crt": []byte("xxxx"),
						"tls.key": []byte("yyyy"),
					},
				},
			},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSecretReconcileJob(tt.args.cn, tt.args.eventType, tt.args.secret); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSecretReconcileJob() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecretReconcileJob_Push(t *testing.T) {
	type args struct {
		queue chan ReconcileJob
	}
	tests := []struct {
		name string
		job  SecretReconcileJob
		args args
	}{
		{
			"Pushes a job to the queue",
			SecretReconcileJob{
				eventType: Update,
				cn:        "common-name",
				secret: corev1.Secret{
					TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "secret"},
					Data: map[string][]byte{
						"tls.crt": []byte("xxxx"),
						"tls.key": []byte("yyyy"),
					},
				},
			},
			args{make(chan ReconcileJob)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var wait sync.WaitGroup
			wait.Add(1)
			go func() {
				<-tt.args.queue
				wait.Done()
			}()
			tt.job.Push(tt.args.queue)
			wait.Wait()
		})
	}
}

func TestSecretReconcileJob_process(t *testing.T) {
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
		job     SecretReconcileJob
		args    args
		want    want
		wantErr bool
	}{
		{
			name: "Processes a secret job and generates a secret for each node's cache",
			job: SecretReconcileJob{
				eventType: Update,
				cn:        "common-name",
				secret: corev1.Secret{
					TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "secret"},
					Data: map[string][]byte{
						"tls.crt": []byte("xxxx"),
						"tls.key": []byte("yyyy"),
					},
				},
			},
			args: args{
				func() cache.Cache {
					c := cache.NewCache()
					c.NewNodeCache("node1")
					c.NewNodeCache("node2")
					return c
				}(),
				&util.K8s{},
				"default",
				func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
			want: want{
				nodeIDs: []string{"node1", "node2"},
				resources: []resource{
					{
						name:  "common-name",
						rtype: cache.Secret,
						value: &envoy_api_auth.Secret{
							Name: "common-name",
							Type: &envoy_api_auth.Secret_TlsCertificate{
								TlsCertificate: &envoy_api_auth.TlsCertificate{
									PrivateKey: &envoy_api_core.DataSource{
										Specifier: &envoy_api_core.DataSource_InlineBytes{InlineBytes: []byte("yyyy")},
									},
									CertificateChain: &envoy_api_core.DataSource{
										Specifier: &envoy_api_core.DataSource_InlineBytes{InlineBytes: []byte("xxxx")},
									}}}}}},
			},
			wantErr: false,
		},
		{
			name: "Missing 'tls.crt' from secret data returns an error",
			job: SecretReconcileJob{
				eventType: Update,
				cn:        "common-name",
				secret: corev1.Secret{
					TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "secret"},
					Data: map[string][]byte{
						"wrong_key": []byte("xxxx"),
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
				nodeIDs:   []string{},
				resources: []resource{},
			},
			wantErr: true,
		},
		{
			name: "Missing 'tls.key' from secret data returns an error",
			job: SecretReconcileJob{
				eventType: Update,
				cn:        "common-name",
				secret: corev1.Secret{
					TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "secret"},
					Data: map[string][]byte{
						"tls.crt": []byte("yyyy"),
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
				nodeIDs:   []string{},
				resources: []resource{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := tt.job.process(tt.args.c, tt.args.clientset, tt.args.namespace, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("SecretReconcileJob.process() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Sort the slices so results are consistent
			sort.Strings(got)
			sort.Strings(tt.want.nodeIDs)
			if !reflect.DeepEqual(got, tt.want.nodeIDs) {
				t.Errorf("SecretReconcileJob.process() = %v, want %v", got, tt.want.nodeIDs)
			}

			for _, nodeID := range got {

				// DeepEqual is not working for comparisons so we serialize
				// to json and then compare the resulting json strings
				marshaler := jsonpb.Marshaler{}
				for _, r := range tt.want.resources {

					var got bytes.Buffer
					_ = marshaler.Marshal(io.Writer(&got), tt.args.c.GetResource(nodeID, r.name, r.rtype))
					t.Log(string(got.Bytes()))
					var res bytes.Buffer
					_ = marshaler.Marshal(io.Writer(&res), r.value)
					t.Log(string(got.Bytes()))

					if string(got.Bytes()) != string(res.Bytes()) {
						t.Errorf("SecretReconcileJob.process() = '%v', want '%v'", string(got.Bytes()), string(res.Bytes()))
					}
				}
			}
		})
	}
}
