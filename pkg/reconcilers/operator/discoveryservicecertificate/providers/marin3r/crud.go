package providers

import (
	"context"
	"crypto/x509"
	"fmt"
	"time"

	reconcilerutil "github.com/3scale-ops/basereconciler/util"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/util/pki"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	tlsCertificateKey = "tls.crt"
	tlsPrivateKeyKey  = "tls.key"
)

// CertificateProvider is the operator internal certificate
// provider
type CertificateProvider struct {
	ctx    context.Context
	logger logr.Logger
	client client.Client
	scheme *runtime.Scheme
	dsc    *operatorv1alpha1.DiscoveryServiceCertificate
}

// NewCertificateProvider returns a CertificateProvider struct for the given parameters
func NewCertificateProvider(ctx context.Context, logger logr.Logger, client client.Client,
	scheme *runtime.Scheme, dsc *operatorv1alpha1.DiscoveryServiceCertificate) *CertificateProvider {

	return &CertificateProvider{
		ctx:    ctx,
		logger: logger,
		client: client,
		scheme: scheme,
		dsc:    dsc,
	}
}

// CreateCertificate creates a certificate with the config options defined
// in the spec of the DiscoveryServiceCertificate
func (cp *CertificateProvider) CreateCertificate() ([]byte, []byte, error) {
	logger := cp.logger.WithValues("method", "CreateCertificate")

	issuerCert, issuerKey, err := cp.getIssuerCertificate()
	if err != nil {
		logger.Error(err, "unable to get issuer certificate")
		return nil, nil, err
	}

	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Name:      cp.dsc.Spec.SecretRef.Name,
		Namespace: cp.dsc.GetNamespace(),
	}
	err = cp.client.Get(cp.ctx, key, secret)

	if err == nil {
		logger.Error(fmt.Errorf("secret %s already exists", reconcilerutil.ObjectKey(secret)), "secret already exists")
		return nil, nil, fmt.Errorf("secret %s already exists", reconcilerutil.ObjectKey(secret))

	} else if !errors.IsNotFound(err) {
		logger.Error(err, "unable to get Secret")
		return nil, nil, err
	}

	secret, err = cp.genSecret(issuerCert, issuerKey)
	if err != nil {
		logger.Error(err, "unable to generate Secret for certificate")
		return nil, nil, err
	}
	if err := controllerutil.SetControllerReference(cp.dsc, secret, cp.scheme); err != nil {
		logger.Error(err, "unable to SetControllerReference on Secret for certificate")
		return nil, nil, err
	}
	if err := cp.client.Create(cp.ctx, secret); err != nil {
		logger.Error(err, "unable to create Secret for certificate")
		return nil, nil, err
	}

	logger.V(1).Info("created certificate")
	return secret.Data[tlsCertificateKey], secret.Data[tlsPrivateKeyKey], nil
}

// GetCertificate loads a certificate form the Secret referred in the
// DiscoveryServiceCertificate resource
func (cp *CertificateProvider) GetCertificate() ([]byte, []byte, error) {
	logger := cp.logger.WithValues("method", "GetCertificate")

	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Name:      cp.dsc.Spec.SecretRef.Name,
		Namespace: cp.dsc.GetNamespace(),
	}
	err := cp.client.Get(cp.ctx, key, secret)

	if err != nil {
		logger.Error(err, "unable to get Secret")
		return nil, nil, err
	}

	return secret.Data[tlsCertificateKey], secret.Data[tlsPrivateKeyKey], nil
}

// UpdateCertificate updates the certificate stored in the Secret referred
// in the DiscoveryServiceCertificate resource
func (cp *CertificateProvider) UpdateCertificate() ([]byte, []byte, error) {
	logger := cp.logger.WithValues("method", "UpdateCertificate")

	issuerCert, issuerKey, err := cp.getIssuerCertificate()
	if err != nil {
		logger.Error(err, "unable to get issuer certificate")
		return nil, nil, err
	}

	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Name:      cp.dsc.Spec.SecretRef.Name,
		Namespace: cp.dsc.GetNamespace(),
	}
	err = cp.client.Get(cp.ctx, key, secret)
	if err != nil {
		logger.Error(err, "unable to get Secret")
		return nil, nil, err
	}

	newPayload, err := cp.genSecret(issuerCert, issuerKey)
	if err != nil {
		logger.Error(err, "unable to generate new payload for certificate")
		return nil, nil, err
	}

	secret.Data = newPayload.Data
	if err := cp.client.Update(cp.ctx, secret); err != nil {
		logger.Error(err, "unable to update Secret for certificate")
		return nil, nil, err
	}

	logger.V(1).Info("updated certificate")
	return secret.Data[tlsCertificateKey], secret.Data[tlsPrivateKeyKey], nil
}

// VerifyCertificate verifies the validity of a certificate. Returns
// 'nil' if verification is correct, an error otherwise.
func (cp *CertificateProvider) VerifyCertificate() error {
	logger := cp.logger.WithValues("method", "VerifyCertificate")

	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Name:      cp.dsc.Spec.SecretRef.Name,
		Namespace: cp.dsc.GetNamespace(),
	}
	err := cp.client.Get(cp.ctx, key, secret)
	if err != nil {
		logger.Error(err, "unable to get Secret")
		return err
	}

	var cert *x509.Certificate
	cert, err = pki.LoadX509Certificate(secret.Data[tlsCertificateKey])
	if err != nil {
		logger.Error(err, "unable to load certificate from Secret")
		return err
	}

	var root *x509.Certificate
	if cp.dsc.Spec.Signer.CASigned != nil {
		root, _, err = cp.getIssuerCertificate()
		if err != nil {
			logger.Error(err, "unable to get issuer certificate")
			return err
		}
	}

	// If root is 'nil' it is a self-signed certificate that must be
	// verified against itself
	if root == nil {
		root = cert
	}

	return pki.Verify(cert, root)
}

// getIssuerCertificate returns the issuer certificate for a DiscoveryServiceCertificate resource
func (cp *CertificateProvider) getIssuerCertificate() (*x509.Certificate, interface{}, error) {

	if cp.dsc.Spec.Signer.CASigned != nil {
		secret := &corev1.Secret{}
		key := types.NamespacedName{
			Name:      cp.dsc.Spec.Signer.CASigned.SecretRef.Name,
			Namespace: cp.dsc.Spec.Signer.CASigned.SecretRef.Namespace,
		}

		if err := cp.client.Get(cp.ctx, key, secret); err != nil {
			return nil, nil, err
		}

		cert, err := pki.LoadX509Certificate(secret.Data[tlsCertificateKey])
		if err != nil {
			return nil, nil, err
		}

		privKey, err := pki.DecodePrivateKeyBytes(secret.Data[tlsPrivateKeyKey])
		if err != nil {
			return nil, nil, err
		}

		return cert, privKey, nil
	}

	return nil, nil, nil
}

func (cp *CertificateProvider) genSecret(issuerCert *x509.Certificate, issuerKey interface{}) (*corev1.Secret, error) {

	crt, key, err := pki.GenerateCertificate(
		issuerCert,
		issuerKey,
		cp.dsc.Spec.CommonName,
		time.Duration(cp.dsc.Spec.ValidFor)*time.Second,
		cp.dsc.IsServerCertificate(),
		cp.dsc.IsCA(),
		cp.dsc.GetHosts()...,
	)
	if err != nil {
		return nil, err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cp.dsc.Spec.SecretRef.Name,
			Namespace: cp.dsc.GetNamespace(),
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{tlsCertificateKey: crt, tlsPrivateKeyKey: key},
	}

	return secret, err
}
