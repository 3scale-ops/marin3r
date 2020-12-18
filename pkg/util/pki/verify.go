package pki

import (
	"crypto/x509"
)

// VerifyError is an error type returned when the
// certificate does not pass validation
type VerifyError struct {
	msg string
}

func (vf VerifyError) Error() string {
	return vf.msg
}

// NewVerifyError returns a VerifyError
func NewVerifyError(msg string) VerifyError {
	return VerifyError{msg: msg}
}

// IsVerifyError returns true if the error
// has type VerifyError
func IsVerifyError(err error) bool {
	switch err.(type) {
	case VerifyError:
		return true
	}
	return false
}

// Verify validates that the given certificate is valid and signed by the given root
func Verify(certificate, root *x509.Certificate) error {

	roots := x509.NewCertPool()
	roots.AddCert(root)

	opts := x509.VerifyOptions{
		Roots: roots,
	}

	_, err := certificate.Verify(opts)
	if err != nil {
		return NewVerifyError(err.Error())
	}

	return nil
}
