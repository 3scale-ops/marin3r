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
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator.marin3r/v1alpha1"
	marin3rcontroller "github.com/3scale/marin3r/controllers/marin3r"
	operatorcontroller "github.com/3scale/marin3r/controllers/operator.marin3r"
	discoveryservice "github.com/3scale/marin3r/pkg/discoveryservice"
	"github.com/3scale/marin3r/pkg/version"
	"github.com/3scale/marin3r/pkg/webhooks/podv1mutator"
	"github.com/spf13/cobra"
	// +kubebuilder:scaffold:imports
)

// Change below variables to serve metrics on different host or port.
const (
	certificateFile    string = "tls.crt"
	certificateKeyFile string = "tls.key"
)

var (
	debug                        bool
	metricsAddr                  string
	enableLeaderElection         bool
	xdssPort                     int
	xdssTLSServerCertificatePath string
	xdssTLSCACertificatePath     string
	webhookPort                  int
	podMutatorTLSCertDir         string
	podMutatorTLSKeyName         string
	podMutatorTLSCertName        string
)

var (
	// Root command
	rootCmd = &cobra.Command{
		Use:   "marin3r",
		Short: "Lightweight, CRD based envoy control plane for kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))
			printVersion()
			fmt.Println("May the force be with you ...")
		},
	}

	// Operator subcommand
	operatorCmd = &cobra.Command{
		Use:   "operator",
		Short: "Run the operator",
		Run:   runOperator,
	}

	// Discovery service subcommand
	discoveryServiceCmd = &cobra.Command{
		Use:   "discovery-service",
		Short: "Run the discovery service",
		Run:   runDiscoveryService,
	}

	// Webhook subcommand
	podMutatorCmd = &cobra.Command{
		Use:   "pod-mutator",
		Short: "Run the Pod mutating webhook",
		Run:   runPodMutator,
	}
)

var (
	scheme   = apimachineryruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	utilruntime.Must(marin3rv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme

	// Subcommands
	rootCmd.AddCommand(operatorCmd)
	rootCmd.AddCommand(discoveryServiceCmd)
	rootCmd.AddCommand(podMutatorCmd)

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logs")
	rootCmd.PersistentFlags().StringVar(&metricsAddr, "metrics-addr", fmt.Sprintf(":%v", operatorv1alpha1.DefaultMetricsPort), "The address the metrics endpoint binds to.")

	// Operator flags
	operatorCmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	// Discovery service flags
	discoveryServiceCmd.Flags().IntVar(&xdssPort, "xdss-port", int(operatorv1alpha1.DefaultXdsServerPort), "The port where the xDS will listen.")
	discoveryServiceCmd.Flags().StringVar(&xdssTLSServerCertificatePath, "server-certificate-path", "/etc/marin3r/tls/server",
		fmt.Sprintf("The path where the server certificate '%s' and key '%s' files are located", certificateFile, certificateKeyFile))
	discoveryServiceCmd.Flags().StringVar(&xdssTLSCACertificatePath, "ca-certificate-path", "/etc/marin3r/tls/ca",
		fmt.Sprintf("The path where the CA certificate '%s' and key '%s' files are located", certificateFile, certificateKeyFile))
	discoveryServiceCmd.Flags().IntVar(&webhookPort, "webhook-port", int(operatorv1alpha1.DefaultWebhookPort), "The port where the pod mutator webhook server will listen.")

	// Webhook flags
	podMutatorCmd.Flags().IntVar(&webhookPort, "webhook-port", int(operatorv1alpha1.DefaultWebhookPort), "The port where the pod mutator webhook server will listen.")
	podMutatorCmd.Flags().StringVar(&podMutatorTLSCertDir, "tls-dir", "/apiserver.local.config/certificates", "The path where the certificate and key for the webhook are located.")
	podMutatorCmd.Flags().StringVar(&podMutatorTLSCertName, "tls-cert-name", "apiserver.crt", "The file name of the certificate for the webhook.")
	podMutatorCmd.Flags().StringVar(&podMutatorTLSKeyName, "tls-key-name", "apiserver.key", "The file name of the private key for the webhook.")

}

func main() {
	rootCmd.Execute()
}

func runOperator(cmd *cobra.Command, args []string) {

	ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))
	printVersion()

	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               webhookPort,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "2cfbe7d6.operator.marin3r.3scale.net",
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

	setupLog.Info("starting the Operator.")
	if err := mgr.Start(stopCh); err != nil {
		setupLog.Error(err, "controller manager exited non-zero")
		os.Exit(1)
	}
}

func runDiscoveryService(cmd *cobra.Command, args []string) {

	ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))
	printVersion()

	cfg := ctrl.GetConfigOrDie()
	ctx := context.Background()

	mgr := discoveryservice.Manager{
		XdsServerPort:         xdssPort,
		MetricsAddr:           metricsAddr,
		ServerCertificatePath: xdssTLSServerCertificatePath,
		CACertificatePath:     xdssTLSCACertificatePath,
		Cfg:                   cfg,
	}

	mgr.Start(ctx)
}

func runPodMutator(cmd *cobra.Command, args []string) {

	ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))
	printVersion()

	stopCh := signals.SetupSignalHandler()
	cfg := ctrl.GetConfigOrDie()

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

	// Setup the webhook
	hookServer := mgr.GetWebhookServer()
	hookServer.CertDir = podMutatorTLSCertDir
	hookServer.KeyName = podMutatorTLSKeyName
	hookServer.CertName = podMutatorTLSCertName
	hookServer.Port = webhookPort
	ctrl.Log.Info("registering the pod mutating webhook with webhook server")
	hookServer.Register(podv1mutator.MutatePath, &webhook.Admission{Handler: &podv1mutator.PodMutator{Client: mgr.GetClient()}})

	setupLog.Info("starting the Pod mutating webhook")
	if err := mgr.Start(stopCh); err != nil {
		setupLog.Error(err, "controller manager exited non-zero")
		os.Exit(1)
	}
}

func printVersion() {
	setupLog.Info(fmt.Sprintf("Marin3r Version: %s", version.Current()))
	setupLog.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}
