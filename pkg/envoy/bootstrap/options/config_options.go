package envoy

const (
	TlsCertificateSdsSecretFileName string = "tls_certificate_sds_secret.json"
	XdsClusterName                  string = "xds_cluster"
)

// ConfigOptions has options to configure the way the bootstrap config is generated
type ConfigOptions struct {
	NodeID                      string
	Cluster                     string
	XdsHost                     string
	XdsPort                     uint32
	XdsClientCertificatePath    string
	XdsClientCertificateKeyPath string
	SdsConfigSourcePath         string
	RtdsLayerResourceName       string
	AdminAddress                string
	AdminPort                   uint32
	AdminAccessLogPath          string
	Metadata                    map[string]string
}
