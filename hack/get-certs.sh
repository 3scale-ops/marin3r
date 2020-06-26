#!/bin/bash

kubectl get secrets marin3r-server-cert-instance -ojsonpath="{.data['ca\.crt']}" | base64 -d > tmp/ca.crt
kubectl get secrets marin3r-server-cert-instance -ojsonpath="{.data['tls\.crt']}" | base64 -d > tmp/server.crt
kubectl get secrets marin3r-server-cert-instance -ojsonpath="{.data['tls\.key']}" | base64 -d > tmp/server.key
kubectl get secrets -n test1 envoy-sidecar-client-cert -ojsonpath="{.data['tls\.crt']}" | base64 -d > tmp/client.crt
kubectl get secrets -n test1 envoy-sidecar-client-cert -ojsonpath="{.data['tls\.key']}" | base64 -d > tmp/client.key

# openssl s_server -accept 9003 -CAfile ca.crt -cert marin3r.default.svc.crt -key marin3r.default.svc.key -Verify 10 -state -quiet
# openssl s_client -connect localhost:9003 -key envoy-client.key -cert envoy-client.crt -CAfile ca.crt -state -tlsextdebug -verify 10 -quiet