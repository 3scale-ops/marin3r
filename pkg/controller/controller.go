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
	"crypto/x509"
	"io/ioutil"
	"log"
	"sync"

	"github.com/roivaz/marin3r/pkg/envoy"
	"github.com/roivaz/marin3r/pkg/events"
	"github.com/roivaz/marin3r/pkg/reconciler"
	"github.com/roivaz/marin3r/pkg/util"
	"go.uber.org/zap"
)

const (
	gatewayPort    = 19001
	managementPort = 18000
)

func NewController(
	tlsCertificatePath string, tlsKeyPath string, tlsCAPath string,
	namespace string, ooCluster bool, ctx context.Context, stopper chan struct{},
	logger *zap.SugaredLogger) error {

	// -------------------------
	// ---- Init components ----
	// -------------------------

	// Init the xDS server
	xdss := envoy.NewXdsServer(
		ctx,
		gatewayPort,
		managementPort,
		&tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				// Sadly, these 2 are required to use http2 and
				// must go first as others could be offered first
				// to the client and cause the connection to be
				// rejected
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,

				// This are for the management gateway
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
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    getCA(tlsCAPath, logger),
		},
		&envoy.Callbacks{Logger: logger},
		logger,
	)

	var client *util.K8s
	var err error
	if ooCluster {
		client, err = util.OutOfClusterClient()
		if err != nil {
			return err
		}
	} else {
		client, err = util.InClusterClient()
		if err != nil {
			return err
		}
	}

	// Init the cache worker
	rec := reconciler.NewReconciler(client, namespace, xdss.GetSnapshotCache(), stopper, logger)

	// Init event handlers
	secretHandler := events.NewSecretHandler(client, namespace, rec.Queue, ctx, logger, stopper)
	configMapHandler := events.NewConfigMapHandler(client, namespace, rec.Queue, ctx, logger, stopper)

	// ------------------------
	// ---- Run components ----
	// ------------------------

	// Start Reconciler
	var waitReconciler sync.WaitGroup
	waitReconciler.Add(1)
	go func() {
		defer waitReconciler.Done()
		rec.RunReconciler()
	}()

	// Start event handlers
	var waitEventHandlers sync.WaitGroup
	waitEventHandlers.Add(2)
	go func() {
		defer waitEventHandlers.Done()
		secretHandler.RunSecretHandler()
	}()
	go func() {
		defer waitEventHandlers.Done()
		configMapHandler.RunConfigMapHandler()
	}()

	// Finally start the servers
	var waitServers sync.WaitGroup
	waitServers.Add(2)
	go func() {
		defer waitServers.Done()
		xdss.RunManagementServer()
	}()
	go func() {
		defer waitServers.Done()
		xdss.RunManagementGateway()
	}()

	// Stop in order
	waitEventHandlers.Wait()
	waitReconciler.Wait()
	waitServers.Wait()

	logger.Infof("Controller has shut down")

	return nil
}

func getCertificate(certPath, keyPath string, logger *zap.SugaredLogger) tls.Certificate {
	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		logger.Fatalf("Could not load server certificate: '%s'", certPath, keyPath, err)
	}
	return certificate
}

func getCA(caPath string, logger *zap.SugaredLogger) *x509.CertPool {
	certPool := x509.NewCertPool()
	if bs, err := ioutil.ReadFile(caPath); err != nil {
		log.Fatalf("Failed to read client ca cert: %s", err)
	} else {
		ok := certPool.AppendCertsFromPEM(bs)
		if !ok {
			log.Fatal("failed to append client certs")
		}
	}
	return certPool
}
