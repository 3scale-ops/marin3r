package defaults

import (
	"strings"

	"github.com/3scale-ops/marin3r/pkg/envoy"
	"github.com/3scale-ops/marin3r/pkg/version"
)

const (
	// common defaults
	Image                           string = "envoyproxy/envoy:v1.18.3"
	EnvoyConfigBasePath             string = "/etc/envoy/bootstrap"
	EnvoyConfigFileName             string = "config.json"
	EnvoyExtraArgs                  string = ""
	EnvoyTLSBasePath                string = "/etc/envoy/tls/client"
	EnvoyAPIVersion                 string = string(envoy.APIv3)
	TlsCertificateSdsSecretFileName string = "tls_certificate_sds_secret.json"
	EnvoyAdminPort                  uint32 = 9901
	EnvoyAdminBindAddress           string = "0.0.0.0"
	EnvoyAdminAccessLogPath         string = "/dev/null"
	GracefulShutdownTimeoutSeconds  int64  = 300

	LivenessInitialDelaySeconds int32 = 30
	LivenessTimeoutSeconds      int32 = 1
	LivenessPeriodSeconds       int32 = 10
	LivenessSuccessThreshold    int32 = 1
	LivenessFailureThreshold    int32 = 10

	ReadinessProbeInitialDelaySeconds int32 = 15
	ReadinessProbeTimeoutSeconds      int32 = 1
	ReadinessProbePeriodSeconds       int32 = 5
	ReadinessProbeSuccessThreshold    int32 = 1
	ReadinessProbeFailureThreshold    int32 = 1

	// sidecar specific defaults
	SidecarContainerName     string = "envoy-sidecar"
	SidecarConfigVolume      string = "envoy-sidecar-bootstrap"
	SidecarTLSVolume         string = "envoy-sidecar-tls"
	SidecarClientCertificate string = "envoy-sidecar-client-cert"

	// deployment specific defaults
	DeploymentContainerName     string = "envoy"
	DeploymentConfigVolume      string = "envoy-bootstrap"
	DeploymentTLSVolume         string = "envoy-tls"
	DeploymentClientCertificate string = "envoy-client-cert"

	// init manager defaults
	InitMgrDefaultImageRegistry  string = "quay.io/3scale/marin3r"
	InitMgrRtdsLayerResourceName string = "runtime"

	// shutdown manager defaults
	ShtdnMgrDefaultImageRegistry      string = "quay.io/3scale/marin3r"
	ShtdnMgrDefaultServerPort         uint32 = 8090
	ShtdnMgrDefaultReadyFile          string = "/tmp/shutdown-ok"
	ShtdnMgrDefaultReadyCheckInterval int    = 1
	ShtdnMgrDefaultDrainCheckInterval int    = 5
	ShtdnMgrDefaultStartDrainDelay    int    = 0
	ShtdnMgrDefaultCheckDrainDelay    int    = 15
	ShtdnMgrDefaultMinOpenConnections int    = 0
	ShtdnMgrDefaultMemoryRequests     string = "25Mi"
	ShtdnMgrDefaultMemoryLimits       string = "50Mi"
	ShtdnMgrDefaultCPURequests        string = "5m"
	ShtdnMgrDefaultCPULimits          string = "50m"
)

func ShtdnMgrImage() string {
	return strings.Join([]string{ShtdnMgrDefaultImageRegistry, version.Current()}, ":")
}

func InitMgrImage() string {
	return strings.Join([]string{InitMgrDefaultImageRegistry, version.Current()}, ":")
}
