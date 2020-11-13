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
	"flag"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	envoycontroller "github.com/3scale/marin3r/controllers/envoy"
	operatorcontroller "github.com/3scale/marin3r/controllers/operator"
	discoveryservice "github.com/3scale/marin3r/pkg/discoveryservice"
	// +kubebuilder:scaffold:imports
)

// Change below variables to serve metrics on different host or port.
const (
	host               string = "0.0.0.0"
	metricsPort        int    = 8383
	webhookPort        int    = 8443
	envoyXdsServerPort int    = 18000
	certificateFile    string = "tls.crt"
	certificateKeyFile string = "tls.key"
)

var (
	tlsServerCertificatePath string
	tlsCACertificatePath     string
	isDiscoveryService       bool
	debug                    bool
	metricsAddr              string
	enableLeaderElection     bool
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
	flag.StringVar(&tlsServerCertificatePath, "server-certificate-path", "/etc/marin3r/tls/server",
		fmt.Sprintf("The path where the server certificate '%s' and key '%s' files are located", certificateFile, certificateKeyFile))
	flag.StringVar(&tlsCACertificatePath, "ca-certificate-path", "/etc/marin3r/tls/ca",
		fmt.Sprintf("The path where the CA certificate '%s' and key '%s' files are located", certificateFile, certificateKeyFile))
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

		if err = (&envoycontroller.EnvoyBootstrapReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("EnvoyBootstrap"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "EnvoyBootstrap")
			os.Exit(1)
		}
		// +kubebuilder:scaffold:builder

		setupLog.Info("Starting the Operator.")
		if err := mgr.Start(stopCh); err != nil {
			setupLog.Error(err, "Controller manager exited non-zero")
			os.Exit(1)
		}

	} else {

		mgr := discoveryservice.Manager{
			XdsServerPort:         envoyXdsServerPort,
			WebhookPort:           webhookPort,
			MetricsAddr:           metricsAddr,
			ServerCertificatePath: tlsServerCertificatePath,
			CACertificatePath:     tlsCACertificatePath,
			Cfg:                   cfg,
		}

		mgr.Start(ctx)
	}
}
