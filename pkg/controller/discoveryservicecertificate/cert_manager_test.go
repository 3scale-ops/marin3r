package discoveryservicecertificate

import (
	"context"
	"reflect"
	"testing"

	controlplanev1alpha1 "github.com/3scale/marin3r/pkg/apis/controlplane/v1alpha1"
	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
)

func TestReconcileDiscoveryServiceCertificate_reconcileCertManagerCertificate(t *testing.T) {
	type args struct {
		ctx    context.Context
		sdcert *controlplanev1alpha1.DiscoveryServiceCertificate
	}
	tests := []struct {
		name    string
		r       *ReconcileDiscoveryServiceCertificate
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.r.reconcileCertManagerCertificate(tt.args.ctx, tt.args.sdcert); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileDiscoveryServiceCertificate.reconcileCertManagerCertificate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconcileDiscoveryServiceCertificate_hasCertManagerCertificate(t *testing.T) {
	tests := []struct {
		name    string
		r       *ReconcileDiscoveryServiceCertificate
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.hasCertManagerCertificate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileDiscoveryServiceCertificate.hasCertManagerCertificate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReconcileDiscoveryServiceCertificate.hasCertManagerCertificate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_genCertManagerCertificateObject(t *testing.T) {
	type args struct {
		cfg controlplanev1alpha1.DiscoveryServiceCertificateSpec
	}
	tests := []struct {
		name string
		args args
		want *certmanagerv1alpha2.Certificate
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := genCertManagerCertificateObject(tt.args.cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("genCertManagerCertificateObject() = %v, want %v", got, tt.want)
			}
		})
	}
}
