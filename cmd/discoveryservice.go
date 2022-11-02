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

package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"syscall"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	marin3rcontroller "github.com/3scale-ops/marin3r/controllers/marin3r"
	"github.com/3scale-ops/marin3r/pkg/discoveryservice"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	certificateFile    string = "tls.crt"
	certificateKeyFile string = "tls.key"
)

var (
	// Discovery service subcommand
	discoveryServiceCmd = &cobra.Command{
		Use:   "discovery-service",
		Short: "Run the discovery service",
		Run:   runDiscoveryService,
	}

	xdssPort                     int
	xdssTLSServerCertificatePath string
	xdssTLSCACertificatePath     string
	dsScheme                     = apimachineryruntime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(dsScheme))
	utilruntime.Must(marin3rv1alpha1.AddToScheme(dsScheme))
	// +kubebuilder:scaffold:scheme

	rootCmd.AddCommand(discoveryServiceCmd)

	// Discovery service flags
	discoveryServiceCmd.Flags().IntVar(&xdssPort, "xdss-port", int(operatorv1alpha1.DefaultXdsServerPort), "The port where the xDS will listen.")
	discoveryServiceCmd.Flags().StringVar(&xdssTLSServerCertificatePath, "server-certificate-path", "/etc/marin3r/tls/server",
		fmt.Sprintf("The path where the server certificate '%s' and key '%s' files are located", certificateFile, certificateKeyFile))
	discoveryServiceCmd.Flags().StringVar(&xdssTLSCACertificatePath, "ca-certificate-path", "/etc/marin3r/tls/ca",
		fmt.Sprintf("The path where the CA certificate '%s' and key '%s' files are located", certificateFile, certificateKeyFile))

}

func runDiscoveryService(cmd *cobra.Command, args []string) {

	ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))
	printVersion()

	cfg := ctrl.GetConfigOrDie()
	ctx := signals.SetupSignalHandler()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             dsScheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     false,
		Namespace:          os.Getenv("WATCH_NAMESPACE"),
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// watch for syscalls
	stopCh := setupSignalHandler()

	var wait sync.WaitGroup

	// Start envoy's aggregated discovery service
	xdss := discoveryservice.NewXdsServer(
		ctx,
		uint(xdssPort),
		&tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				// Sadly, these 2 non 256 are required to use http2 in go
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
			Certificates: []tls.Certificate{loadCertificate(xdssTLSServerCertificatePath, setupLog)},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    loadCA(xdssTLSCACertificatePath, setupLog),
		},
		setupLog,
	)

	wait.Add(1)
	go func() {
		defer wait.Done()
		client, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			setupLog.Error(err, "unable to create k8s client for xdss")
			os.Exit(1)
		}
		if err := xdss.Start(client, os.Getenv("WATCH_NAMESPACE"), stopCh); err != nil {
			setupLog.Error(err, "xDS server returned an unrecoverable error, shutting down")
			os.Exit(1)
		}
	}()

	// Start controllers
	if err := (&marin3rcontroller.EnvoyConfigReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("envoyconfig"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "envoyconfig")
		os.Exit(1)
	}

	if err := (&marin3rcontroller.EnvoyConfigRevisionReconciler{
		Client:         mgr.GetClient(),
		Log:            ctrl.Log.WithName("controllers").WithName(fmt.Sprintf("envoyconfigrevision_%s", string(envoy.APIv3))),
		Scheme:         mgr.GetScheme(),
		XdsCache:       xdss.GetCache(envoy.APIv3),
		APIVersion:     envoy.APIv3,
		DiscoveryStats: xdss.GetDiscoveryStats(envoy.APIv3),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", fmt.Sprintf("envoyconfigrevision_%s", string(envoy.APIv3)))
		os.Exit(1)
	}

	// Start the controllers
	wait.Add(1)
	go func() {
		defer wait.Done()
		if err := mgr.Start(ctx); err != nil {
			setupLog.Error(err, "Controller manager exited non-zero")
			os.Exit(1)
		}
	}()

	// Wait for shutdown
	wait.Wait()
	setupLog.Info("Controller has shut down")
}

func loadCertificate(directory string, logger logr.Logger) tls.Certificate {
	certificate, err := tls.LoadX509KeyPair(
		fmt.Sprintf("%s/%s", directory, certificateFile),
		fmt.Sprintf("%s/%s", directory, certificateKeyFile),
	)
	if err != nil {
		logger.Error(err, "Could not load server certificate")
		os.Exit(1)
	}
	return certificate
}

func loadCA(directory string, logger logr.Logger) *x509.CertPool {
	certPool := x509.NewCertPool()
	if bs, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", directory, certificateFile)); err != nil {
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

var onlyOneSignalHandler = make(chan struct{})

// SetupSignalHandler registers for SIGTERM and SIGINT. A stop channel is returned
// which is closed on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func setupSignalHandler() (stopCh <-chan struct{}) {
	close(onlyOneSignalHandler) // panics when called twice

	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}
