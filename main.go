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
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	marin3rcontroller "github.com/3scale/marin3r/controllers/marin3r"
	operatorcontroller "github.com/3scale/marin3r/controllers/operator"
	discoveryservice "github.com/3scale/marin3r/pkg/discoveryservice"
	"github.com/3scale/marin3r/pkg/reconcilers/lockedresources"
	"github.com/3scale/marin3r/pkg/version"
	"github.com/3scale/marin3r/pkg/webhooks/podv1mutator"
	// +kubebuilder:scaffold:imports
)

// Change below variables to serve metrics on different host or port.
const (
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	watchNamespaceEnvVar string = "WATCH_NAMESPACE"
	certificateFile      string = "tls.crt"
	certificateKeyFile   string = "tls.key"
)

var (
	debug                        bool
	metricsAddr                  string
	leaderElect                  bool
	xdssPort                     int
	xdssTLSServerCertificatePath string
	xdssTLSCACertificatePath     string
	webhookPort                  int
	webhookTLSCertDir            string
	webhookTLSKeyName            string
	webhookTLSCertName           string
	probeAddr                    string
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
	webhookCmd = &cobra.Command{
		Use:   "webhook",
		Short: "Run the Pod mutating webhook",
		Run:   runWebhook,
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
	rootCmd.AddCommand(webhookCmd)

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logs")
	rootCmd.PersistentFlags().StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metrics endpoint binds to.")
	rootCmd.PersistentFlags().StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")

	// Operator flags
	operatorCmd.Flags().BoolVar(&leaderElect, "leader-elect", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	// Discovery service flags
	discoveryServiceCmd.Flags().IntVar(&xdssPort, "xdss-port", int(operatorv1alpha1.DefaultXdsServerPort), "The port where the xDS will listen.")
	discoveryServiceCmd.Flags().StringVar(&xdssTLSServerCertificatePath, "server-certificate-path", "/etc/marin3r/tls/server",
		fmt.Sprintf("The path where the server certificate '%s' and key '%s' files are located", certificateFile, certificateKeyFile))
	discoveryServiceCmd.Flags().StringVar(&xdssTLSCACertificatePath, "ca-certificate-path", "/etc/marin3r/tls/ca",
		fmt.Sprintf("The path where the CA certificate '%s' and key '%s' files are located", certificateFile, certificateKeyFile))
	discoveryServiceCmd.Flags().IntVar(&webhookPort, "webhook-port", int(operatorv1alpha1.DefaultWebhookPort), "The port where the pod mutator webhook server will listen.")

	// Webhook flags
	webhookCmd.Flags().IntVar(&webhookPort, "webhook-port", int(operatorv1alpha1.DefaultWebhookPort), "The port where the pod mutator webhook server will listen.")
	webhookCmd.Flags().StringVar(&webhookTLSCertDir, "tls-dir", "/apiserver.local.config/certificates", "The path where the certificate and key for the webhook are located.")
	webhookCmd.Flags().StringVar(&webhookTLSCertName, "tls-cert-name", "apiserver.crt", "The file name of the certificate for the webhook.")
	webhookCmd.Flags().StringVar(&webhookTLSKeyName, "tls-key-name", "apiserver.key", "The file name of the private key for the webhook.")

}

func main() {
	rootCmd.Execute()
}

func runOperator(cmd *cobra.Command, args []string) {

	ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))
	printVersion()

	cfg := ctrl.GetConfigOrDie()

	watchNamespace, err := getWatchNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get WatchNamespace, "+
			"the manager will watch and manage resources in all Namespaces")
	}

	options := ctrl.Options{
		Scheme:                     scheme,
		MetricsBindAddress:         metricsAddr,
		HealthProbeBindAddress:     probeAddr,
		LeaderElection:             leaderElect,
		LeaderElectionID:           "2cfbe7d6.operator.marin3r.3scale.net",
		LeaderElectionResourceLock: "configmaps",
		Namespace:                  watchNamespace, // namespaced-scope when the value is not an empty string
	}

	var isClusterScoped bool
	if strings.Contains(watchNamespace, ",") {
		setupLog.Info(fmt.Sprintf("manager in MultiNamespaced mode will be watching namespaces %q", watchNamespace))
		options.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(watchNamespace, ","))
		isClusterScoped = false
	} else if watchNamespace == "" {
		setupLog.Info("manager in Cluster scope mode will be watching all namespaces")
		options.Namespace = watchNamespace
		isClusterScoped = true
	} else {
		setupLog.Info(fmt.Sprintf("manager in Namespaced mode will be watching namespace %q", watchNamespace))
		options.Namespace = watchNamespace
		isClusterScoped = false
	}

	mgr, err := ctrl.NewManager(cfg, options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := (&operatorcontroller.DiscoveryServiceReconciler{
		Reconciler: lockedresources.NewFromManager(mgr, mgr.GetEventRecorderFor("DiscoveryService"), isClusterScoped),
		Log:        ctrl.Log.WithName("controllers").WithName("discoveryservice"),
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

	if err = (&marin3rcontroller.EnvoyBootstrapReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("envoybootstrap"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "envoybootstrap")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting the Operator.")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "controller manager exited non-zero")
		os.Exit(1)
	}
}

func runDiscoveryService(cmd *cobra.Command, args []string) {

	ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))
	printVersion()

	cfg := ctrl.GetConfigOrDie()

	mgr := discoveryservice.Manager{
		Namespace:             os.Getenv("WATCH_NAMESPACE"),
		XdsServerPort:         xdssPort,
		MetricsAddr:           metricsAddr,
		ServerCertificatePath: xdssTLSServerCertificatePath,
		CACertificatePath:     xdssTLSCACertificatePath,
		Cfg:                   cfg,
	}

	// TODO: add liveness and readiness

	mgr.Start(signals.SetupSignalHandler())
}

func runWebhook(cmd *cobra.Command, args []string) {

	ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))
	printVersion()

	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     "0",
		HealthProbeBindAddress: probeAddr,
		Port:                   webhookPort,
		LeaderElection:         false,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Setup the webhook
	hookServer := mgr.GetWebhookServer()
	hookServer.CertDir = webhookTLSCertDir
	hookServer.KeyName = webhookTLSKeyName
	hookServer.CertName = webhookTLSCertName
	hookServer.Port = webhookPort
	ctrl.Log.Info("registering the pod mutating webhook with webhook server")
	hookServer.Register(podv1mutator.MutatePath, &webhook.Admission{Handler: &podv1mutator.PodMutator{Client: mgr.GetClient()}})

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting the webhook")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "controller manager exited non-zero")
		os.Exit(1)
	}
}

// getWatchNamespace returns the Namespace the operator should be watching for changes
func getWatchNamespace() (string, error) {

	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}

func printVersion() {
	setupLog.Info(fmt.Sprintf("Marin3r Version: %s", version.Current()))
	setupLog.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}
