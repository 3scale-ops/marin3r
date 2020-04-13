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

package envoy

import (
	"context"
	"crypto/tls"
	"reflect"
	"sync"
	"testing"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"go.uber.org/zap"
)

var (
	snapshotCache = cache.NewSnapshotCache(true, hasher{}, nil)
)

func Test_hasher_ID(t *testing.T) {
	type args struct {
		node *core.Node
	}
	tests := []struct {
		name string
		h    hasher
		args args
		want string
	}{
		{
			"Returns the node id",
			hasher{},
			args{&core.Node{Id: "node1"}},
			"node1",
		},
		{
			"Returns 'unknown'",
			hasher{},
			args{nil},
			"unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h.ID(tt.args.node); got != tt.want {
				t.Errorf("hasher.ID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewXdsServer(t *testing.T) {

	type args struct {
		ctx       context.Context
		adsPort   uint
		tlsConfig *tls.Config
		callbacks xds.Callbacks
		logger    *zap.SugaredLogger
	}
	tests := []struct {
		name string
		args args
		want *XdsServer
	}{
		{
			"Returns a new XdsServer from the given params",
			args{context.Background(), 10000, &tls.Config{}, &Callbacks{}, nil},
			&XdsServer{
				context.Background(),
				10000,
				&tls.Config{},
				xds.NewServer(context.Background(), snapshotCache, &Callbacks{}),
				snapshotCache,
				nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewXdsServer(tt.args.ctx, tt.args.adsPort, tt.args.tlsConfig, tt.args.callbacks, tt.args.logger); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewXdsServer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestXdsServer_RunADSServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	tests := []struct {
		name string
		xdss *XdsServer
	}{
		{
			"Runs the ads server",
			&XdsServer{
				ctx,
				10000,
				&tls.Config{},
				xds.NewServer(context.Background(), snapshotCache, &Callbacks{}),
				snapshotCache,
				func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wait sync.WaitGroup
			wait.Add(1)
			go func() {
				tt.xdss.RunADSServer()
				wait.Done()
			}()
			cancel()
			wait.Wait()
		})
	}
	// Avoid go vet complaining about cancel not being
	// invoked in some code paths
	cancel()
}

func TestXdsServer_GetSnapshotCache(t *testing.T) {
	tests := []struct {
		name string
		xdss *XdsServer
		want *cache.SnapshotCache
	}{
		{
			"Gets the server's SnapshotCache",
			&XdsServer{
				context.Background(),
				10000,
				&tls.Config{},
				xds.NewServer(context.Background(), snapshotCache, &Callbacks{}),
				snapshotCache,
				nil,
			},
			&snapshotCache,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.xdss.GetSnapshotCache(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("XdsServer.GetSnapshotCache() = %v, want %v", got, tt.want)
			}
		})
	}
}
