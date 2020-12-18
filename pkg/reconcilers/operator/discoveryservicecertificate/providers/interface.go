package providers

import (
	"crypto/x509"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
)

// CertificateProvider has methods to manage certificates using a given provider
type CertificateProvider interface {
	CreateCertificate(operatorv1alpha1.DiscoveryServiceCertificate, x509.Certificate) error
	GetCertificate(operatorv1alpha1.DiscoveryServiceCertificate) (*x509.Certificate, error)
	UpdateCertificate(operatorv1alpha1.DiscoveryServiceCertificate, x509.Certificate)
}
