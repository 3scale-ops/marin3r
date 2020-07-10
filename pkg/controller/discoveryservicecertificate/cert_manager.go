package discoveryservicecertificate

import (
	"context"
	"fmt"

	"github.com/3scale/marin3r/pkg/apis/external"
	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	// cert-manager
	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
)

func (r *ReconcileDiscoveryServiceCertificate) reconcileCertManagerCertificate(ctx context.Context, sdcert *operatorv1alpha1.DiscoveryServiceCertificate) error {

	// Validate the cert-manager apis are available
	exists, err := external.HasCertManagerCertificate(r.discoveryClient)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("certmanagerv1alpha2.CertificateKind unavailabe")
	}

	// Fetch the certmanagerv1alpha2.Certificate instance
	cert := &certmanagerv1alpha2.Certificate{}
	err = r.client.Get(ctx,
		types.NamespacedName{
			Name:      sdcert.Spec.SecretRef.Name,
			Namespace: sdcert.Spec.SecretRef.Namespace,
		},
		cert)

	if err != nil {
		if errors.IsNotFound(err) {
			cert = genCertManagerCertificateObject(sdcert.Spec)
			// Set DiscoveryServiceCertificate instance as the owner and controller
			if err := controllerutil.SetControllerReference(sdcert, cert, r.scheme); err != nil {
				return err
			}
			if err := r.client.Create(ctx, cert); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	desired := genCertManagerCertificateObject(sdcert.Spec)
	if !apiequality.Semantic.DeepEqual(cert.Spec, desired.Spec) {
		log.Info("cert-manager certificate needs update")
		patch := client.MergeFrom(cert.DeepCopy())
		cert.Spec = desired.Spec
		if err := r.client.Patch(ctx, cert, patch); err != nil {
			return err
		}
	}

	return nil
}

func genCertManagerCertificateObject(cfg operatorv1alpha1.DiscoveryServiceCertificateSpec) *certmanagerv1alpha2.Certificate {

	hosts := []string{}
	if cfg.IsServerCertificate && len(cfg.Hosts) == 0 {
		hosts = []string{cfg.CommonName}
	} else {
		hosts = cfg.Hosts
	}

	usages := []certmanagerv1alpha2.KeyUsage{
		certmanagerv1alpha2.UsageClientAuth,
	}
	if cfg.IsServerCertificate {
		usages = append(usages, certmanagerv1alpha2.UsageServerAuth)
	}

	return &certmanagerv1alpha2.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.SecretRef.Name,
			Namespace: cfg.SecretRef.Namespace,
		},
		Spec: certmanagerv1alpha2.CertificateSpec{
			CommonName: cfg.CommonName,
			SecretName: cfg.SecretRef.Name,
			IssuerRef: cmmeta.ObjectReference{
				Name: cfg.Signer.CertManager.ClusterIssuer,
				Kind: "ClusterIssuer",
			},
			DNSNames: hosts,
			Usages:   usages,
		},
	}
}
