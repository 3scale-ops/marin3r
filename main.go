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
	"runtime"

	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator.marin3r/v1alpha1"
	marin3rcontroller "github.com/3scale/marin3r/controllers/marin3r"
	operatorcontroller "github.com/3scale/marin3r/controllers/operator.marin3r"
	discoveryservice "github.com/3scale/marin3r/pkg/discoveryservice"
	"github.com/3scale/marin3r/pkg/version"
	"github.com/spf13/cobra"
	// +kubebuilder:scaffold:imports
)

// Change below variables to serve metrics on different host or port.
const (
	certificateFile    string = "tls.crt"
	certificateKeyFile string = "tls.key"
)

var (
	tlsServerCertificatePath string
	tlsCACertificatePath     string
	isDiscoveryService       bool
	debug                    bool
	metricsAddr              string
	xdssPort                 int
	webhookPort              int
	enableLeaderElection     bool
)

var rootCmd = &cobra.Command{
	Use:   "marin3r",
	Short: "marin3r, the simple envoy control plane",
	Run:   run,
}

var (
	scheme   = apimachineryruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	utilruntime.Must(marin3rv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func init() {
	rootCmd.Flags().IntVar(&xdssPort, "xdss-port", int(operatorv1alpha1.DefaultXdsServerPort), "The port where the xDS will listen.")
	rootCmd.Flags().IntVar(&webhookPort, "webhook-port", int(operatorv1alpha1.DefaultWebhookPort), "The port where the mutating webhook server will listen.")
	rootCmd.Flags().StringVar(&tlsServerCertificatePath, "server-certificate-path", "/etc/marin3r/tls/server",
		fmt.Sprintf("The path where the server certificate '%s' and key '%s' files are located", certificateFile, certificateKeyFile))
	rootCmd.Flags().StringVar(&tlsCACertificatePath, "ca-certificate-path", "/etc/marin3r/tls/ca",
		fmt.Sprintf("The path where the CA certificate '%s' and key '%s' files are located", certificateFile, certificateKeyFile))
	rootCmd.Flags().BoolVar(&isDiscoveryService, "discovery-service", false, "Run the discovery-service instead of the operator")
	rootCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug logs")
	rootCmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&metricsAddr, "metrics-addr", fmt.Sprintf(":%v", operatorv1alpha1.DefaultMetricsPort), "The address the metric endpoint binds to.")
}

func main() {

	rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) {

	ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))

	printVersion()

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
			Log:    ctrl.Log.WithName("controllers").WithName("discoveryservice"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "siscoveryservice")
			os.Exit(1)
		}

		if err := (&operatorcontroller.DiscoveryServiceCertificateReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("discoveryservicecertificate"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "discoveryservicecertificate")
			os.Exit(1)
		}

		if err := (&operatorcontroller.DiscoveryServiceCertificateWatcher{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("discoveryservicecertificatewatcher"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "discoveryservicecertificatewatcher")
			os.Exit(1)
		}

		if err = (&marin3rcontroller.EnvoyBootstrapReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("envoybootstrap"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "envoybootstrap")
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
			XdsServerPort:         xdssPort,
			WebhookPort:           webhookPort,
			MetricsAddr:           metricsAddr,
			ServerCertificatePath: tlsServerCertificatePath,
			CACertificatePath:     tlsCACertificatePath,
			Cfg:                   cfg,
		}

		mgr.Start(ctx)
	}
}

func printVersion() {
	setupLog.Info(fmt.Sprintf("Operator Version: %s", version.Current()))
	setupLog.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}
