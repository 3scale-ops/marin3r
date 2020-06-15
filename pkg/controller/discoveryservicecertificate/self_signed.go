package discoveryservicecertificate

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"

	controlplanev1alpha1 "github.com/3scale/marin3r/pkg/apis/controlplane/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileDiscoveryServiceCertificate) reconcileSelfSignedCertificate(ctx context.Context, sdcert *controlplanev1alpha1.DiscoveryServiceCertificate) error {

	// Fetch the certmanagerv1alpha2.Certificate instance
	cert := &corev1.Secret{}
	err := r.client.Get(context.TODO(),
		types.NamespacedName{
			Name:      sdcert.Spec.SecretRef.Name,
			Namespace: sdcert.Spec.SecretRef.Namespace,
		},
		cert)

	if err != nil {
		if errors.IsNotFound(err) {
			// Generate secret with a self signed certificate
			cert, err := genSelfSignedCertificateObject(sdcert.Spec)
			if err != nil {
				return err
			}
			// Set DiscoveryServiceCertificate instance as the owner and controller
			if err := controllerutil.SetControllerReference(sdcert, cert, r.scheme); err != nil {
				return err
			}
			// Write the object to the api
			if err := r.client.Create(ctx, cert); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	return nil
}

func genSelfSignedCertificateObject(cfg controlplanev1alpha1.DiscoveryServiceCertificateSpec) (*corev1.Secret, error) {

	crt, key, err := genCertificate(
		cfg.CommonName,
		time.Duration(cfg.ValidFor)*time.Second,
		cfg.IsServerCertificate,
		cfg.IsCA,
		cfg.Hosts...,
	)
	if err != nil {
		return nil, err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.SecretRef.Name,
			Namespace: cfg.SecretRef.Namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{"tls.crt": crt, "tls.key": key},
	}

	return secret, err
}

func genPrivateKey() (*ecdsa.PrivateKey, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return priv, nil
}

func genCertificate(commonName string, validFor time.Duration, isServer, isCA bool, host ...string) ([]byte, []byte, error) {

	priv, err := genPrivateKey()
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

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	crtPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	return crtPEM, privPEM, nil
}
