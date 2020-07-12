package discoveryservicecertificate

import (
	"context"
	"crypto/x509"
	"time"

	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/util/pki"
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileDiscoveryServiceCertificate) reconcileCASignedCertificate(ctx context.Context, sdc *operatorv1alpha1.DiscoveryServiceCertificate) error {

	// Get the issuer certificate
	issuerCert, issuerKey, err := r.getCACertificate(ctx, sdc.Spec)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{}
	err = r.client.Get(ctx,
		types.NamespacedName{
			Name:      sdc.Spec.SecretRef.Name,
			Namespace: sdc.Spec.SecretRef.Namespace,
		},
		secret)

	if err != nil {
		if errors.IsNotFound(err) {
			secret, err := genCASignedCertificateObject(sdc.Spec, issuerCert, issuerKey)
			if err != nil {
				return err
			}
			if err := controllerutil.SetControllerReference(sdc, secret, r.scheme); err != nil {
				return err
			}
			if err := r.client.Create(ctx, secret); err != nil {
				return err
			}
			r.logger.Info("Created ca-signed certificate")
			return nil
		}
		return err
	}

	// Load the certificate
	cert, err := pki.LoadX509Certificate(secret.Data["tls.crt"])
	if err != nil {
		return err
	}

	// Reconcile only when we detect the certificate is invalid
	err = pki.Verify(cert, issuerCert)
	if err != nil {
		r.logger.Error(err, "Invalid certificate detected")
		new, err := genCASignedCertificateObject(sdc.Spec, issuerCert, issuerKey)
		if err != nil {
			return err
		}
		patch := client.MergeFrom(secret.DeepCopy())
		secret.Data = new.Data
		if err := r.client.Patch(ctx, secret, patch); err != nil {
			return err
		}
		r.logger.Info("Re-issued ca-signed certificate")
	}

	if sdc.Status.Conditions.IsTrueFor(operatorv1alpha1.CertificateNotValidCondition) {

		// remove the condition
		patch := client.MergeFrom(sdc.DeepCopy())
		sdc.Status.Conditions.SetCondition(status.Condition{
			Type:    operatorv1alpha1.CertificateNotValidCondition,
			Status:  corev1.ConditionFalse,
			Reason:  status.ConditionReason("CerificateReissued"),
			Message: "Certificate has been reissued",
		})
		if err := r.client.Status().Patch(ctx, sdc, patch); err != nil {
			return err
		}
	}

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
