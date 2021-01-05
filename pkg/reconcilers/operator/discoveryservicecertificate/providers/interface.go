package providers

// CertificateProvider has methods to manage certificates using a given provider
type CertificateProvider interface {
	CreateCertificate() ([]byte, []byte, error)
	GetCertificate() ([]byte, []byte, error)
	UpdateCertificate() ([]byte, []byte, error)
	VerifyCertificate() error
}
