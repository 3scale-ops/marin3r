# **marin3r**

[![Go Report Card](https://goreportcard.com/badge/github.com/3scale/marin3r)](https://goreportcard.com/report/github.com/3scale/marin3r)
[![codecov](https://codecov.io/gh/3scale/marin3r/branch/master/graph/badge.svg)](https://codecov.io/gh/3scale/marin3r)
[![build status](https://circleci.com/gh/3scale/marin3r.svg?style=shield)](https://codecov.io/gh/3scale/marin3r/.circleci/config.yml)
[![license](https://badgen.net/github/license/3scale/marin3r)](https://github.com/3scale/marin3r/blob/master/LICENSE)

Lighweight, CRD based envoy control plane for kubernetes:

* Runs in a pod
* Feeds envoy configurations from CRDs
* Easy integration with cert-manager certificates
* Any secret of type `kubernetes.io/tls` can be used as a certificate source
* Installation managed by an operator
* Self-healing capabilities
* Implements the sidecar pattern: injects envoy sidecar containers based on pod annotations

## **Motivation**

At Red Hat 3scale operations team we run a SaaS platform based on the Red Hat 3scale API Management product. Running a SaaS poses several challenges that are not usually present on most of the 3scale deployments that our clients run:

* avoid DDoS attacks
* easily manage certificates at a higher scale
* very high platform availability that requires the ability to perform configuration changes without service disruption
* have a platform wide ingress layer that allows us to apply certain intelligence at the network level: transformations, routing, rate limiting ...

Even if a product/application has these capabilities to some extent, the focus and purpose is different. We want to have these capabilities outside of the application, at the platform level, as it gives us flexibility and quicker reaction times.

marin3r is the project we created to manage an envoy discovery service and the envoy configurations we need. It currently supports the sidecar pattern (deploy envoy as a sidecar container in your pods) but we plan to add support for deploying envoy as an ingress controller.

The focus of marin3r is on robustness, flexibility, automation and availability. This comes at the cost of having a tougher learning curve because, as of now, proxy configurations make direct use of the envoy API (v2), which can be challenging if you are new to envoy. We plan to refine this approach over time with domain specific CRDs that can take out this complexity to some extent, but having the full envoy API available from the start was important to fullfill our requirements and that is what we focused on so far.

## **Getting started**

### **Installation**

marin3r is a kubernetes operator and as such can be installed by deploying the operator into the cluster and creating custom resources (we plan to release marin3r through operatorhub.io in the future). The only requirement is that [cert-manager](https://cert-manager.io/) must be available in the cluster because marin3r relies on cert-manager to generate the required certificates that the envoy discovery service needs.

To install cert-manager issue the following commands:

```bash
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v0.14.3/cert-manager.crds.yaml
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v0.14.3/cert-manager.yaml
```

Once cert-manager is deployed you can proceed and deploy marin3r. From the root directory of the repo issue the following commands:

```bash
kubectl create namespace marin3r
kubectl apply -f deploy/crds/
kubectl apply -f deploy/
```

After a while you should see the `marin3r-operator` pod running.

```bash
$ kubectl -n marin3r get pods
NAME                               READY   STATUS    RESTARTS   AGE
marin3r-operator-96757f5d8-vvk82   1/1     Running   0          31s
```

You can now deploy a DiscoveryService custom resource, which will cause an envoy discovery service to be spun up:

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: operator.marin3r.3scale.net/v1alpha1
kind: DiscoveryService
metadata:
  name: instance
spec:
  image: quay.io/3scale/marin3r:latest
  discoveryServiceNamespace: marin3r
  debug: true
  signer:
    certManager:
      namespace: cert-manager
  enabledNamespaces:
    - default
EOF
```

Some more seconds and you should see the envoy discovery service pod running alongside the marin3r-operator pod:

```bash
▶ k -n marin3r get pods
NAME                                READY   STATUS    RESTARTS   AGE
marin3r-instance-86f4dbbf8c-kkh5l   1/1     Running   0          11s
marin3r-operator-96757f5d8-jskp2    1/1     Running   0          2m
```

### **TLS offloading with an envoy sidecar**

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
        marin3r.3scale.net/ports: envoy-https:8443
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

You should see that a new pod is running, but it has 2 containers instead of the one we declared. An envoy container just got added to the pod by marin3r:

```bash
kubectl get pods -l app=kuard
NAME                       READY   STATUS    RESTARTS   AGE
kuard-6bd9456d55-xbs7m     2/2     Running   0          32s
```

We need now to provide the envoy-sidecar with the appropriate config to publish the kuard application through https.

#### **Create a certificate**

Use openSSL to create a self-signed certificate that we will be using for this example.

NOTE: you could also generate a certificate with cert-manager. Check [cert-manager documentation](https://cert-manager.io/docs/).

```bash
openssl req -x509 -newkey rsa:4096 -keyout /tmp/key.pem -out /tmp/cert.pem -days 365 -nodes -subj '/CN=localhost'
```

Generate a kubernetes Secret from the certificate.

```bash
kubectl create secret tls kuard-certificate --cert=/tmp/cert.pem --key=/tmp/key.pem
```

#### **Add an EnvoyConfig to publish kuard through https**

Apply the following EnvoyConfig custom resource to the cluster. The EnvoyConfig objects are used to apply raw envoy configs that will be loaded by any envoy proxy in the cluster that matches the `nodeID` field defined in the spec (notice the `marin3r.3scale.net/node-id` annotation we added in the kuard Deployment). Any update of an EnvoyConfig object will update the configuration of the corresponding envoy proxies without any kind of restart or reload.

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: marin3r.3scale.net/v1alpha1
kind: EnvoyConfig
metadata:
  name: kuard
spec:
  nodeID: kuard
  serialization: yaml
  envoyResources:
    secrets:
      - name: kuard-certificate
        ref:
          name: kuard-certificate
          namespace: default
    clusters:
      - name: kuard
        value: |
          name: kuard
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
        value: |
          name: https
          address:
            socket_address:
              address: 0.0.0.0
              port_value: 8443
          filter_chains:
            - filters:
              - name: envoy.http_connection_manager
                typed_config:
                  "@type": type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager
                  stat_prefix: ingress_http
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
                    - name: envoy.router
              transport_socket:
                name: envoy.transport_sockets.tls
                typed_config:
                  "@type": "type.googleapis.com/envoy.api.v2.auth.DownstreamTlsContext"
                  common_tls_context:
                    tls_certificate_sds_secret_configs:
                      - name: kuard-certificate
                        sds_config:
                          ads: {}
EOF
```

If you now check the status of the EnvoyConfig object, `CacheState` should show `InSync`.

```bash
$ kubectl get envoyconfig
NAME    NODEID   DESIRED VERSION   PUBLISHED VERSION   CACHE STATE
kuard   kuard    99d577784         99d577784           InSync
```

You can now access the kuard service by running `kubectl proxy` (you can also expose the service through an Ingress or with a Service of the LoadBalancer type if your cluster provider supports it).

Run kubectl proxy in a different shell:

```bash
$ kubectl proxy
Starting to serve on 127.0.0.1:8001
```

Access the service using https by using the k8s API proxy:

```bash
curl http://127.0.0.1:8001/api/v1/namespaces/default/services/https:kuard:443/proxy/ -v --silent
```

```bash
$ curl http://127.0.0.1:8001/api/v1/namespaces/default/services/https:kuard:443/proxy/ -v --silent
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
< Content-Length: 1931
< Content-Type: text/html
< Date: Wed, 08 Jul 2020 18:42:46 GMT
< Server: envoy
< X-Envoy-Upstream-Service-Time: 1
< 
<!doctype html>

<html lang="en">
<head>
  <meta charset="utf-8">

  <title>KUAR Demo</title>

  <link rel="stylesheet" href="/api/v1/namespaces/default/services/https:kuard:443/proxy/static/css/bootstrap.min.css">
  <link rel="stylesheet" href="/api/v1/namespaces/default/services/https:kuard:443/proxy/static/css/styles.css">

  <script>
var pageContext = {"urlBase":"","hostname":"kuard-6f8fc46b67-vsgn7","addrs":["10.244.0.10"],"version":"v0.10.0-blue","versionColor":"hsl(339,100%,50%)","requestDump":"GET / HTTP/1.1\r\nHost: 127.0.0.1:8001\r\nAccept: */*\r\nAccept-Encoding: gzip\r\nContent-Length: 0\r\nUser-Agent: curl/7.66.0\r\nX-Envoy-Expected-Rq-Timeout-Ms: 15000\r\nX-Forwarded-For: 127.0.0.1, 172.17.0.1\r\nX-Forwarded-Proto: https\r\nX-Forwarded-Uri: /api/v1/namespaces/default/services/https:kuard:443/proxy/\r\nX-Request-Id: 0cba00a2-c5e6-4d56-b02a-c9b8882ed30d","requestProto":"HTTP/1.1","requestAddr":"127.0.0.1:56484"}
  </script>
</head>

[...]
```

Note the `Server: envoy` header we received, stating that the envoy sidecar is proxying our request to the kuard container.
You can also 

### Self-healing

marin3r has self-healing capabilities and will detect when an envoy proxy rejects the configuration that the discovery service is sending to it (most tipically due to an invalid configuration or change). When one of such situations occur, marin3r will revert the proxy config back to the previous working one to avoid config drifts, like for example having an updated listener pointing to a non existent cluster (because the proxy has rejected the cluster config for some reason).

Let's do an example using our kuard setup.

First, modify the EnvoyConfig object to change the port that the https listener binds to. This is incorrect and the envoy proxy will reject it because address changes are not allowed on listener resources (the correct way of doing this would be adding a new listener and then removing the old one).

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: marin3r.3scale.net/v1alpha1
kind: EnvoyConfig
metadata:
  name: kuard
spec:
  nodeID: kuard
  serialization: yaml
  envoyResources:
    secrets:
      - name: kuard-certificate
        ref:
          name: kuard-certificate
          namespace: default
    clusters:
      - name: kuard
        value: |
          name: kuard
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
        value: |
          name: https
          address:
            socket_address:
              address: 0.0.0.0
              # Changed port value from 8443 to 5000
              port_value: 5000
          filter_chains:
            - filters:
              - name: envoy.http_connection_manager
                typed_config:
                  "@type": type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager
                  stat_prefix: ingress_http
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
                    - name: envoy.router
              transport_socket:
                name: envoy.transport_sockets.tls
                typed_config:
                  "@type": "type.googleapis.com/envoy.api.v2.auth.DownstreamTlsContext"
                  common_tls_context:
                    tls_certificate_sds_secret_configs:
                      - name: kuard-certificate
                        sds_config:
                          ads: {}
EOF
```

After applying this EnvoyConfig, the `CacheState` of the object will be `Rollback` because the envoy proxy rejected the listener configuration and marin3r detected it and reverted the config to the last working one.

```bash
▶ kubectl get envoyconfig
NAME    NODEID   DESIRED VERSION   PUBLISHED VERSION   CACHE STATE
kuard   kuard    6c8c87788         99d577784           Rollback
```

Next time a correct config is applied, the `Rollback` status would go back to `InSync`.

## Configuration

### EnvoyConfig custom resource

marin3r basic functionality is to feed the envoy configs defined in EnvoyConfig custom resources to an envoy discovery service. The discovery service then sends the resources contained in those configs to the envoy proxies that identify themselves with the same `nodeID` defined in the EnvoyConfig object.

Commented example of an EnvoyConfig object:

```yaml
cat <<'EOF' | kubectl apply -f -
apiVersion: marin3r.3scale.net/v1alpha1
kind: EnvoyConfig
metadata:
  # name and namespace uniquelly identify an EnvoyConfig but are
  # not relevant in any other way
  name: config
  namespace: default
spec:
  # nodeID indicates that the resources defined in this EnvoyConfig are relevant
  # to envoy proxies that identify themselves to the discovery service with the same
  # nodeID. The nodeID of an envoy proxy can be specified using the "--node-id" command
  # line flag
  nodeID: proxy
  # Resources can be written either in json or in yaml, being json the default if
  # not specified
  serialization: json
  envoyResources:
    # the "secrets" field holds references to Kubernetes Secrets. Only Secrets of type
    # "kubernetes.io/tls" can be referenced. Any certificate referenced from another envoy
    # resource (for example a listener or a cluster) needs to be present here so marin3r
    # knows where to get the certificate from.
    secrets:
        # name is the name by which the certificate can be referenced to from other resources
      - name: certificate
        # ref is the Kubernetes object name and namespace where the Secret lives. Secrets from
        # other namespaces can be referenced. This is usefull for example if you have a wildcard
        # certificate that is used in different namespaces.
        ref:
          name: some-secret
          namespace: some-namespace
    # envoy resources of the type "endpoint"
    # reference: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/endpoint.proto
    endpoints:
      - name: endpoint1
        value: {"clusterName":"cluster1","endpoints":[{"lbEndpoints":[{"endpoint":{"address":{"socketAddress":{"address":"127.0.0.1","portValue":8080}}}}]}]}
    # envoy resources of the type "cluster"
    # reference: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/cluster.proto
    clusters:
      - name: cluster1
        value: {"name":"cluster1","type":"STRICT_DNS","connectTimeout":"2s","loadAssignment":{"clusterName":"cluster1","endpoints":[]}}
    # envoy resources of the type "route"
    # reference: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/route.proto
    routes:
      - name: route1
        value: {"name":"route1","match":{"prefix":"/"},"directResponse":{"status":200}}
    # envoy resources of the type "listener"
    # reference: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/listener.proto
    listeners:
      - name: listener1
        value: {"name":"listener1","address":{"socketAddress":{"address":"0.0.0.0","portValue":8443}}}
    # envoy resources of the type "runtime"
    # reference: https://www.envoyproxy.io/docs/envoy/latest/configuration/operations/runtime
    runtimes:
      - name: runtime1
        value: {"name":"runtime1","layer":{"static_layer_0":"value"}}
```

### Secrets

Secrets are treated in a special way by marin3r as they contain sensitive information. Instead of directly declaring an envoy API secret resource in the EnvoyConfig CR, you have to reference a Kubernetes Secret. marin3r expects this Secret to be of type `kubernetes.io/tls` and will load it into an envoy secret resource. This way you avoid having to insert sensitive data into the EnvoyConfig objects and allows you to use your regular kubernetes Secret deployment workflow for secrets.

Other approach that can be used is to create certificates using cert-manager because cert-manager also uses `kubernetes.io/tls` Secrets to store the certificates it generates. You just need to point the references in your EnvoyConfig to the proper cert-manager generated Secret.

To use a certificate from a kubernetes Secret refer it like this from an EnvoyConfig:

```yaml
spec:
  envoyResources:
    secrets:
      - name: certificate
        ref:
          name: some-k8s-secret-name
          namespace: some-namespace
```

This certificate can then be referenced in an envoy cluster/listener with the folloing snippet (check the kuard example):

```yaml
transport_socket:
  name: envoy.transport_sockets.tls
  typed_config:
    "@type": "type.googleapis.com/envoy.api.v2.auth.DownstreamTlsContext"
    common_tls_context:
      tls_certificate_sds_secret_configs:
        - name: certificate
          sds_config:
            ads: {}
```

### Sidecar injection configuration

The marin3r mutating admission webhook will inject envoy containers in any pod annotated with `marin3r.3scale.net/node-id` created inside of any of the marin3r enabled namespaces. There are some annotations that can be used in pods to control the behavior of the webhook:

| annotations                           | description                                                                                                                 | default value             |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- | ------------------------- |
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
