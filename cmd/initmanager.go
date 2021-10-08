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
	"net"
	"os"
	"strconv"
	"strings"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_bootstrap "github.com/3scale-ops/marin3r/pkg/envoy/bootstrap"
	envoy_bootstrap_options "github.com/3scale-ops/marin3r/pkg/envoy/bootstrap/options"
	"github.com/3scale-ops/marin3r/pkg/envoy/container/defaults"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	initmgrNodeID                   string
	initmgrCluster                  string
	initmgrXdsHost                  string
	initmgrXdsPort                  int
	initmgrConfigPath               string
	initmgrSdsConfigSourcePath      string
	initmgrXdsClientCertificatePath string
	initmgrRtdsLayerResourceName    string
	initmgrAdminBindAddress         string
	initmgrAdminAccessLogPath       string
	initmgrAPIVersion               string
	initmgrEnvoyImage               string
)

var (
	// Discovery service subcommand
	initManagerServiceCmd = &cobra.Command{
		Use:   "init-manager",
		Short: "Run the init manager to generate an envoy bootstrap configuration",
		Run:   runInitManager,
	}
)

func init() {

	rootCmd.AddCommand(initManagerServiceCmd)

	// Init manager flags
	initManagerServiceCmd.Flags().StringVar(&initmgrNodeID, "node-id", "", "The 'node-id' that identifies this client to the xDS server.")
	initManagerServiceCmd.Flags().StringVar(&initmgrCluster, "cluster", "", "Identifies this cluster to the xDS server.")
	initManagerServiceCmd.Flags().StringVar(&initmgrXdsHost, "xdss-host", "", "Address of the xDS server.")
	initManagerServiceCmd.Flags().IntVar(&initmgrXdsPort, "xdss-port", int(operatorv1alpha1.DefaultXdsServerPort), "The port port the xDS server.")
	initManagerServiceCmd.Flags().StringVar(&initmgrConfigPath, "config-file", fmt.Sprintf("%s/%s", defaults.EnvoyConfigBasePath, defaults.EnvoyConfigFileName), "Path to the xDS client certificate key.")
	initManagerServiceCmd.Flags().StringVar(&initmgrSdsConfigSourcePath, "resources-path", defaults.EnvoyConfigBasePath, "Path to the xDS client certificate key.")
	initManagerServiceCmd.Flags().StringVar(&initmgrXdsClientCertificatePath, "client-certificate-path", defaults.EnvoyTLSBasePath, "Path to the xDS client certificate and key.")
	initManagerServiceCmd.Flags().StringVar(&initmgrRtdsLayerResourceName, "rtds-resource-name", defaults.InitMgrRtdsLayerResourceName, "Name of the 'Runtime' resource to request from the xDS server.")
	initManagerServiceCmd.Flags().StringVar(&initmgrAdminBindAddress, "admin-bind-address", fmt.Sprintf("%s:%d", defaults.EnvoyAdminBindAddress, defaults.EnvoyAdminPort), "Address to bind the admin port to.")
	initManagerServiceCmd.Flags().StringVar(&initmgrAdminAccessLogPath, "admin-access-log-path", defaults.EnvoyAdminAccessLogPath, "Path for the admin access logs.")
	initManagerServiceCmd.Flags().StringVar(&initmgrAPIVersion, "api-version", "v3", "Envoy API version to use.")
	initManagerServiceCmd.Flags().StringVar(&initmgrEnvoyImage, "envoy-image", "", "Envoy image being used.")
}

