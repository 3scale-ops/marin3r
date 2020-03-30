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
)

var rootCmd = &cobra.Command{
	Use:   "marin3r",
	Short: "marin3r, the simple envoy control plane",
	Run:   run,
}

func init() {
	rootCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
	rootCmd.Flags().StringVar(&tlsCertificatePath, "certificate", "", "Server certificate")
	rootCmd.Flags().StringVar(&tlsKeyPath, "private-key", "", "The private key of the server certificate")
	rootCmd.Flags().StringVar(&tlsCAPath, "ca", "", "The CA of the server certificate")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "One of debug, info, warn, error")

	rootCmd.MarkFlagRequired("certificate")
	rootCmd.MarkFlagRequired("private-key")
	rootCmd.MarkFlagRequired("ca")
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
	)
	if err != nil {
		panic(err)
	}
}
