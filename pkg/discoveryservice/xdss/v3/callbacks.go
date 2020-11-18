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
	"fmt"

	"github.com/3scale/marin3r/pkg/envoy"
	envoy_serializer "github.com/3scale/marin3r/pkg/envoy/serializer"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = log.Log.WithName("xds_server").WithName("v3")

// Callbacks is a type that implements go-control-plane/pkg/server/Callbacks
type Callbacks struct {
	OnError       func(nodeID, previousVersion, msg string, envoyAPI envoy.APIVersion) error
	SnapshotCache *cache_v3.SnapshotCache
	Logger        logr.Logger
}

// OnStreamOpen implements go-control-plane/pkg/server/Callbacks.OnStreamOpen
// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
func (cb *Callbacks) OnStreamOpen(ctx context.Context, id int64, typ string) error {
	cb.Logger.V(1).Info("Stream opened", "StreamId", id)
	return nil
}

// OnStreamClosed implements go-control-plane/pkg/server/Callbacks.OnStreamClosed
// OnStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
func (cb *Callbacks) OnStreamClosed(id int64) {
	cb.Logger.V(1).Info("Stream closed", "StreamID", id)
}

// OnStreamRequest implements go-control-plane/pkg/server/Callbacks.OnStreamRequest
// OnStreamRequest is called once a request is received on a stream.
// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
func (cb *Callbacks) OnStreamRequest(id int64, req *envoy_service_discovery_v3.DiscoveryRequest) error {
	cb.Logger.V(1).Info("Received request", "ResourceNames", req.ResourceNames, "Version", req.VersionInfo, "TypeURL", req.TypeUrl, "NodeID", req.Node.Id, "StreamID", id)

	if req.ErrorDetail != nil {
		snap, err := (*cb.SnapshotCache).GetSnapshot(req.Node.Id)
		if err != nil {
			return err
		}
		// All resource types are always kept at the same version
		failingVersion := snap.GetVersion("type.googleapis.com/envoy.api.v3.ClusterLoadAssignment")
		cb.Logger.Error(fmt.Errorf(req.ErrorDetail.Message), "A gateway reported an error", "CurrentVersion", req.VersionInfo, "FailingVersion", failingVersion, "NodeID", req.Node.Id, "StreamID", id)
		if err := cb.OnError(req.Node.Id, failingVersion, req.ErrorDetail.Message, envoy.APIv3); err != nil {
			cb.Logger.Error(err, "Error calling OnErrorFn", "NodeID", req.Node.Id, "StreamID", id)
			return err
		}
	}
	return nil
}

// OnStreamResponse implements go-control-plane/pkgserver/Callbacks.OnStreamResponse
// OnStreamResponse is called immediately prior to sending a response on a stream.
func (cb *Callbacks) OnStreamResponse(id int64, req *envoy_service_discovery_v3.DiscoveryRequest, rsp *envoy_service_discovery_v3.DiscoveryResponse) {
	resources := []string{}
	for _, r := range rsp.Resources {
		j, _ := envoy_serializer.NewResourceMarshaller(envoy_serializer.JSON, envoy.APIv3).Marshal(r)
		resources = append(resources, string(j))
	}
	if rsp.TypeUrl == "type.googleapis.com/envoy.api.v3.auth.Secret" {
		cb.Logger.V(1).Info("Response sent to gateway",
			"ResourcesNames", req.ResourceNames, "TypeURL", req.TypeUrl, "NodeID", req.Node.Id, "StreamID", id, "Version", rsp.GetVersionInfo())
	} else {
		cb.Logger.V(1).Info("Response sent to gateway",
			"Resources", resources, "TypeURL", req.TypeUrl, "NodeID", req.Node.Id, "StreamID", id, "Version", rsp.GetVersionInfo())
	}
}

// OnFetchRequest implements go-control-plane/pkg/server/Callbacks.OnFetchRequest
// OnFetchRequest is called for each Fetch request. Returning an error will end processing of the
// request and respond with an error.
func (cb *Callbacks) OnFetchRequest(ctx context.Context, req *envoy_service_discovery_v3.DiscoveryRequest) error {
	return nil
}

// OnFetchResponse implements go-control-plane/pkg/server/Callbacks.OnFetchRequest
// OnFetchResponse is called immediately prior to sending a response.
func (cb *Callbacks) OnFetchResponse(req *envoy_service_discovery_v3.DiscoveryRequest, resp *envoy_service_discovery_v3.DiscoveryResponse) {
}
