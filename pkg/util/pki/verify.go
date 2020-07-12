package pki

import (
	"crypto/x509"
)

// Verify validates that the given certificate is valid and signed by the given root
func Verify(certificate, root *x509.Certificate) error {

	roots := x509.NewCertPool()
	roots.AddCert(root)

	opts := x509.VerifyOptions{
		Roots: roots,
	}

	_, err := certificate.Verify(opts)
	if err != nil {
		return err
	}

	return nil
}
