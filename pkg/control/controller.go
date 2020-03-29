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

package control

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/roivaz/marin3r/pkg/envoy"
	"go.uber.org/zap"
)

const (
	gatewayPort        = 19001
	managementPort     = 18000
	tlsCertificatePath = "./certs/marin3r-server.crt"
	tlsKeyPath         = "./certs/marin3r-server.key"
	tlsCAPath          = "./certs/ca.crt"
)

func NewController() error {

	// Create the logger
	lg, _ := zap.NewProduction()
	defer lg.Sync() // flushes buffer, if any
	logger := lg.Sugar()

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

	// Init the xDS server with mTLS
	certificate, err := tls.LoadX509KeyPair(tlsCertificatePath, tlsKeyPath)
	if err != nil {
		logger.Panicf("Could not load server certificate: '%s'", tlsCertificatePath, tlsKeyPath, err)
	}
	certPool := x509.NewCertPool()
	if bs, err := ioutil.ReadFile(tlsCAPath); err != nil {
		log.Fatalf("Failed to read client ca cert: %s", err)
	} else {
		ok := certPool.AppendCertsFromPEM(bs)
		if !ok {
			log.Fatal("failed to append client certs")
		}
	}

	envoyXdsServer := envoy.NewXdsServer(
		ctx,
		gatewayPort,
		managementPort,
		&tls.Config{
			// TODO: mechanism to reload server certificate when renewed
			// Probably the easieast way is to have a goroutine force server
			// graceful shutdown when it detects the certificate has changed
			// The goroutine needs to receive both the context and the stopper channel
			// and close all other subroutines the same way as when a os signal is received
			Certificates: []tls.Certificate{certificate},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    certPool,
		},
		&envoy.Callbacks{Logger: logger},
		logger,
	)

	var wg sync.WaitGroup
	wg.Add(3)

	// Start the secret reconciler
	secretReconciler := NewSecretReconciler(
		ctx,
		envoyXdsServer,
		logger,
		stopper,
	)

	go func() {
		defer wg.Done()
		secretReconciler.RunSecretReconciler()
	}()

	go func() {
		defer wg.Done()
		envoyXdsServer.RunManagementServer()
	}()

	go func() {
		defer wg.Done()
		envoyXdsServer.RunManagementGateway()
	}()

	wg.Wait()
	logger.Infof("Controller has shut down")

	return nil
}
