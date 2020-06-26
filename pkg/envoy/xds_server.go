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
	"fmt"
	"net"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	grpcMaxConcurrentStreams = 1000000
)

var logger = logf.Log.WithName("envoy_control_plane")

// XdsServer is a type that holds configuration
// and runtime objects for the envoy xds server
type XdsServer struct {
	ctx           context.Context
	adsPort       uint
	tlsConfig     *tls.Config
	server        xds.Server
	snapshotCache cache.SnapshotCache
}

// hasher returns node ID as an ID
type hasher struct {
}

func (h hasher) ID(node *core.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Id
}

// NewXdsServer creates a new XdsServer object fron the given params
func NewXdsServer(ctx context.Context, adsPort uint,
	tlsConfig *tls.Config, callbacks xds.Callbacks) *XdsServer {
	snapshotCache := cache.NewSnapshotCache(true, hasher{}, nil)
	srv := xds.NewServer(ctx, snapshotCache, callbacks)

	return &XdsServer{
		ctx:           ctx,
		adsPort:       adsPort,
		tlsConfig:     tlsConfig,
		server:        srv,
		snapshotCache: snapshotCache,
	}
}

// Start starts an xDS server at the given port.
func (xdss *XdsServer) Start(stopCh <-chan struct{}) error {
	// gRPC golang library sets a very small upper bound for the number gRPC/h2
	// streams over a single TCP connection. If a proxy multiplexes requests over
	// a single connection to the management server, then it might lead to
	// availability problems.
	grpcServer := grpc.NewServer(
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
		grpc.Creds(credentials.NewTLS(xdss.tlsConfig)),
	)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", xdss.adsPort))
	if err != nil {
		logger.Error(err, "Error starting aDS server")
		return err
	}

	// channel to receive errors from the gorutine running the server
	errCh := make(chan error)

	// goroutine to run server
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, xdss.server)
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	logger.Info(fmt.Sprintf("Aggregated discovery service listening on %d\n", xdss.adsPort))

	// wait until channel stopCh closed or an error is received
	select {
	case <-stopCh:
		grpcServer.GracefulStop()
		select {
		case err := <-errCh:
			logger.Error(err, "Server graceful stop failed")
			return err
		default:
			logger.Info("Server stopped gracefully")
			return nil
		}
	case err := <-errCh:
		logger.Error(err, "Server failed")
		return err
	}

}

// GetSnapshotCache returns the xds_cache.SnapshotCache
func (xdss *XdsServer) GetSnapshotCache() *cache.SnapshotCache {
	return &xdss.snapshotCache
}
