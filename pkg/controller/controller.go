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
	"fmt"
	"io/ioutil"
	"log"
	"sync"

	"github.com/3scale/marin3r/pkg/envoy"
	"github.com/3scale/marin3r/pkg/events"
	"github.com/3scale/marin3r/pkg/reconciler"
	"github.com/3scale/marin3r/pkg/util"
	"go.uber.org/zap"
)

const (
	adsPort = 18000
)

// NewController runs the envoy control plane components
func NewController(
	ctx context.Context, tlsCertificatePath string, tlsKeyPath string,
	tlsCAPath string, namespace string, ooCluster bool, logger *zap.SugaredLogger) error {

	// -------------------------
	// ---- Init components ----
	// -------------------------

	// Init the xDS server
	xdss := envoy.NewXdsServer(
		ctx,
		adsPort,
		&tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				// Sadly, these 2 non 256 are required to use http2 in go
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
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
			return fmt.Errorf("Failed to load the out of cluster k8s client: '%s'", err)
		}
	} else {
		client, err = util.InClusterClient()
		if err != nil {
			return fmt.Errorf("Failed to load the in cluster k8s client: '%s'", err)
		}
	}

	// Init the cache worker
	rec := reconciler.NewReconciler(ctx, client, namespace, xdss.GetSnapshotCache(), logger)

	// Init event handlers
	secretHandler := events.NewSecretHandler(ctx, client, namespace, rec.Queue, logger)
	configMapHandler := events.NewConfigMapHandler(ctx, client, namespace, rec.Queue, logger)

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
		if err := secretHandler.RunSecretHandler(); err != nil {
			logger.Panicf("SecretHandler returned an unrecoverable error, shutting down: '%s'", "err")
		}
	}()
	go func() {
		defer waitEventHandlers.Done()
		if err := configMapHandler.RunConfigMapHandler(); err != nil {
			logger.Panicf("ConfigMapHandler returned an unrecoverable error, shutting down: '%s'", "err")
		}
	}()

	// Finally start the servers
	var waitServers sync.WaitGroup
	waitServers.Add(1)
	go func() {
		defer waitServers.Done()
		if err := xdss.RunADSServer(); err != nil {
			logger.Panicf("ADS server returned an unrecoverable error, shutting down: '%s'", "err")
		}
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
