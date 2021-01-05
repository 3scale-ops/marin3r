package pki

import (
	"crypto/x509"
	"fmt"
	"reflect"
	"testing"
)

func testValidCertificate() *x509.Certificate {
	b := []byte(`
-----BEGIN CERTIFICATE-----
MIICwjCCAiSgAwIBAgIQGdGZ18q7NFDRpp9L9S40bzAKBggqhkjOPQQDBDA7MRsw
GQYDVQQKExJtYXJpbjNyLjNzY2FsZS5uZXQxHDAaBgNVBAMTE21hcmluM3ItY2Et
aW5zdGFuY2UwHhcNMjAxMjE4MTIzNjI0WhcNMzAxMjE4MjI0ODI0WjAsMRswGQYD
VQQKExJtYXJpbjNyLjNzY2FsZS5uZXQxDTALBgNVBAMTBHRlc3QwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQCvHkAScfh7EQ3qTDq252uZLeaXTcsf7hfN
dQwcAnCUJhQONA27j91vOWEiKOjp2/ZPbShS5xxXt1Grrc+DJ2DJfL8odJ/XdN5Q
/W84l7UxnpcxlPUofoXIjccssWAEQDGGFwdxJ0WbGw+8KGsZkE9oYdLKp3gXmKPk
1knxdonEUqJy1bRAmJLaI5BR50bWrTQbfgtASRjxx59X7StLJLPBtG6YQCDY+qdh
JnDqNRX3C0O9pH8ujb1J2A7T2sGMZ2H7kt4C15vcIwzUwZ3ASKGQah1YIxC8P+1P
xRh6wsptXLpOHafkkBIecY4xdljfgJG8ujbsOV4efyL1fVqBsejLAgMBAAGjTjBM
MA4GA1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAMBgNVHRMBAf8E
AjAAMBcGA1UdEQQQMA6CDGV4YW1wbGUudGVzdDAKBggqhkjOPQQDBAOBiwAwgYcC
QgFVnME3LBIOwOCU1Pz0XRMdSMjtaIHxtSzCqlLixIJs64m5hNxmelcjWFlUayaC
hFNB+Cmsg4iE/1ttdggIh0AD7wJBSYUrlaIDPzq82PeQLvRkAPlAJlm21MCqV/kI
sgFV8+UDXCmXKZCOucPi7yD4qOsoqUHENDJ3nqOiQoIVOeC0nc8=
-----END CERTIFICATE-----
`)

	cert, _ := LoadX509Certificate(b)
	return cert
}

func testExpiredCertificate() *x509.Certificate {
	b := []byte(`
-----BEGIN CERTIFICATE-----
MIICwjCCAiSgAwIBAgIQIOyQL+7YxHIda+FqkHjYqTAKBggqhkjOPQQDBDA7MRsw
GQYDVQQKExJtYXJpbjNyLjNzY2FsZS5uZXQxHDAaBgNVBAMTE21hcmluM3ItY2Et
aW5zdGFuY2UwHhcNMjAxMjE4MTMwMTQ2WhcNMjAxMjE4MTMwMTQ3WjAsMRswGQYD
VQQKExJtYXJpbjNyLjNzY2FsZS5uZXQxDTALBgNVBAMTBHRlc3QwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQC6Lq1+onmVDTc9kbuXS5BgTy98aadxDly8
wpmcssRV0OBvIpUYhh17vcYd5t/a52XJQ3P1vT25hMv62Bhw0gokMjNAM76mBILG
BN7uK443+ob0E7Z5ZVxdzqlyUBpHMdb5XCiIbcf4U0XP5iZLF/njvZZjgDa6halh
c5whh0g7ekkhGtyQZYOI+pfdsQCVs7c0HjilIMi4CZfIwkefs1R/PQ6KZnABGUBs
k817R4aNLw54pCinYTQ4FvlxLR2A2cstR/f+CBHhcjDSUDdE25omHmRRaiIAigfl
UiKPoFYKA+kU4i69I2w4mjWTWZ7Y1qoURCXHDd34vIJyUXzx5iFLAgMBAAGjTjBM
MA4GA1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAMBgNVHRMBAf8E
AjAAMBcGA1UdEQQQMA6CDGV4YW1wbGUudGVzdDAKBggqhkjOPQQDBAOBiwAwgYcC
QXvi1o7Tk/Co7aLyQfyjxWTTmFIRbso5E+vp/v70U+RzQzJjMU9LHjLHPpaJT6hv
wC4IQVoqTFV9o0LMfmrY9tu0AkIBkSPte09kukRVH682WPLoSgMAfou9DxiwcilT
osel1VN6TUilTRCHM7UU5UItRDaEhFwxnGr9ErAmvkQIVTGSURM=
-----END CERTIFICATE-----
`)

	cert, _ := LoadX509Certificate(b)
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
