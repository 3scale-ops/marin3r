# **Self-healing**

MARIN3R has self-healing capabilities and will detect when all the Envoy containers reject the configuration that the discovery service is sending them. When one of such situations occur, MARIN3R will revert the config back to the previous working one to avoid config drifts. It also avoids that the discovery service repeatedly keeps trying to send a failing config to the envoy containers, reducing the load caused by config reload retries in Envoy.

Let's do an example using a KUAR Demo setup with Envoy sidecars.

## **Preparation**

You need to have MARIN3R operator installed in the cluster and a DiscoveryService within the namespace you will be using. Follow the [installation instructions](../../README.md#installation) to do so if you haven't already.

## **Deploy the KUAR Demo app with an Envoy sidecar**

Issue the following command to create the required resources: a Deployment with the KUAR Demo app and sidecar injection enabled, and an EnvoyConfig to configure the proxy.

```bash
cat <<'EOF' | kubectl apply -f -
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
        marin3r.3scale.net/ports: envoy-https:8081
        marin3r.3scale.net/envoy-api-version: v3
    spec:
      containers:
        - name: kuard
          image: gcr.io/kuar-demo/kuard-amd64:blue
          ports:
            - containerPort: 8080
              name: http
              protocol: TCP
---
apiVersion: marin3r.3scale.net/v1alpha1
kind: EnvoyConfig
metadata:
  name: kuard
spec:
  nodeID: kuard
  serialization: yaml
  envoyAPI: v3
  envoyResources:
    clusters:
      - name: kuard
        value: |
          name: kuard
          connect_timeout: 10ms
          type: STRICT_DNS
          lb_policy: ROUND_ROBIN
          load_assignment:
            cluster_name: kuard
            endpoints:
              - lb_endpoints:
                  - endpoint:
                      address:
                        socket_address: { address: 127.0.0.1, port_value: 8080 }
    listeners:
      - name: http
        value: |
          name: http
          address: { socket_address: { address: 0.0.0.0, port_value: 8081 } }
          filter_chains:
            - filters:
                - name: envoy.filters.network.http_connection_manager
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                    stat_prefix: ingress_https
                    route_config:
                      name: local_route
                      virtual_hosts:
                        - name: any
                          domains: ["*"]
                          routes:
                            [{ match: { prefix: "/" }, route: { cluster: "kuard" } }]
                    http_filters:
                      - name: envoy.filters.http.router
EOF
```

Open a different shell and use `kubectl port-forward` to gain access to the Pod in your local environment. You could also expose the application using an Ingress or a Service of the type Load Balancer if it is available in your cluster. Leave the `port-forward` command running.

```bash
▶ kubectl port-forward deployment/kuard 8081:8081
Forwarding from 127.0.0.1:8081 -> 8081
Forwarding from [::1]:8081 -> 8081
```

Check that the setup works using the following curl:

```bash
▶ curl http://localhost:8081 -v -o /dev/null --silent
*   Trying ::1:8081...
* Connected to localhost (::1) port 8081 (#0)
> GET / HTTP/1.1
> Host: localhost:8081
> User-Agent: curl/7.76.1
> Accept: */*
>
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< content-length: 1592
< content-type: text/html
< date: Fri, 02 Jul 2021 14:16:50 GMT
< x-envoy-upstream-service-time: 0
< server: envoy
<
{ [1592 bytes data]
* Connection #0 to host localhost left intact
```

## **Modify the Envoy configuration**

Modify the EnvoyConfig resource to change the port that the http listener binds to. This is incorrect and the Envoy proxy will reject it because address changes are not allowed in listener resources (the correct way of doing this would be to add a new listener and then remove the old one).

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: marin3r.3scale.net/v1alpha1
kind: EnvoyConfig
metadata:
  name: kuard
spec:
  nodeID: kuard
  serialization: yaml
  envoyAPI: v3
  envoyResources:
    clusters:
      - name: kuard
        value: |
          name: kuard
          connect_timeout: 10ms
          type: STRICT_DNS
          lb_policy: ROUND_ROBIN
          load_assignment:
            cluster_name: kuard
            endpoints:
              - lb_endpoints:
                  - endpoint:
                      address:
                        socket_address: { address: 127.0.0.1, port_value: 8080 }
    listeners:
      - name: http
        value: |
          name: http
          # Changed listener port from 8081 to 5000
          address: { socket_address: { address: 0.0.0.0, port_value: 5000 } }
          filter_chains:
            - filters:
                - name: envoy.filters.network.http_connection_manager
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                    stat_prefix: ingress_https
                    route_config:
                      name: local_route
                      virtual_hosts:
                        - name: any
                          domains: ["*"]
                          routes:
                            [{ match: { prefix: "/" }, route: { cluster: "kuard" } }]
                    http_filters:
                      - name: envoy.filters.http.router
EOF
```

After applying this EnvoyConfig, the `CacheState` of the EnvoyConfig resource will be `Rollback` because all the Envoy proxies consuming this config rejected the listener update and MARIN3R detected it and reverted the config to the lastest working one.

```bash
▶ kubectl get envoyconfig kuard
NAME    NODE ID   ENVOY API   DESIRED VERSION   PUBLISHED VERSION   CACHE STATE
kuard   kuard     v3          7966f8f4b7        7cbf9788cd          Rollback
```

We can check that the kuard app is still available at port 8081.

```bash
▶ curl http://localhost:8081 -v -o /dev/null --silent
*   Trying ::1:8081...
* Connected to localhost (::1) port 8081 (#0)
> GET / HTTP/1.1
> Host: localhost:8081
> User-Agent: curl/7.76.1
> Accept: */*
>
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< content-length: 1592
< content-type: text/html
< date: Fri, 02 Jul 2021 14:22:36 GMT
< x-envoy-upstream-service-time: 1
< server: envoy
<
{ [1592 bytes data]
* Connection #0 to host localhost left intact
```

Next time a config update is applied and accepted by all proxies, the `Rollback` status will go back to `InSync`.

## **Conclusion**

We have seen how MARIN3R tracks the status of Envoy resource updates and rollbacks the configuration when it sees that a given update is being rejected by all proxies. It's important to note several considerations about the self-healing functionality:

- Failing configuration updates have an impact on Envoy performance because the xDS server will keep trying to send the update to the proxies and the proxies will keep trying to apply the update, failing in a loop. Even though this is mitigated by a backoff retry strategy in the xDS server-side, the self-healing functionality avoids this error loop from going on forever. There is one case when this situation is not revertible though, which occurs when there is no previous correct config to roll back to.
- To mark a configuration as "tainted" (a taint is what signals the controller that the config is failing and a rollback should be done) it is required that all the pods trying to apply the update fail. There are rare occasions where you could have some Pods accepting the config update and some others failing, and in this case, a human operator should look into the problem and decide the best way to proceed. This is why a rate of 100% failure is required for the self-healing functionality to kick in.
- In the case you have some kind of runtime issue that somehow ends up marking a configuration as "tainted" and after solving the problem you want to reapply the same config, you need to delete the specific config revision that was marked with the taint because MARIN3R never marks a config revision back as healthy once it has been marked as tainted. To do so, list all the EnvoyConfigRevision resources related to your EnvoyConfig and delete the tainted one. You can list the EnvoyConfigRevision resources for a given node-id using `kubectl get EnvoyConfigRevisions -l marin3r.3scale.net/node-id=<node-id>`

## **Cleanup**

Execute the following commands to delete the resources created in this walkthough:

```bash
kubectl delete deployment kuard
kubectl delete envoyconfig kuard
```
