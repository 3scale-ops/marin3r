package reconcilers

import (
	"testing"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func TestIsStatusReconciled(t *testing.T) {
	type args struct {
		dsc             *operatorv1alpha1.DiscoveryServiceCertificate
		certificateHash string
		ready           bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Returns true, status up to date",
			args: args{
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					Status: operatorv1alpha1.DiscoveryServiceCertificateStatus{
						Ready:           pointer.BoolPtr(true),
						CertificateHash: pointer.StringPtr("xxxx"),
						Conditions:      status.Conditions{},
					},
				},
				certificateHash: "xxxx",
				ready:           true,
			},
			want: true,
		},
		{
			name: "Returns false, ready needs update",
			args: args{
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					Status: operatorv1alpha1.DiscoveryServiceCertificateStatus{
						Ready:           pointer.BoolPtr(false),
						CertificateHash: pointer.StringPtr("xxxx"),
						Conditions:      status.Conditions{},
					},
				},
				certificateHash: "xxxx",
				ready:           true,
			},
			want: false,
		},
		{
			name: "Returns false, certificateHash needs update",
			args: args{
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					Status: operatorv1alpha1.DiscoveryServiceCertificateStatus{
						Ready:           pointer.BoolPtr(true),
						CertificateHash: pointer.StringPtr("xxxx"),
						Conditions:      status.Conditions{},
					},
				},
				certificateHash: "zzzz",
				ready:           true,
			},
			want: false,
		},
		{
			name: "Returns false, condition needs removal",
			args: args{
				dsc: &operatorv1alpha1.DiscoveryServiceCertificate{
					Status: operatorv1alpha1.DiscoveryServiceCertificateStatus{
						Ready:           pointer.BoolPtr(true),
						CertificateHash: pointer.StringPtr("xxxx"),
						Conditions: status.Conditions{{
							Type:   operatorv1alpha1.CertificateNeedsRenewalCondition,
							Status: corev1.ConditionTrue,
						}},
					},
				},
				certificateHash: "xxxx",
				ready:           true,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStatusReconciled(tt.args.dsc, tt.args.certificateHash, tt.args.ready); got != tt.want {
				t.Errorf("IsStatusReconciled() = %v, want %v", got, tt.want)
			}
		})
	}
}
