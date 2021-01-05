package pki

import (
	"testing"
)

func testCertificate() []byte {
	return []byte(`
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
}

func testPrivateKey() []byte {
	return []byte(`
-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCvHkAScfh7EQ3q
TDq252uZLeaXTcsf7hfNdQwcAnCUJhQONA27j91vOWEiKOjp2/ZPbShS5xxXt1Gr
rc+DJ2DJfL8odJ/XdN5Q/W84l7UxnpcxlPUofoXIjccssWAEQDGGFwdxJ0WbGw+8
KGsZkE9oYdLKp3gXmKPk1knxdonEUqJy1bRAmJLaI5BR50bWrTQbfgtASRjxx59X
7StLJLPBtG6YQCDY+qdhJnDqNRX3C0O9pH8ujb1J2A7T2sGMZ2H7kt4C15vcIwzU
wZ3ASKGQah1YIxC8P+1PxRh6wsptXLpOHafkkBIecY4xdljfgJG8ujbsOV4efyL1
fVqBsejLAgMBAAECggEBAKFCyx/xQjAaSWUsY5lRTR2XTlSg99/kgpjaI5ebi45x
7mASAV9qoTCud0tcvve0boM/8zU8zDXeg+ADxCdz2MZTETmxBA0F+0A70fMciWgz
Meof0Q9ES/Ce1v7KLLBdaP2hMWI7Fvc5mWjdE5tL8EQgaVvjkkywbKTrzNYGIeHd
7Nol0mTjLMgut+FX00AZx2KcFjNyC5vqyhKYULxMySnnQhngSjSTGVdotZnNprfp
+8MOZf6T0r6AN+OgCum06i63JhNkzNK0t+BVmrBSGXvDVnqK3RYWbTYx/s/8liIo
w6mZAOHTx9qkNHWSE6i39WbjOUhztRZ6II6eUr8rZLkCgYEA2IBI6h6+kGyhniq2
R4YVngsLVhAVm8AgC+r+/KN4kaO7C7u8f6deATnnLI/FzKXsnPHkcUc8u6iXmivS
14WpamGmsyRGvyLLsqaaZtf6jUVG+58pXCV+QIKiDaamNsZ8mRg1PWgVcqq42Jbo
BN433FCAZQUh2muOjN/3fyoH0RcCgYEAzxErRXceb6qqHJrz6YH9BhXuSD4XlxPx
Zq9gmtP8HIX9tn0qB1en4b6zxg71V/qCyCY/21qBM2+z7yhQM5bvoZqAk/jBbjRi
9bHgqwugtzK6+e0PRcd57VNi6W2kzabaz6r9F8vanPKS59QFtbifanYek/ovOV9a
aoRgLdtqbm0CgYAEx7hUav9cIvniiyDhLWW2ypmiedJwUOqkOLkOjPFxjcLofGmq
C+D4d/XRtw7v+M3jnTelBKSjpBJM1iDen1XhQmyy0d86AyOqOyF3mdcvXVM25Qm9
vhouhHPdh0tuNC22F6G9TFoE4R4ZsiNHUDy9gY2ELXvU3cEU/TDyvtPTWwKBgAMo
1+gvcR9zEzVsh9xAR4QYQZKIoAOGImDWvDqgkXA9+ykVr9Z81+rx5fxXrhaxk91J
+B94ug/23GAB1Xd0DiQBH4UifpEX64qkNDFn9APXmlLF8z21VX7xjsjRC3q32Q7i
JQp/6c4LRYKUEaI8NvKA6uaHIsFVWyPU8ULB3lXhAoGAIsIoLKqdxaLXu0jHsjEn
PFkgplxnWRhTj+ekm6Y0i4AMOw5w6MXNLIzzDgraj4SRE4V2kpV0ZqmwaI9m8WWN
z1JwIKiafYO1l4L1lRWR9IgYQ5IgpHqctw+dSRgUUVp/ERnJspaLXoVc/lGvOVtm
AjxU49gjXbI3sZeenl3gbDM=
-----END PRIVATE KEY-----
`)
}

func TestLoadX509Certificate(t *testing.T) {
	type args struct {
		cert []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Loads the certificate",
			args:    args{cert: testCertificate()},
			wantErr: false,
		},
		{
			name:    "Returns error",
			args:    args{cert: testPrivateKey()},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadX509Certificate(tt.args.cert)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadX509Certificate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDecodePrivateKeyBytes(t *testing.T) {
	type args struct {
		keyBytes []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Loads the private key",
			args:    args{keyBytes: testPrivateKey()},
			wantErr: false,
		},
		{
			name:    "Returns an error",
			args:    args{keyBytes: testCertificate()},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodePrivateKeyBytes(tt.args.keyBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodePrivateKeyBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
