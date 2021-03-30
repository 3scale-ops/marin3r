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

package discoveryservice

import (
	"context"
	"crypto/tls"
	"reflect"
	"sync"
	"testing"

	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	xdss_v2 "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/v2"
	xdss_v3 "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/v3"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	server_v2 "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	server_v3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	snapshotCacheV2 = cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil)
	snapshotCacheV3 = cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)
	fn              = func(a, b, c string, d envoy.APIVersion) error { return nil }
)

func TestNewDualXdsServer(t *testing.T) {

	type args struct {
		ctx       context.Context
		adsPort   uint
		tlsConfig *tls.Config
		fn        onErrorFn
		logger    logr.Logger
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Returns a new DualXdsServer from the given params",
			args{context.Background(), 10000, &tls.Config{}, fn, ctrl.Log},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewDualXdsServer(tt.args.ctx, tt.args.adsPort, tt.args.tlsConfig, tt.args.fn, tt.args.logger)
			if got.snapshotCacheV2 == nil || got.snapshotCacheV3 == nil ||
				got.serverV2 == nil || got.serverV3 == nil ||
				got.callbacksV2 == nil || got.callbacksV3 == nil {
				t.Errorf("TestNewDualXdsServer = expected non-empty caches")
			}
		})
	}
}

func TestDualXdsServer_Start(t *testing.T) {

	tests := []struct {
		name string
		xdss *DualXdsServer
	}{
		{
			"Runs the ads server",
			&DualXdsServer{
				context.Background(),
				10000,
				&tls.Config{},
				server_v2.NewServer(context.Background(), snapshotCacheV2, &xdss_v2.Callbacks{Logger: ctrl.Log}),
				server_v3.NewServer(context.Background(), snapshotCacheV3, &xdss_v3.Callbacks{Logger: ctrl.Log}),
				snapshotCacheV2,
				snapshotCacheV3,
				&xdss_v2.Callbacks{Logger: ctrl.Log},
				&xdss_v3.Callbacks{Logger: ctrl.Log},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wait sync.WaitGroup
			stopCh := make(chan struct{})
			wait.Add(1)
			go func() {
				if err := tt.xdss.Start(stopCh); err != nil {
					t.Errorf("TestDualXdsServer_Start = non nil error: '%s'", err)
				}
				wait.Done()
			}()
			close(stopCh)
			wait.Wait()
		})
	}
}

func TestDualXdsServer_GetCache(t *testing.T) {
	tests := []struct {
		name    string
		xdss    *DualXdsServer
		want    xdss.Cache
		version envoy.APIVersion
	}{
		{
			"Gets the server's Cache",
			&DualXdsServer{
				context.Background(),
				10000,
				&tls.Config{},
				server_v2.NewServer(context.Background(), snapshotCacheV2, &xdss_v2.Callbacks{Logger: ctrl.Log}),
				server_v3.NewServer(context.Background(), snapshotCacheV3, &xdss_v3.Callbacks{Logger: ctrl.Log}),
				snapshotCacheV2,
				snapshotCacheV3,
				&xdss_v2.Callbacks{Logger: ctrl.Log},
				&xdss_v3.Callbacks{Logger: ctrl.Log},
			},
			xdss_v2.NewCache(snapshotCacheV2),
			envoy.APIv2,
		},
		{
			"Gets the server's Cache",
			&DualXdsServer{
				context.Background(),
				10000,
				&tls.Config{},
				server_v2.NewServer(context.Background(), snapshotCacheV2, &xdss_v2.Callbacks{Logger: ctrl.Log}),
				server_v3.NewServer(context.Background(), snapshotCacheV3, &xdss_v3.Callbacks{Logger: ctrl.Log}),
				snapshotCacheV2,
				snapshotCacheV3,
				&xdss_v2.Callbacks{Logger: ctrl.Log},
				&xdss_v3.Callbacks{Logger: ctrl.Log},
			},
			xdss_v3.NewCache(snapshotCacheV3),
			envoy.APIv3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.xdss.GetCache(tt.version); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DualXdsServer.GetCache() = %v, want %v", got, tt.want)
			}
		})
	}
}
