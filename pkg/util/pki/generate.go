package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

// GeneratePrivateKey generates a new RSA private key
func GeneratePrivateKey() (*rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return priv, nil
}

// GenerateCertificate issues a new certificate with the passed options and signed by the parent certificate if one is given. A self-signed
// is issued otherwise.
func GenerateCertificate(issuerCert *x509.Certificate, signerKey interface{}, commonName string, validFor time.Duration, isServer, isCA bool, host ...string) ([]byte, []byte, error) {

	priv, err := GeneratePrivateKey()
	if err != nil {
		return nil, nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"marin3r.3scale.net"},
			CommonName:   commonName,
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	for _, h := range host {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if isServer {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}

	if isCA {
		template.IsCA = true
		template.KeyUsage = x509.KeyUsageCertSign
	}

	var derBytes []byte

	if issuerCert == nil {
		// Self-signed
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
		if err != nil {
			return nil, nil, err
		}

	} else {
		// CA signed
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, issuerCert, &priv.PublicKey, signerKey)
		if err != nil {
			return nil, nil, err
		}
	}

	crtPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	return crtPEM, privPEM, nil
}
