<img src="docs/logos/white.svg" height="150px" alt="MARIN3R"></img>

[![Go Report Card](https://goreportcard.com/badge/github.com/3scale-ops/marin3r)](https://goreportcard.com/report/github.com/3scale-ops/marin3r)
[![codecov](https://codecov.io/gh/3scale-ops/marin3r/branch/master/graph/badge.svg)](https://codecov.io/gh/3scale-ops/marin3r)
[![build](https://github.com/3scale-ops/marin3r/actions/workflows/build.yaml/badge.svg)](https://github.com/3scale-ops/marin3r/actions/workflows/build.yaml)
[![license](https://badgen.net/github/license/3scale-ops/marin3r)](https://github.com/3scale-ops/marin3r/blob/master/LICENSE)

Lighweight, CRD based Envoy control plane for Kubernetes:

- Implemented as a Kubernetes Operator
- Dynamic Envoy configuration using Kubernetes Custom Resources
- Use any secret of type `kubernetes.io/tls` as a certificate source
- Self-healing
- Injects Envoy sidecar containers based on Pod annotations

<!-- omit in toc -->
## Table of Contents

- [**Overview**](#overview)
- [**Getting started**](#getting-started)
  - [**Installation**](#installation)
    - [**Install using OLM**](#install-using-olm)
    - [**Install using kustomize**](#install-using-kustomize)
  - [**Deploy a discovery service**](#deploy-a-discovery-service)
  - [**Example: TLS offloading with an Envoy sidecar**](#example-tls-offloading-with-an-envoy-sidecar)
  - [**Self-healing**](#self-healing)
- [**Configuration**](#configuration)
  - [**API reference**](#api-reference)
  - [**EnvoyConfig custom resource**](#envoyconfig-custom-resource)
  - [**Secrets**](#secrets)
  - [**Sidecar injection configuration**](#sidecar-injection-configuration)
- [**Use cases**](#use-cases)
  - [**Ratelimit**](#ratelimit)
- [**Design docs**](#design-docs)
  - [**Discovery service**](#discovery-service)
  - [**Sidecar injection**](#sidecar-injection)
  - [**Operator**](#operator)
- [**Development**](#development)

## **Overview**

MARIN3R is a Kubernetes operator to manage a fleet of Envoy proxies within a Kubernetes cluster. It takes care of the deployment of the proxies and manages their configuration, feeding it to them through a discovery service using Envoy's [xDS protocol](https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol). This allows for dynamic reconfiguration of the proxies without any reloads or restarts, favoring the ability to perform configuration changes in a non-disruptive way.

Users can write their Envoy configurations by making use of [Kubernetes Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) that the operator will watch and make available to the proxies through the discovery service. Configurations are defined making direct use of Envoy's v2/v3 APIs so anything supported in the Envoy APIs is available in MARIN3R. See the [configuration section](#configuration) or the [API reference](docs/api-reference/reference.asciidoc) for more details.

A great way to use this project is to have your own operator generating the Envoy configurations that your platform/service requires by making use of MARIN3R APIs. This way you can just focus on developing the Envoy configurations you need and let MARIN3R take care of the rest.

## **Getting started**

### **Installation**

MARIN3R can be installed either by using [kustomize](https://kustomize.io/) or by using [Operator Lifecycle Manager (OLM)](https://github.com/operator-framework/operator-lifecycle-manager). We recommend using OLM installation whenever possible.

#### **Install using OLM**

OLM is installed by default in Openshift 4.x clusters. For any other Kubernetes flavor, check if it is already installed in your cluster. If not, you can easily do so by following the [OLM install guide](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/install/install.md).

Once OLM is installed in your cluster, you can proceed with the operator installation by applying the install manifests. This will create a namespaced install of MARIN3R that will only watch for resources in the `default` namespace, with the operator deployed in the `marin3r-system` namespace. Modify the field `spec.targetNamespaces` of the OperatorGroup resource in `examples/quickstart/olm-install.yaml` to modify the namespaces that MARIN3R will watch.

```bash
kubectl apply -f examples/quickstart/olm-install.yaml
```

Wait until you see the following Pods running:

```bash
▶ kubectl -n marin3r-system get pods | grep Running
marin3r-catalog-qsx9t                                             1/1     Running     0          103s
marin3r-controller-manager-5f97f86fc5-qbp6d                       2/2     Running     0          42s
marin3r-controller-webhook-5d4d855859-67zr6                       1/1     Running     0          42s
marin3r-controller-webhook-5d4d855859-6972h                       1/1     Running     0          42s
```

#### **Install using kustomize**

This method will install MARIN3R with cluster scope permissions in your cluster. It requires that [cert-manager](https://cert-manager.io/) is present in the cluster.

To install cert-manager you can execute the following command in the root directory of this repository:

```bash
make deploy-cert-manager
```

You can also refer to the [cert-manager install documentation](https://cert-manager.io/docs/installation/).

Once cert-manager is available in the cluster, you can install MARIN3R by issuing the following command:

```bash
kustomize build config/default | kubectl apply -f -
```

After a while you should see the following Pods running:

```bash
▶ kubectl -n marin3r-system get pods
NAME                                          READY   STATUS    RESTARTS   AGE
marin3r-controller-manager-6c45f7675f-cs6dq   2/2     Running   0          31s
marin3r-controller-webhook-684bf5bbfd-cp2x4   1/1     Running   0          31s
marin3r-controller-webhook-684bf5bbfd-zdvrk   1/1     Running   0          31s
```

### **Deploy a discovery service**

A discovery service is a Pod that users need to deploy in their namespaces to provide such namespaces with the ability to configure Envoy proxies dynamically using configurations loaded from Kubernetes Custom Resources. This Pod runs a couple of Kubernetes controllers as well as an Envoy xDS server. To deploy a discovery service users make use of the DiscoveryService custom resource that MARIN3R provides. The DiscoveryService is a namespace scoped resource, so one is required for each namespace where Envoy proxies are going to be deployed.

Continuing with our example, we are going to deploy a DiscoveryService resource in the `default` namespace of our cluster:

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: operator.marin3r.3scale.net/v1alpha1
kind: DiscoveryService
metadata:
  name: discoveryservice
  namespace: default
EOF
```

After a while you should see the discovery service Pod running:

```bash
▶ kubectl -n default get pods
NAME                                READY   STATUS    RESTARTS   AGE
marin3r-discoveryservice-676b5cd7db-xk9rt   1/1     Running   0          4s
```

We are now ready to deploy Envoy proxies in the `default` namespace and Kubernetes Custom Resources to configure them.

### **Example: TLS offloading with an Envoy sidecar**

For this example, let's deploy the [kubernetes up and running demo app](https://github.com/kubernetes-up-and-running/kuard).

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

Note that we have added some labels and annotations to the Pod's metadata in order to activate sidecar injection for the Deployment, specifically the label `marin3r.3scale.net/status=enabled` and the annotation `marin3r.3scale.net/node-id=kuard`. The presence of these in a Pod makes the Pod creation request to be modified by MARIN3R's mutating webhook, which will inject an Envoy sidecar in the Pod's spec. The sidecar injection can be further configured with several other annotations, see the [configuration section](#sidecar-injection-configuration) for more details.

You should see that a new Pod is running, but it has 2 containers instead of the one we declared. An Envoy container just got added to the pod by MARIN3R:

```bash
kubectl get pods -l app=kuard
NAME                       READY   STATUS    RESTARTS   AGE
kuard-6bd9456d55-xbs7m     2/2     Running   0          32s
```

We need now to provide the Envoy sidecar with the appropriate config to publish the kuard application through https.

<!-- omit in toc -->
#### **Create a certificate**

Use openSSL to create a self-signed certificate that we will be using for this example.

NOTE: you could also generate a certificate with cert-manager if you have it available in your cluster. This would be a typical case for a production environment. Check [cert-manager documentation](https://cert-manager.io/docs/).

```bash
openssl req -x509 -newkey rsa:4096 -keyout /tmp/key.pem -out /tmp/cert.pem -days 365 -nodes -subj '/CN=localhost'
```

Generate a kubernetes Secret from the certificate.

```bash
kubectl create secret tls kuard-certificate --cert=/tmp/cert.pem --key=/tmp/key.pem
```

<!-- omit in toc -->
#### **Add an EnvoyConfig to publish kuard through https**

Apply the following EnvoyConfig custom resource to the cluster. The EnvoyConfig objects are used to apply raw Envoy configs that will be loaded by any Envoy proxy in the namespace that matches the `nodeID` field defined in the spec (notice the `marin3r.3scale.net/node-id` annotation we added in the kuard Deployment). Any update of an EnvoyConfig object will update the configuration of the corresponding Envoy proxies without any kind of restart or reload.

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

Note the `Server: envoy` header we received, stating that the Envoy sidecar is proxying our request to the kuard container.

### **Self-healing**

MARIN3R has self-healing capabilities and will detect when an Envoy proxy rejects the configuration that the discovery service is sending to it (most tipically due to an invalid configuration or change). When one of such situations occur, MARIN3R will revert the proxy config back to the previous working one to avoid config drifts, like for example having an updated listener pointing to a non existent cluster (which could happend if the proxy rejects the cluster config for some reason).

Let's do an example using our kuard setup.

First, modify the EnvoyConfig object to change the port that the https listener binds to. This is incorrect and the Envoy proxy will reject it because address changes are not allowed in listener resources (the correct way of doing this would be adding a new listener and then removing the old one).

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

After applying this EnvoyConfig, the `CacheState` of the object will be `Rollback` because the Envoy proxy rejected the listener configuration and MARIN3R detected it and reverted the config to the last working one.

```bash
▶ kubectl get envoyconfig
NAME    NODEID   DESIRED VERSION   PUBLISHED VERSION   CACHE STATE
kuard   kuard    6c8c87788         99d577784           Rollback
```

Next time a correct config is applied, the `Rollback` status will go back to `InSync`.

## **Configuration**

### **API reference**

The full MARIN3R API reference can be found [here](docs/api-reference/reference.asciidoc)

### **EnvoyConfig custom resource**

MARIN3R basic functionality is to feed the Envoy configs defined in EnvoyConfig custom resources to an Envoy discovery service. The discovery service then sends the resources contained in those configs to the Envoy proxies that identify themselves with the same `nodeID` defined in the EnvoyConfig object.

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
  # to Envoy proxies that identify themselves to the discovery service with the same
  # nodeID. The nodeID of an Envoy proxy can be specified using the "--node-id" command
  # line flag
  nodeID: proxy
  # Resources can be written either in json or in yaml, being json the default if
  # not specified
  serialization: json
  # Resources can be written using either v2 Envoy API or v3 Envoy API. Mixing v2 and v3 resources
  # in the same EnvoyConfig is not allowed. Default is v2.
  envoyAPI: v3
  # envoyResources is where users can write the different type of resources supported by MARIN3R
  envoyResources:
    # the "secrets" field holds references to Kubernetes Secrets. Only Secrets of type
    # "kubernetes.io/tls" can be referenced. Any certificate referenced from another Envoy
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
    # Endpoints is a list of the Envoy ClusterLoadAssignment resource type.
    # V2 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/endpoint.proto
    # V3 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/endpoint/v3/endpoint.proto
    endpoints:
      - name: endpoint1
        value: {"clusterName":"cluster1","endpoints":[{"lbEndpoints":[{"endpoint":{"address":{"socketAddress":{"address":"127.0.0.1","portValue":8080}}}}]}]}
    # Clusters is a list of the Envoy Cluster resource type.
    # V2 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/cluster.proto
    # V3 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/cluster/v3/cluster.proto
    clusters:
      - name: cluster1
        value: {"name":"cluster1","type":"STRICT_DNS","connectTimeout":"2s","loadAssignment":{"clusterName":"cluster1","endpoints":[]}}
    # Routes is a list of the Envoy Route resource type.
    # V2 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/route.proto
    # V3 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/route/v3/route.proto
    routes:
      - name: route1
        value: {"name":"route1","virtual_hosts":[{"name":"vhost","domains":["*"],"routes":[{"match":{"prefix":"/"},"direct_response":{"status":200}}]}]}
    # Listeners is a list of the Envoy Listener resource type.
    # V2 referece: https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/listener.proto
    # V3 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/listener/v3/listener.proto
    listeners:
      - name: listener1
        value: {"name":"listener1","address":{"socketAddress":{"address":"0.0.0.0","portValue":8443}}}
    # Runtimes is a list of the Envoy Runtime resource type.
    # V2 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v2/service/discovery/v2/rtds.proto
    # V3 reference: https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/runtime/v3/rtds.proto
    runtimes:
      - name: runtime1
        value: {"name":"runtime1","layer":{"static_layer_0":"value"}}
```

### **Secrets**

Secrets are treated in a special way by MARIN3R as they contain sensitive information. Instead of directly declaring an Envoy API secret resource in the EnvoyConfig CR, you have to reference a Kubernetes Secret, which should exists in the same namespace. MARIN3R expects this Secret to be of type `kubernetes.io/tls` and will load it into an Envoy secret resource. This way you avoid having to insert sensitive data into the EnvoyConfig objects and allows you to use your regular kubernetes Secret management workflow for sensitive data.

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

This certificate can then be referenced in an Envoy cluster/listener with the folloing snippet (check the kuard example):

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

### **Sidecar injection configuration**

The MARIN3R mutating admission webhook will inject Envoy containers in any Pod annotated with `marin3r.3scale.net/node-id` created inside of any of the MARIN3R enabled namespaces. There are some annotations that can be used in Pods to control the behavior of the webhook:

| annotations                                  | description                                                                                                                                                                                                    | default value             |
| -------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------- |
| marin3r.3scale.net/node-id                   | Envoy's node-id                                                                                                                                                                                                | N/A                       |
| marin3r.3scale.net/cluster-id                | Envoy's cluster-id                                                                                                                                                                                             | same as node-id           |
| marin3r.3scale.net/envoy-api-version         | Envoy's API version (v2/v3)                                                                                                                                                                                    | v2                        |
| marin3r.3scale.net/container-name            | the name of the Envoy sidecar                                                                                                                                                                                  | envoy-sidecar             |
| marin3r.3scale.net/ports                     | the exposed ports in the Envoy sidecar                                                                                                                                                                         | N/A                       |
| marin3r.3scale.net/host-port-mappings        | Envoy sidecar ports that will be mapped to the host. This is used for local development, no recommended for production use.                                                                                    | N/A                       |
| marin3r.3scale.net/envoy-image               | the Envoy image to be used in the injected sidecar container                                                                                                                                                   | envoyproxy/envoy:v1.14.1  |
| marin3r.3scale.net/ads-configmap             | the Envoy bootstrap configuration                                                                                                                                                                              | envoy-sidecar-bootstrap   |
| marin3r.3scale.net/config-volume             | the Pod volume where the ads-configmap will be mounted                                                                                                                                                         | envoy-sidecar-bootstrap   |
| marin3r.3scale.net/tls-volume                | the Pod volume where the marin3r client certificate will be mounted.                                                                                                                                           | envoy-sidecar-tls         |
| marin3r.3scale.net/client-certificate        | the marin3r client certificate to use to authenticate to the marin3r control plane (marin3r uses mTLS))                                                                                                        | envoy-sidecar-client-cert |
| marin3r.3scale.net/envoy-extra-args          | extra command line arguments to pass to the Envoy sidecar container                                                                                                                                            | ""                        |
| marin3r.3scale.net/resources.limits.cpu      | Envoy sidecar container resource cpu limits. See [syntax format](https://v1-17.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#quantity-resource-core) to specify the resource quantity      | N/A                       |
| marin3r.3scale.net/resources.limits.memory   | Envoy sidecar container resource memory limits. See [syntax format](https://v1-17.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#quantity-resource-core) to specify the resource quantity   | N/A                       |
| marin3r.3scale.net/resources.requests.cpu    | Envoy sidecar container resource cpu requests. See [syntax format](https://v1-17.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#quantity-resource-core) to specify the resource quantity    | N/A                       |
| marin3r.3scale.net/resources.requests.memory | Envoy sidecar container resource memory requests. See [syntax format](https://v1-17.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#quantity-resource-core) to specify the resource quantity | N/A                       |

<!-- omit in toc -->
#### `marin3r.3scale.net/ports` syntax

The `port` syntax is a comma-separated list of `name:port[:protocol]` as in `"envoy-http:1080,envoy-https:1443"`.

<!-- omit in toc -->
#### `marin3r.3scale.net/host-port-mappings` syntax

The `host-port-mappings` syntax is a comma-separated list of `container-port-name:host-port-number` as in `"envoy-http:1080,envoy-https:1443"`.

## **Use cases**

### [**Ratelimit**](/docs/use-cases/ratelimit/README.md)

## **Design docs**

For an in-depth look at how MARIN3R works, check the [design docs](/docs/design).

### [**Discovery service**](/docs/design/discovery-service.md)

### [**Sidecar injection**](/docs/design/sidecar-injection.md)

### [**Operator**](/docs/design/operator.md)

## **Development**

You can find development documentation [here](/docs/development/README.md).
