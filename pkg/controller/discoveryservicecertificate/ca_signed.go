package discoveryservicecertificate

import (
	"context"
	"crypto/x509"
	"time"

	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/util/pki"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileDiscoveryServiceCertificate) reconcileCASignedCertificate(ctx context.Context, sdcert *operatorv1alpha1.DiscoveryServiceCertificate) error {

	cert := &corev1.Secret{}
	err := r.client.Get(ctx,
		types.NamespacedName{
			Name:      sdcert.Spec.SecretRef.Name,
			Namespace: sdcert.Spec.SecretRef.Namespace,
		},
		cert)

	if err != nil {
		if errors.IsNotFound(err) {
			// Get the CA certificate
			issuerCert, issuerKey, err := r.getCACertificate(ctx, sdcert.Spec)
			if err != nil {
				return err
			}
			// Generate secret with a self signed certificate
			cert, err := genCASignedCertificateObject(sdcert.Spec, issuerCert, issuerKey)
			if err != nil {
				return err
			}
			// Set DiscoveryServiceCertificate instance as the owner and controller
			// unless it is a CA certificate, in which case we do not want garbage collection
			// to occur, nor do we want it to be reconciled after initial creation
			if !sdcert.Spec.IsCA {
				if err := controllerutil.SetControllerReference(sdcert, cert, r.scheme); err != nil {
					return err
				}
			}
			// Write the object to the api
			if err := r.client.Create(ctx, cert); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	// TODO: reconcile (reissue) certificate when condition "NotValid" is

	return nil
}

func (r *ReconcileDiscoveryServiceCertificate) getCACertificate(ctx context.Context, cfg operatorv1alpha1.DiscoveryServiceCertificateSpec) (*x509.Certificate, interface{}, error) {

	s := &corev1.Secret{}
	err := r.client.Get(
		ctx,
		types.NamespacedName{
			Name:      cfg.Signer.CASigned.SecretRef.Name,
			Namespace: cfg.Signer.CASigned.SecretRef.Namespace,
		},
		s,
	)

	if err != nil {
		return nil, nil, err
	}

	cert, err := pki.LoadX509Certificate(s.Data["tls.crt"])
	if err != nil {
		return nil, nil, err
	}

	key, err := pki.DecodePrivateKeyBytes(s.Data["tls.key"])
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

func genCASignedCertificateObject(cfg operatorv1alpha1.DiscoveryServiceCertificateSpec, issuerCert *x509.Certificate, issuerKey interface{}) (*corev1.Secret, error) {

	crt, key, err := pki.GenerateCertificate(
		issuerCert,
		issuerKey,
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
