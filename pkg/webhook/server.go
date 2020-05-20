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

package webhook

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("mutating_webhook")

type WebhookServer struct {
	ctx       context.Context
	port      int32
	tlsConfig *tls.Config
}

// NewWebhookServer creates a new WebhookServer object fron the given params
func NewWebhookServer(ctx context.Context, port int32, tlsConfig *tls.Config) *WebhookServer {
	return &WebhookServer{
		ctx:       ctx,
		port:      port,
		tlsConfig: tlsConfig,
	}
}

// Start runs the mutating admission controller in a goroutine and
// waits forever until the stopper signal is sent.
func (ws *WebhookServer) Start(stopCh <-chan struct{}) error {

	mux := http.NewServeMux()
	mux.Handle("/mutate", AdmitFuncHandler(MutatePod, logger))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%v", ws.port),
		Handler:      mux,
		TLSConfig:    ws.tlsConfig,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	go func() {
		if err := srv.ListenAndServeTLS("", ""); err != nil {
			logger.Error(err, "Webhook server exited unexpectedly")
			os.Exit(1)
		}
	}()

	logger.Info("Mutating admission webhook started")
	<-stopCh
	logger.Info("Shutting down mutating admission webhook")
	if err := srv.Shutdown(ws.ctx); err != nil {
		logger.Error(err, "Webhook failed to shutdown gracefully")
	}
	return nil
}
