package reconcilers

import (
	"testing"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"k8s.io/utils/pointer"
)

func TestIsInitialized(t *testing.T) {
	type args struct {
		dsc *operatorv1alpha1.DiscoveryServiceCertificate
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Returns true, no initialization required",
			args: args{
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						IsServerCertificate:      pointer.BoolPtr(false),
						IsCA:                     pointer.BoolPtr(false),
						Hosts:                    []string{"host"},
						CertificateRenewalConfig: &operatorv1alpha1.CertificateRenewalConfig{Enabled: true},
					},
				},
			},
			want: true,
		},
		{
			name: "Returns false, IsServerCertificate requires init",
			args: args{
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						IsCA:                     pointer.BoolPtr(false),
						Hosts:                    []string{"host"},
						CertificateRenewalConfig: &operatorv1alpha1.CertificateRenewalConfig{Enabled: true},
					},
				},
			},
			want: false,
		},
		{
			name: "Returns false, IsCA requires init",
			args: args{
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						IsServerCertificate:      pointer.BoolPtr(false),
						Hosts:                    []string{"host"},
						CertificateRenewalConfig: &operatorv1alpha1.CertificateRenewalConfig{Enabled: true},
					},
				},
			},
			want: false,
		},
		{
			name: "Returns false, Hosts requires init",
			args: args{
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						CommonName:               "test",
						IsServerCertificate:      pointer.BoolPtr(false),
						IsCA:                     pointer.BoolPtr(false),
						CertificateRenewalConfig: &operatorv1alpha1.CertificateRenewalConfig{Enabled: true},
					},
				},
			},
			want: false,
		},
		{
			name: "Returns false, CertificateRenewalConfig requires init",
			args: args{
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					Spec: operatorv1alpha1.DiscoveryServiceCertificateSpec{
						IsServerCertificate: pointer.BoolPtr(false),
						IsCA:                pointer.BoolPtr(false),
						Hosts:               []string{"host"},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInitialized(tt.args.dsc); got != tt.want {
				t.Errorf("IsInitialized() = %v, want %v", got, tt.want)
			}
		})
	}
}
