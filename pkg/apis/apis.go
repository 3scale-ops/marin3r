package apis

import (
	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
	controlplanesv1alpha1 "github.com/3scale/marin3r/pkg/apis/controlplane/v1alpha1"
	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	// AddToSchemes is used to add resources to the Scheme
	AddToSchemes runtime.SchemeBuilder = runtime.SchemeBuilder{cachesv1alpha1.SchemeBuilder.AddToScheme}
	// AddToOperatorSchemes is used to add resources to the Operator Scheme
	AddToOperatorSchemes runtime.SchemeBuilder = runtime.SchemeBuilder{
		controlplanesv1alpha1.SchemeBuilder.AddToScheme,
		certmanagerv1alpha2.SchemeBuilder.AddToScheme,
	}
)

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}

// AddToOperatorScheme adds operator Resources to the OperatorScheme
func AddToOperatorScheme(s *runtime.Scheme) error {
	return AddToOperatorSchemes.AddToScheme(s)
}
