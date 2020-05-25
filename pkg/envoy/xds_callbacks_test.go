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
	"testing"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/golang/protobuf/ptypes/any"
	"google.golang.org/genproto/googleapis/rpc/status"
)

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
			&Callbacks{},
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
			&Callbacks{},
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
		req *v2.DiscoveryRequest
	}
	tests := []struct {
		name    string
		cb      *Callbacks
		args    args
		wantErr bool
	}{
		{
			"OnStreamRequest()",
			&Callbacks{},
			args{1, &v2.DiscoveryRequest{
				Node:          &core.Node{Id: "node1", Cluster: "cluster1"},
				ResourceNames: []string{"string1", "string2"},
				TypeUrl:       "some-type",
				ErrorDetail:   nil,
			}},
			false,
		},
		{
			"OnStreamRequest() error",
			&Callbacks{},
			args{1, &v2.DiscoveryRequest{
				Node:          &core.Node{Id: "node1", Cluster: "cluster1"},
				ResourceNames: []string{"string1", "string2"},
				TypeUrl:       "some-type",
				ErrorDetail:   &status.Status{Code: 0, Message: "xxxx"},
			}},
			true,
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
		request  *v2.DiscoveryRequest
		response *v2.DiscoveryResponse
	}
	tests := []struct {
		name string
		cb   *Callbacks
		args args
	}{
		{
			"OnStreamResponse()",
			&Callbacks{},
			args{1,
				&v2.DiscoveryRequest{
					Node:          &core.Node{Id: "node1", Cluster: "cluster1"},
					ResourceNames: []string{"string1", "string2"},
					TypeUrl:       "some-type",
					ErrorDetail:   nil,
				},
				&v2.DiscoveryResponse{
					Resources: []*any.Any{
						{TypeUrl: "some-type", Value: []byte("some-value")},
					},
					TypeUrl: "some-type",
				},
			},
		},
		{
			"OnStreamResponse() special treatment of secret resources",
			&Callbacks{},
			args{1,
				&v2.DiscoveryRequest{
					Node:          &core.Node{Id: "node1", Cluster: "cluster1"},
					ResourceNames: []string{"string1", "string2"},
					TypeUrl:       "some-type",
					ErrorDetail:   nil,
				},
				&v2.DiscoveryResponse{
					Resources: []*any.Any{
						{TypeUrl: "type.googleapis.com/envoy.api.v2.auth.Secret", Value: []byte("some-value")},
					},
					TypeUrl: "type.googleapis.com/envoy.api.v2.auth.Secret",
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
		req *v2.DiscoveryRequest
	}
	tests := []struct {
		name    string
		cb      *Callbacks
		args    args
		wantErr bool
	}{
		{
			"OnFetchRequest()",
			&Callbacks{},
			args{
				context.Background(),
				&v2.DiscoveryRequest{},
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
		req  *v2.DiscoveryRequest
		resp *v2.DiscoveryResponse
	}
	tests := []struct {
		name string
		cb   *Callbacks
		args args
	}{
		{
			"OnFetchResponse()",
			&Callbacks{},
			args{&v2.DiscoveryRequest{}, &v2.DiscoveryResponse{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cb.OnFetchResponse(tt.args.req, tt.args.resp)
		})
	}
}
