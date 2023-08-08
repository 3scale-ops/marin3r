package pki

import (
	"crypto/x509"
	"fmt"
	"reflect"
	"testing"

	"github.com/3scale-ops/marin3r/pkg/util/test"
)

func testValidCertificate() *x509.Certificate {
	cert, _ := LoadX509Certificate(test.TestValidCertificate())
	return cert
}

func testExpiredCertificate() *x509.Certificate {
	cert, _ := LoadX509Certificate(test.TestExpiredCertificate())
	return cert
}

func TestVerify(t *testing.T) {
	type args struct {
		certificate *x509.Certificate
		root        *x509.Certificate
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Certificate is valid",
			args: args{
				certificate: testValidCertificate(),
				root:        testIssuerCertificate(),
			},
			wantErr: false,
		},
		{
			name: "Certificate is expired",
			args: args{
				certificate: testExpiredCertificate(),
				root:        testIssuerCertificate(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Verify(tt.args.certificate, tt.args.root); (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyError_Error(t *testing.T) {
	type fields struct {
		msg string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "Returns the error message",
			fields: fields{msg: "test"},
			want:   "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vf := VerifyError{
				msg: tt.fields.msg,
			}
			if got := vf.Error(); got != tt.want {
				t.Errorf("VerifyError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewVerifyError(t *testing.T) {
	type args struct {
		msg string
	}
	tests := []struct {
		name string
		args args
		want VerifyError
	}{
		{
			name: "Returns a VerifyError",
			args: args{msg: "test"},
			want: VerifyError{msg: "test"},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewVerifyError(tt.args.msg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewVerifyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsVerifyError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Returns true",
			args: args{err: VerifyError{msg: "test"}},
			want: true,
		},
		{
			name: "Returns false",
			args: args{err: fmt.Errorf("teste")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVerifyError(tt.args.err); got != tt.want {
				t.Errorf("IsVerifyError() = %v, want %v", got, tt.want)
			}
		})
	}
}
