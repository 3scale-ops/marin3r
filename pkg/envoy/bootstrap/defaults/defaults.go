package bootstrap

import "github.com/3scale-ops/marin3r/pkg/envoy"

const (
	// common defaults
	Image                           = "envoyproxy/envoy:v1.16.0"
	EnvoyConfigBasePath             = "/etc/envoy/bootstrap"
	EnvoyConfigFileName             = "config.json"
	EnvoyExtraArgs                  = ""
	EnvoyTLSBasePath                = "/etc/envoy/tls/client"
	EnvoyAPIVersion                 = string(envoy.APIv2)
	TlsCertificateSdsSecretFileName = "tls_certificate_sds_secret.yaml"

	// sidecar specific defaults
	SidecarContainerName        = "envoy-sidecar"
	SidecarBootstrapConfigMapV2 = "envoy-sidecar-bootstrap"
	SidecarBootstrapConfigMapV3 = "envoy-sidecar-bootstrap-v3"
	SidecarConfigVolume         = "envoy-sidecar-bootstrap"
	SidecarTLSVolume            = "envoy-sidecar-tls"
	SidecarClientCertificate    = "envoy-sidecar-client-cert"

	// deployment specific defaults
	DeploymentContainerName        = "envoy"
	DeploymentBootstrapConfigMapV2 = "envoy-bootstrap"
	DeploymentBootstrapConfigMapV3 = "envoy-bootstrap-v3"
	DeploymentConfigVolume         = "envoy-bootstrap"
	DeploymentTLSVolume            = "envoy-tls"
	DeploymentClientCertificate    = "envoy-client-cert"
)
