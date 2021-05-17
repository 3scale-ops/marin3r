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

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/discoveryservice"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	certificateFile    string = "tls.crt"
	certificateKeyFile string = "tls.key"
)

var (
	xdssPort                     int
	xdssTLSServerCertificatePath string
	xdssTLSCACertificatePath     string
)

var (
	// Discovery service subcommand
	discoveryServiceCmd = &cobra.Command{
		Use:   "discovery-service",
		Short: "Run the discovery service",
		Run:   runDiscoveryService,
	}
)

func init() {

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
