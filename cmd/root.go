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
	"runtime"

	"github.com/3scale-ops/marin3r/pkg/version"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	watchNamespaceEnvVar string = "WATCH_NAMESPACE"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

var (
	debug       bool
	metricsAddr string
	probeAddr   string
)

var rootCmd = &cobra.Command{
	Use:   "marin3r",
	Short: "Lightweight, CRD based envoy control plane for kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))
		printVersion()
		fmt.Println("May the force be with you ...")
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logs")
	rootCmd.PersistentFlags().StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metrics endpoint binds to.")
	rootCmd.PersistentFlags().StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printVersion() {
	setupLog.Info(fmt.Sprintf("Marin3r Version: %s", version.Current()))
	setupLog.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}
