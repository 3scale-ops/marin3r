package external

import (
	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/client-go/discovery"
)

func HasCertManagerClusterIssuer(dc discovery.DiscoveryInterface) (bool, error) {
	return k8sutil.ResourceExists(
		dc,
		certmanagerv1alpha2.SchemeGroupVersion.String(),
		certmanagerv1alpha2.ClusterIssuerKind,
	)
}

func HasCertManagerCertificate(dc discovery.DiscoveryInterface) (bool, error) {
	return k8sutil.ResourceExists(
		dc,
		certmanagerv1alpha2.SchemeGroupVersion.String(),
		certmanagerv1alpha2.CertificateKind,
	)
}
