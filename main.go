/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	envoycontroller "github.com/3scale/marin3r/controllers/envoy"
	operatorcontroller "github.com/3scale/marin3r/controllers/operator"
	"github.com/3scale/marin3r/pkg/envoy"
	"github.com/3scale/marin3r/pkg/webhook"
	"github.com/go-logr/logr"
	// +kubebuilder:scaffold:imports
)

// Change below variables to serve metrics on different host or port.
const (
	host                  string = "0.0.0.0"
	metricsPort           int    = 8383
	webhookPort           int    = 8443
	envoyControlPlanePort uint   = 18000
)

var (
	tlsCertificatePath   string
	tlsKeyPath           string
	tlsCAPath            string
	isDiscoveryService   bool
	debug                bool
	metricsAddr          string
	enableLeaderElection bool
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	utilruntime.Must(envoyv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	flag.StringVar(&metricsAddr, "metrics-addr", fmt.Sprintf(":%v", metricsPort), "The address the metric endpoint binds to.")
	flag.StringVar(&tlsCertificatePath, "certificate", "/etc/marin3r/tls/server.crt", "Server certificate")
	flag.StringVar(&tlsKeyPath, "private-key", "/etc/marin3r/tls/server.key", "The private key of the server certificate")
	flag.StringVar(&tlsCAPath, "ca", "/etc/marin3r/tls/ca.crt", "The CA of the server certificate")
	flag.BoolVar(&isDiscoveryService, "discovery-service", false, "Run the discovery-service instead of the operator")
	flag.BoolVar(&debug, "debug", false, "Enable debug logs")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))

	cfg := ctrl.GetConfigOrDie()
	ctx := context.Background()

	if !isDiscoveryService {

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme,
			MetricsBindAddress: metricsAddr,
			Port:               webhookPort,
			LeaderElection:     enableLeaderElection,
			LeaderElectionID:   "2cfbe7d6.marin3r.3scale.net",
		})
		if err != nil {
			setupLog.Error(err, "unable to start manager")
			os.Exit(1)
		}

		// watch for syscalls
		stopCh := signals.SetupSignalHandler()

		// Start controllers
		runOperator(ctx, mgr, stopCh)

	} else {

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme,
			MetricsBindAddress: metricsAddr,
			Port:               webhookPort,
			LeaderElection:     false,
		})
		if err != nil {
			setupLog.Error(err, "unable to start manager")
			os.Exit(1)
		}

		// watch for syscalls
		stopCh := signals.SetupSignalHandler()

		var wait sync.WaitGroup

		// Start envoy's aggregated discovery service
		xdss := runADSServer(ctx, cfg, &wait, stopCh)

		// Start controllers
		runADSServerControllers(ctx, mgr, cfg, &wait, stopCh, xdss.GetSnapshotCache())

		// Start webhook server
		runWebhookServer(ctx, cfg, &wait, stopCh)

		// Wait for shutdown
		wait.Wait()
		setupLog.Info("Controller has shut down")

	}
}

func runOperator(ctx context.Context, mgr manager.Manager, stopCh <-chan struct{}) {

	if err := (&operatorcontroller.DiscoveryServiceReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("DiscoveryService"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DiscoveryService")
		os.Exit(1)
	}

	if err := (&operatorcontroller.DiscoveryServiceCertificateReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("DiscoveryServiceCertificate"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DiscoveryServiceCertificate")
		os.Exit(1)
	}

	if err := (&operatorcontroller.DiscoveryServiceCertificateWatcher{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("DiscoveryServiceCertificateWatcher"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DiscoveryServiceCertificateWatcher")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("Starting the Operator.")
	if err := mgr.Start(stopCh); err != nil {
		setupLog.Error(err, "Controller manager exited non-zero")
		os.Exit(1)
	}

}

func runADSServer(ctx context.Context, cfg *rest.Config, wait *sync.WaitGroup, stopCh <-chan struct{}) *envoy.XdsServer {
	xdss := envoy.NewXdsServer(
		ctx,
		envoyControlPlanePort,
		&tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				// Sadly, these 2 non 256 are required to use http2 in go
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
			Certificates: []tls.Certificate{getCertificate(tlsCertificatePath, tlsKeyPath, setupLog)},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    getCA(tlsCAPath, setupLog),
		},
		&envoy.Callbacks{
			OnError: envoycontroller.OnError(cfg),
		},
	)

	wait.Add(1)
	go func() {
		defer wait.Done()
		if err := xdss.Start(stopCh); err != nil {
			setupLog.Error(err, "ADS server returned an unrecoverable error, shutting down")
			os.Exit(1)
		}
	}()

	return xdss
}

func runADSServerControllers(ctx context.Context, mgr manager.Manager, cfg *rest.Config, wait *sync.WaitGroup, stopCh <-chan struct{}, c *xds_cache.SnapshotCache) {

	if err := (&envoycontroller.EnvoyConfigReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("EnvoyConfig"),
		Scheme:   mgr.GetScheme(),
		ADSCache: c,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EnvoyConfig")
		os.Exit(1)
	}

	if err := (&envoycontroller.EnvoyConfigRevisionReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("EnvoyConfigRevision"),
		Scheme:   mgr.GetScheme(),
		ADSCache: c,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EnvoyConfigRevision")
		os.Exit(1)
	}

	// Start the controllers
	wait.Add(1)
	go func() {
		defer wait.Done()
		if err := mgr.Start(stopCh); err != nil {
			setupLog.Error(err, "Controller manager exited non-zero")
			os.Exit(1)
		}
	}()

}

func runWebhookServer(ctx context.Context, cfg *rest.Config, wait *sync.WaitGroup, stopCh <-chan struct{}) {
	webhook := webhook.NewWebhookServer(
		context.TODO(),
		int32(webhookPort),
		&tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
			Certificates: []tls.Certificate{getCertificate(tlsCertificatePath, tlsKeyPath, setupLog)},
		},
	)

	wait.Add(1)
	go func() {
		defer wait.Done()
		if err := webhook.Start(stopCh); err != nil {
			setupLog.Error(err, "Webhook server returned an unrecoverable error, shutting down")
			os.Exit(1)
		}
	}()
}

func getCertificate(certPath, keyPath string, logger logr.Logger) tls.Certificate {
	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		logger.Error(err, "Could not load server certificate")
		os.Exit(1)
	}
	return certificate
}

func getCA(caPath string, logger logr.Logger) *x509.CertPool {
	certPool := x509.NewCertPool()
	if bs, err := ioutil.ReadFile(caPath); err != nil {
		logger.Error(err, "Failed to read client ca cert")
		os.Exit(1)
	} else {
		ok := certPool.AppendCertsFromPEM(bs)
		if !ok {
			logger.Error(err, "Failed to append client certs")
			os.Exit(1)
		}
	}
	return certPool
}
