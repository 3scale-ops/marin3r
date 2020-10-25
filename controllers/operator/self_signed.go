package controllers

import (
	"context"
	"time"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/util/pki"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *DiscoveryServiceCertificateReconciler) reconcileSelfSignedCertificate(ctx context.Context, dsc *operatorv1alpha1.DiscoveryServiceCertificate) error {

	// Fetch the certmanagerv1alpha2.Certificate instance
	secret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(),
		types.NamespacedName{
			Name:      dsc.Spec.SecretRef.Name,
			Namespace: dsc.Spec.SecretRef.Namespace,
		},
		secret)

	if err != nil {
		if errors.IsNotFound(err) {
			// Generate secret with a self signed certificate
			secret, err := genSelfSignedCertificateObject(dsc.Spec)
			if err != nil {
				return err
			}
			if err := controllerutil.SetControllerReference(dsc, secret, r.Scheme); err != nil {
				return err
			}
			if err := r.Client.Create(ctx, secret); err != nil {
				return err
			}
			r.Log.Info("Created self-signed certificate")
			return nil
		}
		return err
	}

	// Don't reconcile if renewal is disabled
	if dsc.Spec.CertificateRenewalConfig != nil && !dsc.Spec.CertificateRenewalConfig.Enabled {
		return nil
	}

	// Load the certificate
	cert, err := pki.LoadX509Certificate(secret.Data["tls.crt"])
	if err != nil {
		return err
	}

	// Check if certificate is invalid
	err = pki.Verify(cert, cert)
	if err != nil {
		r.Log.Error(err, "Invalid certificate detected")
	}

	// If certificate is invalid or has been marked for renewal, reissue it
	if err != nil || dsc.Status.Conditions.IsTrueFor(operatorv1alpha1.CertificateNeedsRenewalCondition) {
		new, err := genSelfSignedCertificateObject(dsc.Spec)
		if err != nil {
			return err
		}
		patch := client.MergeFrom(secret.DeepCopy())
		secret.Data = new.Data
		if err := r.Client.Patch(ctx, secret, patch); err != nil {
			return err
		}
		r.Log.Info("Re-issued self-signed certificate")

	}

	if dsc.Status.Conditions.IsTrueFor(operatorv1alpha1.CertificateNeedsRenewalCondition) {
		// remove the condition
		patch := client.MergeFrom(dsc.DeepCopy())
		dsc.Status.Conditions.RemoveCondition(operatorv1alpha1.CertificateNeedsRenewalCondition)
		if err := r.Client.Status().Patch(ctx, dsc, patch); err != nil {
			return err
		}
	}

	return nil
}

func genSelfSignedCertificateObject(cfg operatorv1alpha1.DiscoveryServiceCertificateSpec) (*corev1.Secret, error) {

	crt, key, err := pki.GenerateCertificate(
		nil,
		nil,
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
