// Copyright 2020 rvazquez@redhat.com
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package main

import (
	"fmt"
	"sync"

	"github.com/3scale/marin3r/pkg/controller"
	"github.com/spf13/cobra"
)

var (
	kubeconfig         string
	tlsCertificatePath string
	tlsKeyPath         string
	tlsCAPath          string
	logLevel           string
	ooCluster          bool
	namespace          string
	mode               string

	rootCmd = &cobra.Command{
		Use:   "marin3r",
		Short: "marin3r, the simple envoy control plane",
		Run:   run,
	}
)

func init() {
	rootCmd.Flags().StringVar(&tlsCertificatePath, "certificate", "/etc/marin3r/tls/server.crt", "Server certificate")
	rootCmd.Flags().StringVar(&tlsKeyPath, "private-key", "/etc/marin3r/tls/server.key", "The private key of the server certificate")
	rootCmd.Flags().StringVar(&tlsCAPath, "ca", "/etc/marin3r/tls/ca.crt", "The CA of the server certificate")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "One of debug, info, warn, error")
	rootCmd.Flags().BoolVar(&ooCluster, "out-of-cluster", false, "Use this flag if running outside of the cluster")
	rootCmd.Flags().StringVar(&namespace, "namespace", "", "Namespace that marin3r is scoped to. Only namespace scope is supported")
	rootCmd.Flags().StringVar(&mode, "mode", "all", "marin3r mode of operation, one of: 'control-plane', 'webhook', 'all'")

	rootCmd.MarkFlagRequired("namespace")
}

func main() {

	rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) {

	logger := controller.NewLogger(logLevel)
	ctx, stopper := controller.RunSignalWatcher(logger)

	var wait sync.WaitGroup

	switch mode {
	case "all":
		wait.Add(2)
		go func() {
			defer wait.Done()
			err := controller.NewController(
				ctx, tlsCertificatePath, tlsKeyPath, tlsCAPath,
				namespace, ooCluster, stopper, logger,
			)

			if err != nil {
				panic(err)
			}
		}()

		go func() {
			defer wait.Done()
			err := controller.RunWebhook(ctx, tlsCertificatePath, tlsKeyPath, tlsCAPath,
				namespace, logger)
			if err != nil {
				panic(err)
			}
		}()
		wait.Wait()

	case "control-plane":
		if err := controller.NewController(
			ctx, tlsCertificatePath, tlsKeyPath, tlsCAPath,
			namespace, ooCluster, stopper, logger,
		); err != nil {
			panic(err)
		}

	case "webhook":
		err := controller.RunWebhook(ctx, tlsCertificatePath, tlsKeyPath, tlsCAPath,
			namespace, logger)
		if err != nil {
			panic(err)
		}

	default:
		panic(fmt.Errorf("Unsupported mode '%s'", mode))
	}

	// The signal watcher will close this channle
	// upon receiving a system signal
	<-stopper
}
