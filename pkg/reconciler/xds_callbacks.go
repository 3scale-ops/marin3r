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

package reconciler

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"go.uber.org/zap"
)

// OnStreamOpen implements go-control-plane/pkg/server/Callbacks.OnStreamOpen
// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
func (cb *Callbacks) OnStreamOpen(ctx context.Context, id int64, typ string) error {
	cb.Logger.Debugf("OnStreamOpen for id %v", id)
	return nil
}

// OnStreamClosed implements go-control-plane/pkg/server/Callbacks.OnStreamClosed
// OnStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
func (cb *Callbacks) OnStreamClosed(id int64) {
	cb.Logger.Debugf("OnStreamClosed for id %v", id)
}

// OnStreamRequest implements go-control-plane/pkg/server/Callbacks.OnStreamRequest
// OnStreamRequest is called once a request is received on a stream.
// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
func (cb *Callbacks) OnStreamRequest(id int64, req *v2.DiscoveryRequest) error {
	cb.Logger.Debugf("OnStreamRequest. Node: '%s'. ResourceNames: '%s'. TypeURL: '%s' ", req.Node.Id, req.ResourceNames, req.TypeUrl)

	if req.ErrorDetail != nil {
		cb.Logger.Errorf("OnStreamRequest error pushing snapshot to gateway: code: %v message %s", req.ErrorDetail.Code, req.ErrorDetail.Message)
		// if cb.OnError != nil {
		// 	cb.OnError()
		// }
		return fmt.Errorf("OnStreamRequest error pushing snapshot to gateway %v", req.ErrorDetail.Message)
	}
	return nil
}

// OnStreamResponse implements go-control-plane/pkg/server/Callbacks.OnStreamResponse
// OnStreamResponse is called immediately prior to sending a response on a stream.
func (cb *Callbacks) OnStreamResponse(i int64, request *v2.DiscoveryRequest, response *v2.DiscoveryResponse) {
	cb.Logger.Debug("OnStreamResponse: %s", spew.Sprint(response))
}

// OnFetchRequest implements go-control-plane/pkg/server/Callbacks.OnFetchRequest
// OnFetchRequest is called for each Fetch request. Returning an error will end processing of the
// request and respond with an error.
func (cb *Callbacks) OnFetchRequest(ctx context.Context, req *v2.DiscoveryRequest) error {
	return nil
}

// OnFetchRequest implements go-control-plane/pkg/server/Callbacks.OnFetchRequest
// OnFetchResponse is called immediately prior to sending a response.
func (cb *Callbacks) OnFetchResponse(req *v2.DiscoveryRequest, resp *v2.DiscoveryResponse) {
}

type Callbacks struct {
	Logger *zap.SugaredLogger
	// OnError func()
}
