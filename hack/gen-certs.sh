#!/bin/bash

CERTS_BASE_PATH=tmp/certs
CA_CERT_PATH=${CERTS_BASE_PATH}/ca
SERVER_CERT_PATH=${CERTS_BASE_PATH}/server
ENVOY_CERT=${CERTS_BASE_PATH}/envoy-client

mkdir -p ${CA_CERT_PATH}
go run hack/gen_cert.go \
    --not-before=$(date '+%Y-%m-%dT%H:%M:%SZ') \
    --not-after=$(date '+%Y-%m-%dT%H:%M:%SZ' -d '+10 years') \
    --is-ca-certificate=true \
    --common-name=ca \
    --out ${CA_CERT_PATH}/tls

mkdir -p ${SERVER_CERT_PATH}
go run hack/gen_cert.go \
    --not-before=$(date '+%Y-%m-%dT%H:%M:%SZ') \
    --not-after=$(date '+%Y-%m-%dT%H:%M:%SZ' -d '+10 years') \
    --is-server-certificate=true \
    --common-name=marin3r.default.svc \
    --signer-cert=${CA_CERT_PATH}/tls.crt \
    --signer-key=${CA_CERT_PATH}/tls.key \
    --out ${SERVER_CERT_PATH}/tls

go run hack/gen_cert.go \
    --not-before=$(date '+%Y-%m-%dT%H:%M:%SZ') \
    --not-after=$(date '+%Y-%m-%dT%H:%M:%SZ' -d '+10 years') \
    --signer-cert=${CA_CERT_PATH}/tls.crt \
    --common-name=envoy-client \
    --signer-key=${CA_CERT_PATH}/tls.key \
    --out ${ENVOY_CERT}
