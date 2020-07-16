#!/bin/bash

kubectl get secrets marin3r-ca-cert-instance -ojsonpath="{.data['tls\.crt']}" | base64 -d > tmp/ca.crt
kubectl get secrets marin3r-ca-cert-instance -ojsonpath="{.data['tls\.key']}" | base64 -d > tmp/ca.key
kubectl get secrets marin3r-server-cert-instance -ojsonpath="{.data['tls\.crt']}" | base64 -d > tmp/server.crt
kubectl get secrets marin3r-server-cert-instance -ojsonpath="{.data['tls\.key']}" | base64 -d > tmp/server.key
kubectl get secrets -n test1 envoy-sidecar-client-cert -ojsonpath="{.data['tls\.crt']}" | base64 -d > tmp/client.crt
kubectl get secrets -n test1 envoy-sidecar-client-cert -ojsonpath="{.data['tls\.key']}" | base64 -d > tmp/client.key

# openssl s_server -accept 9003 -CAfile tmp/ca.crt -cert tmp/server.crt -key tmp/server.key -Verify 10 -state -quiet
# openssl s_client -connect localhost:9003 -key tmp/client.key -cert tmp/client.crt -CAfile tmp/ca.crt -state -tlsextdebug -verify 10 -quiet
