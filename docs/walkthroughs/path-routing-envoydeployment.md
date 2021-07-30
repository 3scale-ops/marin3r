# **Path based routing using an EnvoyDeployment**

In this walkthrough, we will show how you can use the MARIN3R EnvoyDeployment custom resource to deploy Envoy as a Kubernetes Deployment and dynamically configure it to proxy requests to a couple of different applications based on the path of the request. We will first configure it with just one upstream application and then add the second one to see how the config is updated without the need of restarting the Envoy proxies at any time.

## **Preparation**

You need to have MARIN3R operator installed in the cluster and a DiscoveryService within the namespace you will be using. Follow the [installation instructions](../../README.md#installation) to do so if you haven't already.

## **Deploy two applications that we will use as upstreams to proxy requests to**

We will be using `nginxdemos/hello:plain-text` to deploy our two upstream applications:

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: upstream-a
  labels:
    app: upstream-a
spec:
  selector:
    matchLabels:
      app: upstream-a
  template:
    metadata:
      labels:
        app: upstream-a
    spec:
      containers:
        - name: nginx
          image: nginxdemos/nginx-hello:plain-text
          ports:
            - containerPort: 8080
              name: http
              protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  name: upstream-a
  labels:
    app: upstream-a
spec:
  type: ClusterIP
  clusterIP: None
  ports:
    - name: http
      port: 8080
      targetPort: http
  selector:
    app: upstream-a
EOF
```

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: upstream-b
  labels:
    app: upstream-b
spec:
  selector:
    matchLabels:
      app: upstream-b
  template:
    metadata:
      labels:
        app: upstream-b
    spec:
      containers:
        - name: nginx
          image: nginxdemos/nginx-hello:plain-text
          ports:
            - containerPort: 8080
              name: http
              protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  name: upstream-b
  labels:
    app: upstream-b
spec:
  type: ClusterIP
  clusterIP: None
  ports:
    - name: http
      port: 8080
      targetPort: http
  selector:
    app: upstream-b
EOF
```

Now we will prepare the config to make an Envoy forward all requests with path matching `/a` to the Deployment we named as `upstream-a`. As usual, we will deploy this config as an EnvoyConfig.

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: marin3r.3scale.net/v1alpha1
kind: EnvoyConfig
metadata:
  name: envoy
spec:
  nodeID: envoy
  serialization: yaml
  envoyAPI: v3
  envoyResources:
    clusters:
      - name: upstream-a
        value: |
          name: upstream-a
          connect_timeout: 10ms
          type: STRICT_DNS
          lb_policy: ROUND_ROBIN
          load_assignment:
            cluster_name: upstream-a
            endpoints:
              - lb_endpoints:
                  - endpoint:
                      address:
                        socket_address: { address: upstream-a, port_value: 8080 }
    listeners:
      - name: http
        value: |
          name: http
          address: { socket_address: { address: 0.0.0.0, port_value: 8080 } }
          filter_chains:
            - filters:
                - name: envoy.filters.network.http_connection_manager
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                    stat_prefix: ingress_http
                    route_config:
                      name: route
                      virtual_hosts:
                        - name: any
                          domains: ["*"]
                          routes:
                            - { match: { prefix: "/a" }, route: { cluster: "upstream-a" } }
                    http_filters:
                      - name: envoy.filters.http.router
EOF
```

We are ready to create an EnvoyDeployment resource that references this configuration.

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: operator.marin3r.3scale.net/v1alpha1
kind: EnvoyDeployment
metadata:
  name: envoy
spec:
  discoveryServiceRef: discoveryservice
  envoyConfigRef: envoy
  ports:
    - name: http
      port: 8080
  replicas:
    static: 1
EOF
```

If we perform a `kubectl get pods` in our namespace we should see something like this:

```bash
▶ kubectl get pods
NAME                                             READY   STATUS    RESTARTS   AGE
marin3r-discoveryservice-57f69c978f-hj26d        1/1     Running   0          10m
marin3r-envoydeployment-envoy-5db644bf7c-dv2d6   1/1     Running   0          99s
upstream-a-694bfb66fd-s7ppw                      1/1     Running   0          3m
upstream-b-759687c4d5-5rzvx                      1/1     Running   0          3m
```

- Pod `marin3r-discoveryservice-57f69c978f-hj26d`: the discovery service running Envoy xDS server
- Pod `upstream-a-694bfb66fd-s7ppw`: upstream A
- Pod `upstream-b-759687c4d5-5rzvx`: upstream B
- Pod `marin3r-envoydeployment-envoy-5db644bf7c-dv2d6`: the Envoy proxy we have just created using an EnvoyDeployment resource

## **Test the setup**

You can expose the EnvoyDeployment Pod by running `kubectl port-forward` in your local environment. You could also expose the proxy through a Service of the LoadBalancer type if your cluster provider supports it.

In a different shell, execute the following command and leave it running:

```bash
▶ kubectl port-forward deployment/marin3r-envoydeployment-envoy 8080:8080
Forwarding from 127.0.0.1:8080 -> 8080
Forwarding from [::1]:8080 -> 8080
```

With the current configuration, we have just exposed "upstream A" with our EnvoyConfig so we should get an HTTP 200 when requesting path `/a`, but an HTTP 404 when requesting `/b` (or any other path) because we have no configuration for it (yet).

```bash
▶ curl http://localhost:8080/a -i
HTTP/1.1 200 OK
server: envoy
date: Fri, 02 Jul 2021 12:59:27 GMT
content-type: text/plain
content-length: 160
expires: Fri, 02 Jul 2021 12:59:26 GMT
cache-control: no-cache
x-envoy-upstream-service-time: 0

Server address: 10.244.0.15:8080
Server name: upstream-a-694bfb66fd-s7ppw
Date: 02/Jul/2021:12:59:27 +0000
URI: /a
Request ID: 487c3454d3996b6e2d329e56d6930c6c
```

```bash
▶ curl http://localhost:8080/b -i
HTTP/1.1 404 Not Found
date: Fri, 02 Jul 2021 12:59:31 GMT
server: envoy
content-length: 0
```

## **Modify the configuration to include a route for upstream B**

Apply the following EnvoyConfig. This is exactly the same that we used before, but with an extra route to forward requests to path `/b` to the "upstream B" and a new cluster that points to the `upstream-b` Service.

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: marin3r.3scale.net/v1alpha1
kind: EnvoyConfig
metadata:
  name: envoy
spec:
  nodeID: envoy
  serialization: yaml
  envoyAPI: v3
  envoyResources:
    clusters:
      - name: upstream-a
        value: |
          name: upstream-a
          connect_timeout: 10ms
          type: STRICT_DNS
          lb_policy: ROUND_ROBIN
          load_assignment:
            cluster_name: upstream-a
            endpoints:
              - lb_endpoints:
                  - endpoint:
                      address:
                        socket_address: { address: upstream-a, port_value: 8080 }
      - name: upstream-b
        value: |
          name: upstream-b
          connect_timeout: 10ms
          type: STRICT_DNS
          lb_policy: ROUND_ROBIN
          load_assignment:
            cluster_name: upstream-b
            endpoints:
              - lb_endpoints:
                  - endpoint:
                      address:
                        socket_address: { address: upstream-b, port_value: 8080 }
    listeners:
      - name: http
        value: |
          name: http
          address: { socket_address: { address: 0.0.0.0, port_value: 8080 } }
          filter_chains:
            - filters:
                - name: envoy.filters.network.http_connection_manager
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                    stat_prefix: ingress_http
                    route_config:
                      name: route
                      virtual_hosts:
                        - name: any
                          domains: ["*"]
                          routes:
                            - { match: { prefix: "/a" }, route: { cluster: "upstream-a" } }
                            - { match: { prefix: "/b" }, route: { cluster: "upstream-b" } }
                    http_filters:
                      - name: envoy.filters.http.router
EOF
```

Issue a `kubectl get pods` to verify that the Envoy Pod is still the same as before and that no restarts have occurred.

```bash
▶ kubectl get pods
NAME                                             READY   STATUS    RESTARTS   AGE
marin3r-discoveryservice-57f69c978f-hj26d        1/1     Running   0          15m
marin3r-envoydeployment-envoy-5db644bf7c-dv2d6   1/1     Running   0          2m
upstream-a-694bfb66fd-s7ppw                      1/1     Running   0          5m
upstream-b-759687c4d5-5rzvx                      1/1     Running   0          5m
```

Use curl to check that now both paths `/a` and `/b` reply with an HTTP 200 code, each one from the respective upstream (check the "Server name" field in the response body).

```bash
▶ curl http://localhost:8080/a -i
HTTP/1.1 200 OK
server: envoy
date: Fri, 02 Jul 2021 13:11:48 GMT
content-type: text/plain
content-length: 160
expires: Fri, 02 Jul 2021 13:11:47 GMT
cache-control: no-cache
x-envoy-upstream-service-time: 0

Server address: 10.244.0.15:8080
Server name: upstream-a-694bfb66fd-s7ppw
Date: 02/Jul/2021:13:11:48 +0000
URI: /a
Request ID: e8f087d7b10e72dc59875ab4666a809d
```

```bash
▶ curl http://localhost:8080/b -i
HTTP/1.1 200 OK
server: envoy
date: Fri, 02 Jul 2021 13:32:37 GMT
content-type: text/plain
content-length: 160
expires: Fri, 02 Jul 2021 13:32:36 GMT
cache-control: no-cache
x-envoy-upstream-service-time: 0

Server address: 10.244.0.17:8080
Server name: upstream-b-759687c4d5-5rzvx
Date: 02/Jul/2021:13:32:37 +0000
URI: /b
Request ID: f4a0c6f4651fe5529e79311b05a38356
```

## **Conclusion**

We have seen how we can use MARIN3R to use Envoy in front of several applications to distribute traffic among them based on the path of the incoming request. Traffic distribution could also be done by hostname, by inspecting headers, etc. The configuration management for the Envoy proxy is completely declarative as manual operations are not required when modifying the configuration. You could go further and develop your own domain-specific ingress leveraging the EnvoyConfig and EnvoyDeployment resource, adding a layer of abstraction that properly adapts your requirements.

## **Cleanup**

Execute the following commands to delete the resources created in this walkthough:

```bash
kubectl delete envoydeployment envoy
kubectl delete envoyconfig envoy
kubectl delete deployment upstream-a
kubectl delete deployment upstream-b
```
