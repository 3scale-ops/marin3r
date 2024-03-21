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
	"fmt"
	"os"
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
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/webhooks/podv1mutator"
	// +kubebuilder:scaffold:imports
)

var (
	webhookPort        int
	webhookTLSCertDir  string
	webhookTLSKeyName  string
	webhookTLSCertName string
)

var (
	// Webhook subcommand
	webhookCmd = &cobra.Command{
		Use:   "webhook",
		Short: "Run the webhook server",
		Run:   runWebhook,
	}
)

var (
	webhookScheme = apimachineryruntime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(webhookScheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(webhookScheme))
	utilruntime.Must(marin3rv1alpha1.AddToScheme(webhookScheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(webhookScheme))
	// +kubebuilder:scaffold:scheme

	rootCmd.AddCommand(webhookCmd)

	// Webhook flags
	webhookCmd.Flags().IntVar(&webhookPort, "webhook-port", 9443, "The port where the pod mutator webhook server will listen.")
	webhookCmd.Flags().StringVar(&webhookTLSCertDir, "tls-dir", "/apiserver.local.config/certificates", "The path where the certificate and key for the webhook are located.")
	webhookCmd.Flags().StringVar(&webhookTLSCertName, "tls-cert-name", "apiserver.crt", "The file name of the certificate for the webhook.")
	webhookCmd.Flags().StringVar(&webhookTLSKeyName, "tls-key-name", "apiserver.key", "The file name of the private key for the webhook.")
}

func runWebhook(cmd *cobra.Command, args []string) {

	ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))
	printVersion()

	cfg := ctrl.GetConfigOrDie()

	watchNamespace, err := getWatchNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get WatchNamespace, "+
			"the webhook will watch and manage resources in all Namespaces")
	}

	options := ctrl.Options{
		Scheme:                 webhookScheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         false,
		WebhookServer: webhook.NewServer(webhook.Options{
			// Setup the webhook
			Port:     webhookPort,
			CertDir:  webhookTLSCertDir,
			CertName: webhookTLSCertName,
			KeyName:  webhookTLSKeyName,
		}),
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	}

	if strings.Contains(watchNamespace, ",") {
		setupLog.Info(fmt.Sprintf("manager in MultiNamespaced mode will be watching namespaces %q", watchNamespace))
		options.Cache = cache.Options{DefaultNamespaces: map[string]cache.Config{}}
		for _, ns := range strings.Split(watchNamespace, ",") {
			options.Cache.DefaultNamespaces[ns] = cache.Config{}
		}
	} else if watchNamespace != "" {
		setupLog.Info(fmt.Sprintf("manager in Namespaced mode will be watching namespace %q", watchNamespace))
		options.Cache = cache.Options{DefaultNamespaces: map[string]cache.Config{
			watchNamespace: {},
		}}
	} else {
		setupLog.Info("manager in Cluster scope mode will be watching all namespaces")
	}

	mgr, err := ctrl.NewManager(cfg, options)
	if err != nil {
		setupLog.Error(err, "unable to start webhook")
		os.Exit(1)
	}

	// Register the Pod mutating webhook
	hookServer := mgr.GetWebhookServer()
	ctrl.Log.Info("registering the pod mutating webhook with webhook server")
	hookServer.Register(podv1mutator.MutatePath, &webhook.Admission{
		Handler: &podv1mutator.PodMutator{
			Client:  mgr.GetClient(),
			Decoder: admission.NewDecoder(mgr.GetScheme()),
		},
	})

	// Register the EnvoyConfig v1alpha1 webhooks
	if err = (&marin3rv1alpha1.EnvoyConfig{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "EnvoyConfig", "version", "v1alpha1")
		os.Exit(1)
	}

	// Register the EnvoyDeployment validating webhook
	if err = (&operatorv1alpha1.EnvoyDeployment{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "EnvoyDeployment")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting the webhook")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "controller manager exited non-zero")
		os.Exit(1)
	}
}
