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

package events

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/3scale/marin3r/pkg/reconciler"
	"github.com/3scale/marin3r/pkg/util"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewSecretHandler(t *testing.T) {
	type args struct {
		ctx       context.Context
		client    *util.K8s
		namespace string
		queue     chan reconciler.ReconcileJob
		logger    *zap.SugaredLogger
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Creates a new SecretHandler",
			args{
				context.Background(),
				&util.K8s{},
				"default",
				make(chan reconciler.ReconcileJob),
				func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSecretHandler(tt.args.ctx, tt.args.client, tt.args.namespace, tt.args.queue, tt.args.logger)
			want := &SecretHandler{tt.args.ctx, tt.args.client, tt.args.namespace, tt.args.queue, tt.args.logger}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("NewSecretHandler() = %v, want %v", got, want)
			}
		})
	}
}

func TestSecretHandler_RunSecretHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	tests := []struct {
		name string
		cmh  *SecretHandler
	}{
		{
			"Runs a SecretHandler",
			&SecretHandler{
				ctx,
				util.FakeClusterClient(),
				"default",
				make(chan reconciler.ReconcileJob),
				func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wait sync.WaitGroup
			wait.Add(1)
			go func() {
				tt.cmh.RunSecretHandler()
				wait.Done()
			}()
			cancel()
			wait.Wait()
		})
	}
	cancel()
}

func Test_onSecretAdd(t *testing.T) {
	type args struct {
		obj    interface{}
		queue  chan reconciler.ReconcileJob
		logger *zap.SugaredLogger
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Pushes job when an 'Add' event on a watched secret is received",
			args{
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name: "secret",
						Annotations: map[string]string{
							certificateAnnotation: "xxxx",
						},
					},
					Data: map[string][]byte{
						"tls.crt": []byte("xxxx"),
						"tls.key": []byte("yyyy"),
					},
				},
				make(chan reconciler.ReconcileJob),
				func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
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
			onSecretAdd(tt.args.obj, tt.args.queue, tt.args.logger)
			wait.Wait()
		})
	}
}

func Test_onSecretUpdate(t *testing.T) {
	type args struct {
		obj    interface{}
		queue  chan reconciler.ReconcileJob
		logger *zap.SugaredLogger
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Pushes job when an 'Update' event on a watched secret is received",
			args{
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name: "secret",
						Annotations: map[string]string{
							certificateAnnotation: "xxxx",
						},
					},
					Data: map[string][]byte{
						"tls.crt": []byte("xxxx"),
						"tls.key": []byte("yyyy"),
					},
				},
				make(chan reconciler.ReconcileJob),
				func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
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
			onSecretUpdate(tt.args.obj, tt.args.queue, tt.args.logger)
			wait.Wait()
		})
	}
}

func Test_onSecretDelete(t *testing.T) {
	type args struct {
		obj    interface{}
		queue  chan reconciler.ReconcileJob
		logger *zap.SugaredLogger
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Pushes job when an 'Delete' event on a watched secret is received",
			args{
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name: "secret",
						Annotations: map[string]string{
							certificateAnnotation: "xxxx",
						},
					},
					Data: map[string][]byte{
						"tls.crt": []byte("xxxx"),
						"tls.key": []byte("yyyy"),
					},
				},
				make(chan reconciler.ReconcileJob),
				func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
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
			onSecretDelete(tt.args.obj, tt.args.queue, tt.args.logger)
			wait.Wait()
		})
	}
}
