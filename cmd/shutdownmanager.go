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
	"time"

	"github.com/3scale-ops/marin3r/pkg/envoy/container/defaults"
	"github.com/3scale-ops/marin3r/pkg/envoy/container/shutdownmanager"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	// Shutdown manager flags
	shutdownmmgrHTTPServePort      int
	shutdownmmgrReadyFile          string
	shutdownmmgrReadyCheckInterval int
	shutdownmmgrDrainCheckInterval int
	shutdownmmgrCheckDrainDelay    int
	shutdownmmgrStartDrainDelay    int
	shutdownmmgrEnvoyAdminURL      string
	shutdownmmgrMinOpenConnections int
)

var (
	// Shutdown manager subcommand
	shutdownManagerCmd = &cobra.Command{
		Use:   "shutdown-manager",
		Short: "Run envoy's shutdown manager",
		Run:   runShutdownManager,
	}
)

func init() {

	// Shutdown manager subcommand
	rootCmd.AddCommand(shutdownManagerCmd)

	// Shutdown manager flags
	shutdownManagerCmd.Flags().IntVar(&shutdownmmgrHTTPServePort, "port", int(defaults.ShtdnMgrDefaultServerPort),
		"Port for the shutdown manager to listen at")
	shutdownManagerCmd.Flags().StringVar(&shutdownmmgrReadyFile, "ready-file", defaults.ShtdnMgrDefaultReadyFile,
		"File to communicate the shutdown status between processes")
	shutdownManagerCmd.Flags().IntVar(&shutdownmmgrReadyCheckInterval, "check-ready-interval", defaults.ShtdnMgrDefaultReadyCheckInterval,
		"Polling interval to check if envoy is ready for shutdown")
	shutdownManagerCmd.Flags().IntVar(&shutdownmmgrDrainCheckInterval, "check-drain-interval", defaults.ShtdnMgrDefaultDrainCheckInterval,
		"Polling interval to check if envoy listeners have drained")
	shutdownManagerCmd.Flags().IntVar(&shutdownmmgrStartDrainDelay, "start-drain-delay", defaults.ShtdnMgrDefaultStartDrainDelay,
		"Time to wait before polling Envoy for open connections")
	shutdownManagerCmd.Flags().IntVar(&shutdownmmgrCheckDrainDelay, "check-drain-delay", defaults.ShtdnMgrDefaultCheckDrainDelay,
		"Time to wait before draining Envoy connections")
	shutdownManagerCmd.Flags().StringVar(&shutdownmmgrEnvoyAdminURL, "envoy-admin-address", fmt.Sprintf("http://localhost:%d", defaults.EnvoyAdminPort),
		"Envoy admin port address")
	shutdownManagerCmd.Flags().IntVar(&shutdownmmgrMinOpenConnections, "min-open-connections", defaults.ShtdnMgrDefaultMinOpenConnections,
		"minimum amount of connections that can be open when polling for active connections in Envoy")
}

func runShutdownManager(cmd *cobra.Command, args []string) {

	ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))
	printVersion()

	mgr := shutdownmanager.Manager{
		HTTPServePort:              shutdownmmgrHTTPServePort,
		ShutdownReadyFile:          shutdownmmgrReadyFile,
		ShutdownReadyCheckInterval: time.Duration(shutdownmmgrReadyCheckInterval) * time.Second,
		CheckDrainInterval:         time.Duration(shutdownmmgrDrainCheckInterval) * time.Second,
		CheckDrainDelay:            time.Duration(shutdownmmgrCheckDrainDelay) * time.Second,
		StartDrainDelay:            time.Duration(shutdownmmgrStartDrainDelay) * time.Second,
		EnvoyAdminAddress:          shutdownmmgrEnvoyAdminURL,
		MinOpenConnections:         shutdownmmgrMinOpenConnections,
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "shutdown manager exited non-zero")
		os.Exit(1)
	}
}
