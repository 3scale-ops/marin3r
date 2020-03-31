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
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/roivaz/marin3r/pkg/envoy"
	"github.com/roivaz/marin3r/pkg/reconciler"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

const (
	gatewayPort    = 19001
	managementPort = 18000
)

func NewController(tlsCertificatePath, tlsKeyPath, tlsCAPath, logLevel, namespace string, ooCluster bool) error {

	logger := newLogger(logLevel)

	ctx, stopper := runSignalWatcher(logger)

	// -------------------------
	// ---- Init components ----
	// -------------------------

	// Init the xDS server
	xdss := envoy.NewXdsServer(
		ctx,
		gatewayPort,
		managementPort,
		&tls.Config{
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

	// Init the cache worker
	xds_cache := envoy.NewCacheWorker(xdss.GetSnapshotCache(), stopper, logger)

	// Init the secret reconciler
	var clientset *kubernetes.Clientset
	var err error
	if ooCluster {
		clientset, err = outOfClusterClient()
		if err != nil {
			return err
		}
	} else {
		clientset, err = inClusterClient()
		if err != nil {
			return err
		}
	}
	secretReconciler := reconciler.NewSecretReconciler(clientset, namespace, xds_cache.Queue, ctx, logger, stopper)

	// ------------------------
	// ---- Run components ----
	// ------------------------

	// Start CacheWorker
	var waitCacheWorker sync.WaitGroup
	waitCacheWorker.Add(1)
	go func() {
		defer waitCacheWorker.Done()
		xds_cache.RunCacheWorker()
	}()

	// Start reconcilers
	var waitReconcilers sync.WaitGroup
	waitReconcilers.Add(1)
	go func() {
		defer waitReconcilers.Done()
		secretReconciler.RunSecretReconciler()
	}()

	// Finally start the servers
	var waitServers sync.WaitGroup
	waitServers.Add(1)
	go func() {
		defer waitServers.Done()
		xdss.RunManagementServer()
	}()

	waitServers.Add(1)
	go func() {
		defer waitServers.Done()
		xdss.RunManagementGateway()
	}()

	// Stop in order
	waitReconcilers.Wait()
	waitCacheWorker.Wait()
	waitServers.Wait()

	logger.Infof("Controller has shut down")

	return nil
}

func runSignalWatcher(logger *zap.SugaredLogger) (context.Context, chan struct{}) {
	// Create a context and cancel it when proper
	// signals are received
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	// Channel to stop reconcilers
	stopper := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		oscall := <-sigc
		logger.Infof("Received system call: %+v", oscall)
		close(stopper)
		cancel()
	}()

	return ctx, stopper
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

func newLogger(logLevel string) *zap.SugaredLogger {

	rawJSON := []byte(`{
		"level": "info",
		"outputPaths": ["stdout"],
		"errorOutputPaths": ["stderr"],
		"encoding": "json",
		"encoderConfig": {
			"messageKey": "message",
			"levelKey": "level",
			"levelEncoder": "lowercase"
		}
	}`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	if logLevel != "info" {
		switch logLevel {
		case "debug":
			cfg.Level.SetLevel(zap.DebugLevel)
		case "warn":
			cfg.Level.SetLevel(zap.WarnLevel)
		case "error":
			cfg.Level.SetLevel(zap.ErrorLevel)
		}
	}

	defer logger.Sync() // flushes buffer, if any
	return logger.Sugar()
}
