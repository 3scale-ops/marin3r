package bootstrap

import "github.com/3scale-ops/marin3r/pkg/envoy"

const (
	// common defaults
	Image                           string = "envoyproxy/envoy:v1.16.0"
	EnvoyConfigBasePath             string = "/etc/envoy/bootstrap"
	EnvoyConfigFileName             string = "config.json"
	EnvoyExtraArgs                  string = ""
	EnvoyTLSBasePath                string = "/etc/envoy/tls/client"
	EnvoyAPIVersion                 string = string(envoy.APIv2)
	TlsCertificateSdsSecretFileName string = "tls_certificate_sds_secret.yaml"
	EnvoyAdminPort                  uint32 = 9901
	EnvoyAdminAccessLogPath         string = "/dev/null"

	// sidecar specific defaults
	SidecarContainerName        string = "envoy-sidecar"
	SidecarBootstrapConfigMapV2 string = "envoy-sidecar-bootstrap"
	SidecarBootstrapConfigMapV3 string = "envoy-sidecar-bootstrap-v3"
	SidecarConfigVolume         string = "envoy-sidecar-bootstrap"
	SidecarTLSVolume            string = "envoy-sidecar-tls"
	SidecarClientCertificate    string = "envoy-sidecar-client-cert"

	// deployment specific defaults
	DeploymentContainerName        string = "envoy"
	DeploymentBootstrapConfigMapV2 string = "envoy-bootstrap"
	DeploymentBootstrapConfigMapV3 string = "envoy-bootstrap-v3"
	DeploymentConfigVolume         string = "envoy-bootstrap"
	DeploymentTLSVolume            string = "envoy-tls"
	DeploymentClientCertificate    string = "envoy-client-cert"
)
