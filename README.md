# marin3r

[![Go Report Card](https://goreportcard.com/badge/github.com/3scale/marin3r)](https://goreportcard.com/report/github.com/3scale/marin3r)
[![codecov](https://codecov.io/gh/3scale/marin3r/branch/master/graph/badge.svg)](https://codecov.io/gh/3scale/marin3r)
[![build status](https://circleci.com/gh/3scale/marin3r.svg?style=shield)](https://codecov.io/gh/3scale/marin3r/.circleci/config.yml)
[![release](https://badgen.net/github/release/3scale/marin3r)](https://github.com/3scale/marin3r/releases)
[![license](https://badgen.net/github/license/3scale/marin3r)](https://github.com/3scale/marin3r/blob/master/LICENSE)

Small and simple envoy control plane for kubernetes:

* Runs in a pod
* Feeds envoy configurations from configmaps
* Integrates with cert-manager certificates

## Motivation

marin3r started in a weekend (thank you COVID-19 for the countless indoor hours ...) as an exploratory project: is it possible to write a stupid simple yet flexible envoy control plane that meets production requirements? It needed to be able to manage a fleet of envoy sidecars containers without all the hassle of having to perform envoy hot restarts to reload configs (necessary when using file based config). At the same time it needed to be flexible so we could do through the control plane anything that could be done using an envoy config file. The result of the experiment is an envoy control plane that reads envoy configurations from configmaps and feeds them through the envoy discovery services to the envoy containers.

## Getting started

marin3r has two components, an envoy aggregated discovery service (a.k.a "the control plamne") and an optional (though recommended) kubernetes mutating admission webhook to inject envoy containers in your pods. With these two components you can very quickly some service mesh patterns, with the full envoy configuration set available.

NOTE: marin3r is currently scoped to a single namespace, this might change in the future.

NOTE2: currently only cluster, listeners and secrets can be discovered by marin3r.

For a quick start, both the control plane and the webhook can be deployed as a single container. Also, some certificates are required both for the webhook server and the ads (aggregated discovery service) grpc server. Some example certicates are already present in `deploy/getting-started` so you can follow this tutorial to deploy and test out marin3r. Issue the following commands in the root of the repository:

```bash
kubectl label namespace/default marin3r.3scale.net/status="enabled"
kubectl create secret tls marin3r-server-cert --cert=deploy/getting-started/certs/marin3r.default.svc.crt --key=deploy/getting-started/certs/marin3r.default.svc.key
kubectl create secret tls marin3r-ca-cert --cert=deploy/getting-started/certs/ca.crt --key=deploy/getting-started/certs/ca.key
kubectl create secret tls envoy-sidecar-client-cert --cert=deploy/getting-started/certs/envoy-client.crt --key=deploy/getting-started/certs/envoy-client.key
kubectl apply -f deploy/getting-started/marin3r.yaml
```

After a few secons seconds you should see the marin3r pod running:

```bash
$ kubectl get pods -l app=marin3r
NAME                       READY   STATUS    RESTARTS   AGE
marin3r-76f99458dd-fj7gg   1/1     Running   0          21s
```

Now, let's deploy the kubernetes up and running demo app.

```bash
cat <<'EOF' | kubectl apply -f -
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kuard
  labels:
    app: kuard
spec:
  selector:
    matchLabels:
      app: kuard
  template:
    metadata:
      labels:
        app: kuard
        marin3r.3scale.net/status: "enabled"
      annotations:
        marin3r.3scale.net/node-id: kuard
        marin3r.3scale.net/ports: envoy-https:1443
    spec:
      containers:
        - name: kuard
          image: gcr.io/kuar-demo/kuard-amd64:blue
          ports:
            - containerPort: 8080
              name: http
              protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  name: kuard
  labels:
    app: kuard
spec:
  type: ClusterIP
  ports:
    - name: envoy-https
      port: 443
      targetPort: envoy-https
  selector:
    app: kuard
EOF
```

You should see that a new pod is running, but is has 2 containers instead of the one we declared. The webhook just added an envoy sidecar to the pod:

```bash
kubectl get pods -l app=kuard
NAME                       READY   STATUS    RESTARTS   AGE
kuard-6bd9456d55-xbs7m     2/2     Running   0          32s
```

We now need to provide the envoy-sidecar with the appropriate config to publish the kuard application through https:

* We need first to create a certificate

```bash
kubectl create secret tls kuard-certificate --cert=deploy/getting-started/certs/envoy-server.crt --key=deploy/getting-started/certs/envoy-server.key
kubectl annotate secret kuard-certificate cert-manager.io/common-name=envoy-server1
```

* Then apply the following envoy config by applying this ConfigMap to the cluster

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: envoy-config
  namespace: default
  annotations:
    marin3r.3scale.net/node-id: kuard
data:
  config.yaml: |
    clusters:
      - name: kuard
        connect_timeout: 2s
        type: STRICT_DNS
        lb_policy: ROUND_ROBIN
        load_assignment:
          cluster_name: kuard
          endpoints:
            - lb_endpoints:
                - endpoint:
                    address:
                      socket_address:
                        address: 127.0.0.1
                        port_value: 8080
    listeners:
      - name: https
        address:
          socket_address:
            address: 0.0.0.0
            port_value: 1443
        filter_chains:
          - filters:
            - name: envoy.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager
                stat_prefix: kuard_https
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: kuard
                      domains: ["*"]
                      routes:
                        - match:
                            prefix: "/"
                          route:
                            cluster: kuard
                http_filters:
                  - name: envoy.filters.http.router
            transport_socket:
              name: envoy.transport_sockets.tls
              typed_config:
                "@type": "type.googleapis.com/envoy.api.v2.auth.DownstreamTlsContext"
                common_tls_context:
                  tls_certificate_sds_secret_configs:
                    - name: envoy-server1
                      sds_config:
                        ads: {}
EOF
```

You can now access the kuard service by running `kubectl proxy` (you can also expose the service through an ingress or with a Load Balancer if your cluster provider supports it):

* Run kubectl proxy in a differen shell:

```bash
$ kubectl proxy
Starting to serve on 127.0.0.1:8001
```

* Then you can access the service by using the k8s API proxy:

```bash
$ curl http://127.0.0.1:8001/api/v1/namespaces/default/services/https:kuard:443/proxy/ -v
*   Trying 127.0.0.1:8001...
* TCP_NODELAY set
* Connected to 127.0.0.1 (127.0.0.1) port 8001 (#0)
> GET /api/v1/namespaces/default/services/https:kuard:443/proxy/ HTTP/1.1
> Host: 127.0.0.1:8001
> User-Agent: curl/7.66.0
> Accept: */*
>
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< Content-Length: 1930
< Content-Type: text/html
< Date: Mon, 20 Apr 2020 16:59:36 GMT
< Server: envoy
< X-Envoy-Upstream-Service-Time: 1
<
<!doctype html>

<html lang="en">
<head>
  <meta charset="utf-8">

  <title>KUAR Demo</title>
```

Note the `Server: envoy` header we received, stating that the envoy sidecar is proxying our request to the kuard container.

## Configuration

marin3r is configured with 3 different inputs: ConfigMaps, Secrets and Pod annotations.

### ConfigMaps

marin3r reads envoy clusters and listeners from kubernetes ConfigMap resources annotated with the `marin3r.3scale.net/node-id` annotation. The value in this annotation identifies the envoy `node-id` that the cofinguration belongs to. For example, a common pattern would be to have a deployment with several pods sharing the same `node-id` so the all get the same envoy config.

The contents of the ConfigMap must be a key `config.yaml` whose value follows the structure:

```yaml
clusters:
  - cluster1
  - cluster2
lsterners:
  - listerner1
  - listerner2
```

### Secrets

marin3r is designed to work with [cert-manager](cert-manager.io) certificates, but users can also deploy their own certificates as secrets in the clusters and set te appropriate annotations for marin3r to discover them automatically. The certificate discovery is base in the `"cert-manager.io/common-name"` annotation of secrets. This annotations is expected to hold the common name of the certificate, and it is the name that should be used to refer certificates from the envoy clusters/listeners.

For example, if we have the following certificate

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: test-certificate
  annotations:
    cert-manager.io/common-name: test-certificate.local
type: kubernetes.io/tls
data:
  tls.crt: <base64 encoded cert>
  tls.key: <base64 encoded key>
```

This certifica should be referred in an envoy cluster/listener as:

```yaml
transport_socket:
  name: envoy.transport_sockets.tls
  typed_config:
    "@type": "type.googleapis.com/envoy.api.v2.auth.DownstreamTlsContext"
    common_tls_context:
      tls_certificate_sds_secret_configs:
        - name: test-certificate.local
          sds_config:
            ads: {}
```

### Annotations

The marin3r mutating admission webhook will inject envoy sidecars in any pod annotated with `marin3r.3scale.net/node-id`, but there are also other annotations available that allow for some configuration:

| annotations                           | description                                                                                                                 | default value             |
|---------------------------------------|-----------------------------------------------------------------------------------------------------------------------------|---------------------------|
| marin3r.3scale.net/node-id            | envoy's node-id                                                                                                             | N/A                       |
| marin3r.3scale.net/cluster-id         | envoy's cluster-id                                                                                                          | same as node-id           |
| marin3r.3scale.net/container-name     | the name of the envoy sidecar                                                                                               | envoy-sidecar             |
| marin3r.3scale.net/ports              | the exposed ports in the envoy sidecar                                                                                      | N/A                       |
| marin3r.3scale.net/host-port-mappings | envoy sidecar ports that will be mapped to the host. This is used for local development, no recommended for production use. | N/A                       |
| marin3r.3scale.net/envoy-image        | the envoy image to be used in the injected sidecar container                                                                | envoyproxy/envoy:v1.14.1  |
| marin3r.3scale.net/ads-configmap      | the envoy bootstrap configuration                                                                                           | envoy-sidecar-bootstrap   |
| marin3r.3scale.net/config-volume      | the pod volume where the ads-configmap will be mounted                                                                      | envoy-sidecar-bootstrap   |
| marin3r.3scale.net/tls-volume         | the pod volume where the marin3r client certificate will be mounted.                                                        | envoy-sidecar-tls         |
| marin3r.3scale.net/client-certificate | the marin3r client certificate to use to authenticate to the marin3r control plane (marin3r uses mTLS))                     | envoy-sidecar-client-cert |
| marin3r.3scale.net/envoy-extra-args   | extra command line arguments to pass to the envoy sidecar container                                                         | ""                        |

#### `marin3r.3scale.net/ports` syntax

The `port` syntax is a comma-separated list of `name:port[:protocol]` as in `"envoy-http:1080,envoy-https:1443"`.

#### `marin3r.3scale.net/host-port-mappings` syntax

The `host-port-mappings` syntax is a comma-separated list of `container-port-name:host-port-number` as in `"envoy-http:1080,envoy-https:1443"`.
