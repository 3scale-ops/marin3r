package discoveryservicecertificate

import (
	"testing"

	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
)

func Test_genSelfSignedCertificateObject(t *testing.T) {
	tests := []struct {
		name    string
		cfg     operatorv1alpha1.DiscoveryServiceCertificateSpec
		wantErr bool
	}{
		{
			name: "Generates a new self-signed client certificate",
			cfg: operatorv1alpha1.DiscoveryServiceCertificateSpec{
				CommonName: "test",
				ValidFor:   86400,
				Signer: operatorv1alpha1.DiscoveryServiceCertificateSigner{
					SelfSigned: &operatorv1alpha1.SelfSignedConfig{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := genSelfSignedCertificateObject(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("genSelfSignedCertificateObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Data["tls.crt"] == nil || got.Data["tls.key"] == nil {
				t.Errorf("genSelfSignedCertificateObject() empty crt or key in secret")
				return
			}
		})
	}
}