func runInitManager(cmd *cobra.Command, args []string) {

	ctrl.SetLogger(zap.New(zap.UseDevMode(debug)))
	printVersion()

	if initmgrXdsHost == "" {
		err := fmt.Errorf("cannot be empty")
		setupLog.Error(err, "error parsing '--xdss-host'")
		os.Exit(-1)
	}

	host, port, err := parseBindAddress(initmgrAdminBindAddress)
	if err != nil {
		setupLog.Error(err, "error parsing '--admin-bind-address' flag")
		os.Exit(-1)
	}

	envoyAPI, err := envoy.ParseAPIVersion(initmgrAPIVersion)
	if err != nil {
		setupLog.Error(err, "error parsing '--api-version' flag")
		os.Exit(-1)
	}

	bootstrap := envoy_bootstrap.NewConfig(envoyAPI, envoy_bootstrap_options.ConfigOptions{
		NodeID:                      initmgrNodeID,
		Cluster:                     initmgrCluster,
		XdsHost:                     initmgrXdsHost,
		XdsPort:                     uint32(initmgrXdsPort),
		XdsClientCertificatePath:    fmt.Sprintf("%s/%s", initmgrXdsClientCertificatePath, corev1.TLSCertKey),
		XdsClientCertificateKeyPath: fmt.Sprintf("%s/%s", initmgrXdsClientCertificatePath, corev1.TLSPrivateKeyKey),
		SdsConfigSourcePath:         fmt.Sprintf("%s/%s", initmgrSdsConfigSourcePath, envoy_bootstrap_options.TlsCertificateSdsSecretFileName),
		RtdsLayerResourceName:       initmgrRtdsLayerResourceName,
		AdminAddress:                host,
		AdminPort:                   port,
		AdminAccessLogPath:          initmgrAdminAccessLogPath,
		Metadata: map[string]string{
			"pod_name":      os.Getenv("POD_NAME"),
			"pod_namespace": os.Getenv("POD_NAMESPACE"),
			"host_name":     os.Getenv("HOST_NAME"),
			"envoy_image":   initmgrEnvoyImage,
		},
	})

	config, err := bootstrap.GenerateStatic()
	if err != nil {
		setupLog.Error(err, "Error generating envoy config'")
		os.Exit(-1)
	}

	sdsResources, err := bootstrap.GenerateSdsResources()
	if err != nil {
		setupLog.Error(err, "Error generating envoy client certificate sds config'")
		os.Exit(-1)
	}

	// Write static config the file
	cf, err := os.Create(initmgrConfigPath)
	if err != nil {
		setupLog.Error(err, "")
		os.Exit(-1)
	}
	defer cf.Close()

	_, err = cf.WriteString(config)
	if err != nil {
		setupLog.Error(err, "")
		os.Exit(-1)
	}
	setupLog.Info("Config succesfully generated", "config", config)
	setupLog.Info(fmt.Sprintf("Created file '%s' with config", initmgrConfigPath))

	// Write the resource files
	for file, contents := range sdsResources {
		rf, err := os.Create(fmt.Sprintf("%s/%s", initmgrSdsConfigSourcePath, file))
		if err != nil {
			setupLog.Error(err, "")
			os.Exit(-1)
		}
		defer rf.Close()

		_, err = rf.WriteString(contents)
		if err != nil {
			setupLog.Error(err, "")
			os.Exit(-1)
		}
		rf.Close()
		setupLog.Info("Config succesfully generated", "config", contents)
		setupLog.Info(fmt.Sprintf("Created file '%s/%s' with config", initmgrSdsConfigSourcePath, file))
	}

}

func parseBindAddress(address string) (string, uint32, error) {

	var err error
	var host string
	var port int

	var parts []string
	if parts = strings.Split(address, ":"); len(parts) != 2 {
		return "", 0, fmt.Errorf("wrong 'spec.envoyStaticConfig.adminBindAddress' specification, expected '<ip>:<port>'")
	}

	host = parts[0]
	if net.ParseIP(host) == nil {
		err := fmt.Errorf("ip address %s is invalid", host)
		return "", 0, err
	}

	if port, err = strconv.Atoi(parts[1]); err != nil {
		return "", 0, fmt.Errorf("unable to parse port value in 'spec.envoyStaticConfig.adminBindAddress'")
	}

	return host, uint32(port), nil
}
