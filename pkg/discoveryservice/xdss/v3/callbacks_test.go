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
	"testing"

	"github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/stats"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/golang/protobuf/ptypes/any"
	"google.golang.org/genproto/googleapis/rpc/status"
	ctrl "sigs.k8s.io/controller-runtime"
)

func fakeTestCache() *cache_v3.SnapshotCache {

	snapshotCache := cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)

	snapshotCache.SetSnapshot("node1", cache_v3.Snapshot{
		Resources: [6]cache_v3.Resources{
			{Version: "1", Items: map[string]cache_types.Resource{
				"endpoint1": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint1"},
			}},
			{Version: "1", Items: map[string]cache_types.Resource{
				"cluster1": &envoy_config_cluster_v3.Cluster{Name: "cluster1"},
			}},
			{Version: "1", Items: map[string]cache_types.Resource{}},
			{Version: "1", Items: map[string]cache_types.Resource{}},
			{Version: "1", Items: map[string]cache_types.Resource{}},
			{Version: "1", Items: map[string]cache_types.Resource{}},
		}},
	)

	return &snapshotCache
}

func TestCallbacks_OnStreamOpen(t *testing.T) {
	type args struct {
		ctx context.Context
		id  int64
		typ string
	}
	tests := []struct {
		name    string
		cb      *Callbacks
		args    args
		wantErr bool
	}{
		{
			"OnStreamOpen()",
			&Callbacks{Logger: ctrl.Log},
			args{context.Background(), 1, "xxxx"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cb.OnStreamOpen(tt.args.ctx, tt.args.id, tt.args.typ); (err != nil) != tt.wantErr {
				t.Errorf("Callbacks.OnStreamOpen() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCallbacks_OnStreamClosed(t *testing.T) {
	type args struct {
		id int64
	}
	tests := []struct {
		name string
		cb   *Callbacks
		args args
	}{
		{
			"OnStreamClosed()",
			&Callbacks{
				Stats:  stats.New(),
				Logger: ctrl.Log,
			},
			args{1},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cb.OnStreamClosed(tt.args.id)
		})
	}
}

func TestCallbacks_OnStreamRequest(t *testing.T) {
	type args struct {
		id  int64
		req *envoy_service_discovery_v3.DiscoveryRequest
	}
	tests := []struct {
		name    string
		cb      *Callbacks
		args    args
		wantErr bool
	}{
		{
			"OnStreamRequest()",
			&Callbacks{
				Stats:  stats.New(),
				Logger: ctrl.Log,
			},
			args{1, &envoy_service_discovery_v3.DiscoveryRequest{
				Node:          &envoy_config_core_v3.Node{Id: "node1", Cluster: "cluster1"},
				ResourceNames: []string{"string1", "string2"},
				TypeUrl:       "some-type",
				ErrorDetail:   nil,
			}},
			false,
		},
		{
			"OnStreamRequest() NACK received",
			&Callbacks{
				Stats:  stats.New(),
				Logger: ctrl.Log,
			},
			args{1, &envoy_service_discovery_v3.DiscoveryRequest{
				Node:          &envoy_config_core_v3.Node{Id: "node1", Cluster: "cluster1"},
				ResourceNames: []string{"string1", "string2"},
				TypeUrl:       "some-type",
				ErrorDetail:   &status.Status{Code: 0, Message: "xxxx"},
			}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cb.OnStreamRequest(tt.args.id, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("Callbacks.OnStreamRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCallbacks_OnStreamResponse(t *testing.T) {
	type args struct {
		id       int64
		request  *envoy_service_discovery_v3.DiscoveryRequest
		response *envoy_service_discovery_v3.DiscoveryResponse
	}
	tests := []struct {
		name string
		cb   *Callbacks
		args args
	}{
		{
			"OnStreamResponse()",
			&Callbacks{
				Stats:  stats.New(),
				Logger: ctrl.Log,
			},
			args{1,
				&envoy_service_discovery_v3.DiscoveryRequest{
					Node:          &envoy_config_core_v3.Node{Id: "node1", Cluster: "cluster1"},
					ResourceNames: []string{"string1", "string2"},
					TypeUrl:       "some-type",
					ErrorDetail:   nil,
				},
				&envoy_service_discovery_v3.DiscoveryResponse{
					Resources: []*any.Any{
						{TypeUrl: "some-type", Value: []byte("some-value")},
					},
					TypeUrl: "some-type",
				},
			},
		},
		{
			"OnStreamResponse() special treatment of secret resources",
			&Callbacks{
				Stats:  stats.New(),
				Logger: ctrl.Log,
			},
			args{1,
				&envoy_service_discovery_v3.DiscoveryRequest{
					Node:          &envoy_config_core_v3.Node{Id: "node1", Cluster: "cluster1"},
					ResourceNames: []string{"string1", "string2"},
					TypeUrl:       "some-type",
					ErrorDetail:   nil,
				},
				&envoy_service_discovery_v3.DiscoveryResponse{
					Resources: []*any.Any{
						{TypeUrl: "type.googleapis.com/envoy.api.v3.auth.Secret", Value: []byte("some-value")},
					},
					TypeUrl: "type.googleapis.com/envoy.api.v3.auth.Secret",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cb.OnStreamResponse(tt.args.id, tt.args.request, tt.args.response)
		})
	}
}

func TestCallbacks_OnFetchRequest(t *testing.T) {
	type args struct {
		ctx context.Context
		req *envoy_service_discovery_v3.DiscoveryRequest
	}
	tests := []struct {
		name    string
		cb      *Callbacks
		args    args
		wantErr bool
	}{
		{
			"OnFetchRequest()",
			&Callbacks{Logger: ctrl.Log},
			args{
				context.Background(),
				&envoy_service_discovery_v3.DiscoveryRequest{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cb.OnFetchRequest(tt.args.ctx, tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("Callbacks.OnFetchRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCallbacks_OnFetchResponse(t *testing.T) {
	type args struct {
		req  *envoy_service_discovery_v3.DiscoveryRequest
		resp *envoy_service_discovery_v3.DiscoveryResponse
	}
	tests := []struct {
		name string
		cb   *Callbacks
		args args
	}{
		{
			"OnFetchResponse()",
			&Callbacks{Logger: ctrl.Log},
			args{&envoy_service_discovery_v3.DiscoveryRequest{}, &envoy_service_discovery_v3.DiscoveryResponse{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cb.OnFetchResponse(tt.args.req, tt.args.resp)
		})
	}
}
