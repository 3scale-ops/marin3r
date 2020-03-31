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
	"github.com/roivaz/marin3r/pkg/controller"
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

	rootCmd.MarkFlagRequired("namespace")
}

func main() {

	rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) {
	err := controller.NewController(
		tlsCertificatePath,
		tlsKeyPath,
		tlsCAPath,
		logLevel,
		namespace,
		ooCluster,
	)
	if err != nil {
		panic(err)
	}
}
