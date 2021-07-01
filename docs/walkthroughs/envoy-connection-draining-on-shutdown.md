# **Connection draining on Pod shutdown**

When a container is terminated in kubernetes, by default a SIGTERM signal is sent to the PID 1 of the container to indicate that the process must stop. Envoy, by default, immediatly closes all connections and exits upon receiving a SIGTERM, which can cause errors for the connections that are in flight. To avoid this MARIN3R provides a mechanism to perform connection draining before shutting down the Envoy process.

This mechanism is called the **shutdown manager** and consists of an extra container that runs alongside the Envoy container and is in charge of draining connections by calling Envoy's admin API whenever the Envoy process is signaled to stop. The shutdown manager is not enabled by default, but you can activate it (and we strongly advise you to do so in production environments) both for Envoy sidecars and for envoydeployments.

In this walkthrough we are going to enable the shutdown manager for an EnvoyDeployment resource and validate the functionality with a simple test.

## **Preparation**

You need to have MARIN3R operator installed in the cluster and a DiscoveryService within the namespace you will be using. Follow the [installation instructions](../../README.md#installation) to do so if you haven't already.

## **Deploy an EnvoyConfig**

First of all, we need to create the Envoy configuration that our EnvoyDeployment will use. In this case, we are going to use a config that always returns a hardcoded 200 OK HTTP response.

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
                            - {"match":{"prefix":"/"},"direct_response":{"status":200}}
                    http_filters:
                      - name: envoy.filters.http.router
EOF
```

## **Deploy an EnvoyDeployment with shutdown manager enabled**

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
  # this enables the shutdown manager
  shutdownManager: {}
EOF
```

After some seconds you should see the following pods in the namespace:

```bash
▶ kubectl get pods
NAME                                             READY   STATUS    RESTARTS   AGE
marin3r-discoveryservice-f8bc788bd-2296r         1/1     Running   0          2m39s
marin3r-envoydeployment-envoy-64c976564f-jc8q7   2/2     Running   0          94s
```

## **Test the shutdown manager**

We are going to use `kubectl port-forward` to access the Envoy Pod. As usual, you can also use a Service of LoadBalancer type if your cluster supports it.

In a different shell execute:

```bash
▶ kubectl port-forward deployment/marin3r-envoydeployment-envoy 8080:8080
Forwarding from 127.0.0.1:8080 -> 8080
Forwarding from [::1]:8080 -> 8080
```

We should be able to curl our Envoy and get an HTTP 200 code.

```bash
▶ curl http://localhost:8080 -i
HTTP/1.1 200 OK
date: Mon, 05 Jul 2021 15:23:46 GMT
server: envoy
content-length: 0
```

Let's now set our EnvoyDeployment to zero replicas and see what happens. Before doing so, we will be opening a persistent connetion to the server using telnet. Open another shell and execute:

```bash
▶ telnet localhost 8080

Trying ::1...
Connected to localhost.
Escape character is '^]'.
```

Leave the connection open.

Patch the EnvoyDeployment resource to leave it with 0 replicas:

```bash
▶ kubectl patch envoydeployment envoy --type merge --patch '{"spec":{"replicas":{"static":0}}}'
envoydeployment.operator.marin3r.3scale.net/envoy patched
```

If you list the pods now you will see that the Envoy Pod is still in terminating, but not yet terminated. Our open telnet is preventing the Pod from terminating because the shutdown manager is waiting for all connections to be drained before proceeding with the shutdown of the server:

```bash
▶ kubectl get pods
NAME                                             READY   STATUS        RESTARTS   AGE
marin3r-discoveryservice-f8bc788bd-2296r         1/1     Running       0          20m
marin3r-envoydeployment-envoy-64c976564f-dhmct   1/2     Terminating   0          2m59s
```

We can check the logs of the shutdown manager to see how it checks the number of open connections to determine if it is safe to continue with the shutdown of the server:

```bash
▶ kubectl logs -c envoy-shtdn-mgr -f -l app.kubernetes.io/component=envoy-deployment,app.kubernetes.io/instance=envoy
{"level":"info","ts":1625499723.074794,"logger":"shutdownmanager","msg":"file /tmp/shutdown-ok does not exist, recheck in 1s","context":"waitForDrainHandler"}
{"level":"info","ts":1625499723.890846,"logger":"shutdownmanager","msg":"polled open connections","context":"DrainListeners","open_connections":1,"min_connections":0}
{"level":"info","ts":1625499724.0751417,"logger":"shutdownmanager","msg":"file /tmp/shutdown-ok does not exist, recheck in 1s","context":"waitForDrainHandler"}
{"level":"info","ts":1625499725.0754566,"logger":"shutdownmanager","msg":"file /tmp/shutdown-ok does not exist, recheck in 1s","context":"waitForDrainHandler"}
{"level":"info","ts":1625499726.0758047,"logger":"shutdownmanager","msg":"file /tmp/shutdown-ok does not exist, recheck in 1s","context":"waitForDrainHandler"}
{"level":"info","ts":1625499727.0760198,"logger":"shutdownmanager","msg":"file /tmp/shutdown-ok does not exist, recheck in 1s","context":"waitForDrainHandler"}
{"level":"info","ts":1625499728.0763586,"logger":"shutdownmanager","msg":"file /tmp/shutdown-ok does not exist, recheck in 1s","context":"waitForDrainHandler"}
{"level":"info","ts":1625499728.964318,"logger":"shutdownmanager","msg":"polled open connections","context":"DrainListeners","open_connections":1,"min_connections":0}
{"level":"info","ts":1625499729.0767376,"logger":"shutdownmanager","msg":"file /tmp/shutdown-ok does not exist, recheck in 1s","context":"waitForDrainHandler"}
```

Let's go back to the shell where we opened our telnet connection and close it. You will see that soon after closing the connection the Pod is finally terminated:

```bash
▶ kubectl get pods
NAME                                       READY   STATUS    RESTARTS   AGE
marin3r-discoveryservice-f8bc788bd-2296r   1/1     Running   0          24m
```

## **Conclusion**

In this walkthrough we have showcased how we can use the shutdown manager component of MARIN3R to ensure proper ordered shutdown of our proxies with connection draining, which is something usally desirable in production environments. Take into account that even with connection draining, MARIN3R configures the Envoy pods to be terminated anyway after 5 minutes if connection draining has not completed past that time.

## **Cleanup**

Execute the following commands to delete the resources created in this walkthough:

```bash
kubectl delete envoydeployment envoy
kubectl delete envoyconfig envoy
```
