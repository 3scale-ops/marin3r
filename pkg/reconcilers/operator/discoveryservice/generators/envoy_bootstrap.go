package generators

import (
	"fmt"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/webhooks/podv1mutator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (cfg *GeneratorOptions) EnvoyBootstrap() client.Object {

	return &marin3rv1alpha1.EnvoyBootstrap{
		TypeMeta: metav1.TypeMeta{
			Kind:       marin3rv1alpha1.EnvoyBootstrapKind,
			APIVersion: marin3rv1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.resourceName(),
			Namespace: cfg.Namespace,
			Labels:    cfg.labels(),
		},
		Spec: marin3rv1alpha1.EnvoyBootstrapSpec{
			DiscoveryService: cfg.InstanceName,
			ClientCertificate: &marin3rv1alpha1.ClientCertificate{
				Directory:  podv1mutator.DefaultEnvoyTLSBasePath,
				SecretName: podv1mutator.DefaultClientCertificate,
				Duration: metav1.Duration{
					Duration: cfg.ClientCertificateDuration,
				},
			},
			EnvoyStaticConfig: &marin3rv1alpha1.EnvoyStaticConfig{
				ConfigMapNameV2:       podv1mutator.DefaultBootstrapConfigMapV2,
				ConfigMapNameV3:       podv1mutator.DefaultBootstrapConfigMapV3,
				ConfigFile:            fmt.Sprintf("%s/%s", podv1mutator.DefaultEnvoyConfigBasePath, podv1mutator.DefaultEnvoyConfigFileName),
				ResourcesDir:          podv1mutator.DefaultEnvoyConfigBasePath,
				RtdsLayerResourceName: "runtime",
				AdminBindAddress:      "0.0.0.0:9901",
				AdminAccessLogPath:    "/dev/null",
			},
		},
	}
}
