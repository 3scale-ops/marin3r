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

package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/3scale/marin3r/pkg/webhook"
	"go.uber.org/zap"
)

const (
	webhookPort = 8443
)

var (
// runtimeScheme = runtime.NewScheme()
// codecs        = serializer.NewCodecFactory(runtimeScheme)
// deserializer  = codecs.UniversalDeserializer()

// // (https://github.com/kubernetes/kubernetes/issues/57982)
// defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

// RunWebhook runs the mutating admission controller in a gorouting and
// waits forever until the context is closed.
func RunWebhook(ctx context.Context, tlsCertificatePath string, tlsKeyPath string, tlsCAPath string, namespace string, logger *zap.SugaredLogger) error {

	mux := http.NewServeMux()
	mux.Handle("/mutate", webhook.AdmitFuncHandler(webhook.MutatePod, logger))

	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		// TODO: mechanism to reload server certificate when renewed
		// Probably the easieast way is to have a goroutine force server
		// graceful shutdown when it detects the certificate has changed
		// The goroutine needs to receive both the context and the stopper channel
		// and close all other subroutines the same way as when a os signal is received
		Certificates: []tls.Certificate{getCertificate(tlsCertificatePath, tlsKeyPath, logger)},
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%v", webhookPort),
		Handler:      mux,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	go func() {
		logger.Fatal(srv.ListenAndServeTLS("", ""))
	}()

	<-ctx.Done()
	logger.Infof("Shutting down admission webhook")
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error(err)
	}
	return nil
}
