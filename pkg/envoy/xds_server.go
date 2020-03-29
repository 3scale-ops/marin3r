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
	"net/http"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	secretservice "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
)

const (
	grpcMaxConcurrentStreams = 1000000
)

type XdsServer struct {
	ctx            context.Context
	gatewayPort    uint
	managementPort uint
	tlsConfig      *tls.Config
	server         xds.Server
	snapshotCache  cache.SnapshotCache
	logger         *zap.SugaredLogger
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

func NewXdsServer(ctx context.Context, gatewayPort uint, managementPort uint, tlsConfig *tls.Config, callbacks xds.Callbacks, logger *zap.SugaredLogger) *XdsServer {
	snapshotCache := cache.NewSnapshotCache(true, hasher{}, nil)
	srv := xds.NewServer(ctx, snapshotCache, callbacks)

	return &XdsServer{
		ctx:            ctx,
		gatewayPort:    gatewayPort,
		managementPort: managementPort,
		tlsConfig:      tlsConfig,
		server:         srv,
		snapshotCache:  snapshotCache,
		logger:         logger,
	}
}

// RunManagementServer starts an xDS server at the given port.
func (xdss *XdsServer) RunManagementServer() {
	// gRPC golang library sets a very small upper bound for the number gRPC/h2
	// streams over a single TCP connection. If a proxy multiplexes requests over
	// a single connection to the management server, then it might lead to
	// availability problems.
	grpcServer := grpc.NewServer(
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
		grpc.Creds(credentials.NewTLS(xdss.tlsConfig)),
	)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", xdss.managementPort))
	if err != nil {
		xdss.logger.Fatal(err)
	}

	// register services
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, xdss.server)
	// endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	// clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	// routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	// listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	secretservice.RegisterSecretDiscoveryServiceServer(grpcServer, xdss.server)
	// runtimeservice.RegisterRuntimeDiscoveryServiceServer(grpcServer, server)

	xdss.logger.Infof("Management server listening on %d\n", xdss.managementPort)
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			xdss.logger.Error(err)
		}
	}()
	<-xdss.ctx.Done()

	xdss.logger.Infof("Shutting down management server")
	grpcServer.GracefulStop()
}

// RunManagementGateway starts an HTTP gateway to an xDS server.
// TOD: mTLS support -> https://smallstep.com/hello-mtls/doc/server/go
func (xdss *XdsServer) RunManagementGateway() {
	xdss.logger.Infof("Starting HTTP/1.1 gateway on Port %d\n", xdss.gatewayPort)
	httpServer := &http.Server{
		Addr:      fmt.Sprintf(":%d", xdss.gatewayPort),
		Handler:   &xds.HTTPGateway{Server: xdss.server, Log: xdss.logger},
		TLSConfig: xdss.tlsConfig,
	}
	go func() {
		if err := httpServer.ListenAndServeTLS("", ""); err != nil {
			xdss.logger.Error(err)
		}
	}()

	<-xdss.ctx.Done()
	xdss.logger.Infof("Shutting down gateway")
	if err := httpServer.Shutdown(xdss.ctx); err != nil {
		xdss.logger.Error(err)
	}
}

func (envoyXdsServer *XdsServer) SetSnapshot(snapshot *cache.Snapshot, nodeID string) error {
	err := envoyXdsServer.snapshotCache.SetSnapshot(nodeID, *snapshot)

	if err != nil {
		return err
	}

	return nil
}
