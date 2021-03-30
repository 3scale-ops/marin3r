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
	"fmt"
	"net"
	"time"

	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	xdss_v2 "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/v2"
	xdss_v3 "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/v3"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	server_v2 "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	server_v3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/go-logr/logr"

	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

const (
	grpcMaxConcurrentStreams                          = 1000000
	grpcMaxConnectionAge                              = 43200 // 12 hours
	grpcMaxConnectionAgeGrace                         = 300   // 5 min
	grpcKeepaliveEnforcementPolicyMinTime             = 50
	grpcKeepaliveEnforcementPolicyPermitWithoutStream = false
)

// XdsServer in an interface that any xDS server should implement
type XdsServer interface {
	Start(<-chan struct{}) error
	GetCache(envoy.APIVersion) xdss.Cache
}

type onErrorFn func(nodeID, previousVersion, msg string, envoyAPI envoy.APIVersion) error

// DualXdsServer is a type that holds configuration
// and runtime objects for the envoy xds server
type DualXdsServer struct {
	ctx             context.Context
	xDSPort         uint
	tlsConfig       *tls.Config
	serverV2        server_v2.Server
	serverV3        server_v3.Server
	snapshotCacheV2 cache_v2.SnapshotCache
	snapshotCacheV3 cache_v3.SnapshotCache
	callbacksV2     *xdss_v2.Callbacks
	callbacksV3     *xdss_v3.Callbacks
}

// NewDualXdsServer creates a new DualXdsServer object fron the given params
func NewDualXdsServer(ctx context.Context, xDSPort uint, tlsConfig *tls.Config, fn onErrorFn, logger logr.Logger) *DualXdsServer {

	xdsLogger := logger.WithName("xds")

	snapshotCacheV2 := cache_v2.NewSnapshotCache(
		true,
		cache_v2.IDHash{},
		clogger{Logger: xdsLogger.WithName("cache").WithName("v2")},
	)
	snapshotCacheV3 := cache_v3.NewSnapshotCache(
		true,
		cache_v3.IDHash{},
		clogger{Logger: xdsLogger.WithName("cache").WithName("v3")},
	)

	callbacksV2 := &xdss_v2.Callbacks{
		OnError:       fn,
		SnapshotCache: &snapshotCacheV2,
		Logger:        xdsLogger.WithName("server").WithName("v2"),
	}
	callbacksV3 := &xdss_v3.Callbacks{
		OnError:       fn,
		SnapshotCache: &snapshotCacheV3,
		Logger:        xdsLogger.WithName("server").WithName("v3"),
	}

	srvV2 := server_v2.NewServer(ctx, snapshotCacheV2, callbacksV2)
	srvV3 := server_v3.NewServer(ctx, snapshotCacheV3, callbacksV3)

	return &DualXdsServer{
		ctx:             ctx,
		xDSPort:         xDSPort,
		tlsConfig:       tlsConfig,
		serverV2:        srvV2,
		serverV3:        srvV3,
		snapshotCacheV2: snapshotCacheV2,
		snapshotCacheV3: snapshotCacheV3,
		callbacksV2:     callbacksV2,
		callbacksV3:     callbacksV3,
	}
}

// Start starts an xDS server at the given port.
func (xdss *DualXdsServer) Start(stopCh <-chan struct{}) error {

	// gRPC golang library sets a very small upper bound for the number gRPC/h2
	// streams over a single TCP connection. If a proxy multiplexes requests over
	// a single connection to the management server, then it might lead to
	// availability problems.
	grpcServer := grpc.NewServer(
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcKeepaliveEnforcementPolicyMinTime * time.Second,
			PermitWithoutStream: grpcKeepaliveEnforcementPolicyPermitWithoutStream,
		}),
		grpc.Creds(credentials.NewTLS(xdss.tlsConfig)),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge:      grpcMaxConnectionAge * time.Second,
			MaxConnectionAgeGrace: grpcMaxConnectionAgeGrace * time.Second,
		}),
	)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", xdss.xDSPort))
	if err != nil {
		setupLog.Error(err, "Error starting aDS server")
		return err
	}

	// channel to receive errors from the gorutine running the server
	errCh := make(chan error)

	// goroutine to run server
	envoy_service_discovery_v2.RegisterAggregatedDiscoveryServiceServer(grpcServer, xdss.serverV2)
	envoy_service_discovery_v3.RegisterAggregatedDiscoveryServiceServer(grpcServer, xdss.serverV3)

	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	setupLog.Info(fmt.Sprintf("Aggregated discovery service listening on %d\n", xdss.xDSPort))

	// wait until channel stopCh closed or an error is received
	select {

	case <-stopCh:
		setupLog.Info("shutting down xds server")
		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()

		// Timeout on graceful stop
		t := time.NewTimer(10 * time.Second)
		select {
		case <-t.C:
			grpcServer.Stop()
		case <-stopped:
			t.Stop()
		}
		return nil

	case err := <-errCh:
		setupLog.Error(err, "Server failed")
		return err
	}

}

// GetCache returns the Cache
func (xdss *DualXdsServer) GetCache(version envoy.APIVersion) xdss.Cache {
	if version == envoy.APIv2 {
		return xdss_v2.NewCache(xdss.snapshotCacheV2)
	}
	return xdss_v3.NewCache(xdss.snapshotCacheV3)
}

type clogger struct {
	Logger logr.Logger
}

func (cl clogger) Debugf(format string, args ...interface{}) {
	cl.Logger.V(1).Info(fmt.Sprintf(format, args...))
}

func (cl clogger) Infof(format string, args ...interface{}) {
	cl.Logger.Info(fmt.Sprintf(format, args...))
}

func (cl clogger) Warnf(format string, args ...interface{}) {
	cl.Logger.Info(fmt.Sprintf(format, args...))
}

func (cl clogger) Errorf(format string, args ...interface{}) {
	cl.Logger.Error(fmt.Errorf("xds cache error"), fmt.Sprintf(format, args...))
}
