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
	"github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/stats"
	xdss_v3 "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/v3"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	server_v3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/go-logr/logr"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

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

var (
	setupLog = ctrl.Log.WithName("xds_server")
)

// XdsServer is a type that holds configuration
// and runtime objects for the envoy xds server
type XdsServer struct {
	ctx              context.Context
	xDSPort          uint
	tlsConfig        *tls.Config
	serverV3         server_v3.Server
	snapshotCacheV3  cache_v3.SnapshotCache
	callbacksV3      *xdss_v3.Callbacks
	discoveryStatsV3 *stats.Stats
}

// NewXdsServer creates a new XdsServer object fron the given params
func NewXdsServer(ctx context.Context, xDSPort uint, tlsConfig *tls.Config, logger logr.Logger) *XdsServer {

	xdsLogger := logger.WithName("xds")

	discoveryStatsV3 := stats.New()

	// register the custom metrics collector with the global
	// prometheus registry
	metrics.Registry.MustRegister(discoveryStatsV3)

	snapshotCacheV3 := cache_v3.NewSnapshotCache(
		true,
		cache_v3.IDHash{},
		clogger{Logger: xdsLogger.WithName("cache").WithName("v3")},
	)

	callbacksV3 := &xdss_v3.Callbacks{
		Stats:  discoveryStatsV3,
		Logger: xdsLogger.WithName("server").WithName("v3"),
	}

	srvV3 := server_v3.NewServer(ctx, snapshotCacheV3, callbacksV3)

	return &XdsServer{
		ctx:              ctx,
		xDSPort:          xDSPort,
		tlsConfig:        tlsConfig,
		serverV3:         srvV3,
		snapshotCacheV3:  snapshotCacheV3,
		callbacksV3:      callbacksV3,
		discoveryStatsV3: discoveryStatsV3,
	}
}

// Start starts an xDS server at the given port.
func (xdss *XdsServer) Start(client kubernetes.Interface, namespace string) error {

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
		setupLog.Error(err, "Error starting ADS server")
		return err
	}

	// channel to receive errors from the gorutine running the server
	errCh := make(chan error)

	// goroutine to run server
	envoy_service_discovery_v3.RegisterAggregatedDiscoveryServiceServer(grpcServer, xdss.serverV3)

	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	setupLog.Info(fmt.Sprintf("Aggregated discovery service listening on %d\n", xdss.xDSPort))

	// start the stats garbage collector
	stopGC := make(chan struct{})
	if err := xdss.callbacksV3.Stats.RunGC(client, namespace, stopGC); err != nil {
		return err
	}

	// wait until channel stopCh closed or an error is received
	select {

	case <-xdss.ctx.Done():
		setupLog.Info("shutting down xds server")
		close(stopGC)
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
func (xdss *XdsServer) GetCache(version envoy.APIVersion) xdss.Cache {
	return xdss_v3.NewCacheFromSnapshotCache(xdss.snapshotCacheV3)
}

// GetCache returns the discovery stats
func (xdss *XdsServer) GetDiscoveryStats(version envoy.APIVersion) *stats.Stats {
	return xdss.discoveryStatsV3
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
