# **Upgrade from Envoy API v2 to v3**

**NOTE: API v2 is deprecated in MARIN3R releases > 0.9.0**

MARIN3R discovery services support both v2 and v3 APIs, which allows for non disruptive upgrades from Envoy v2 API to v3. This walkthrough describes the process of upgrading between API versions.

## **Preparation**

You need to have MARIN3R operator installed in the cluster and a DiscoveryService within the namespace you will be using. Follow the [installation instructions](../../README.md#installation) to do so if you haven't already.

## **Procedure for EnvoyDeployment resources**

### Deploy an EnvoyConfig and EnvoyDeployment using v2

Deploy the following v2 EnvoyConfig and EnvoyDeployment:

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: marin3r.3scale.net/v1alpha1
kind: EnvoyConfig
metadata:
  name: envoy
spec:
  nodeID: envoy
  serialization: yaml
  envoyAPI: v2
  envoyResources:
    listeners:
      - value: |
          name: http
          address:
            socket_address: { address: 0.0.0.0, port_value: 8080 }
          filter_chains:
            - filters:
                - name: envoy.http_connection_manager
                  typed_config:
                    "@type": type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager
                    stat_prefix: ingress_http
                    route_config:
                      name: route
                      virtual_hosts:
                        - name: any
                          domains: ["*"]
                          routes:
                            - {"match":{"prefix":"/"},"direct_response":{"status":200}}
                    http_filters:
                      - name: envoy.router
---
apiVersion: operator.marin3r.3scale.net/v1alpha1
kind: EnvoyDeployment
metadata:
  name: envoy
spec:
  discoveryServiceRef: discoveryservice
  envoyConfigRef: envoy
  # We need to use version v1.16.4 or lower because v2 API has been
  # deprecated starting v1.17.0 onwards
  image: envoyproxy/envoy:v1.16.4
  shutdownManager: {}
  ports:
    - name: http
      port: 8080
EOF
```

### Update the EnvoyConfig to v3

In order to upgrade to v3, you just need to set the `spec.envoyAPI` field to `v3` and update the syntax of your envoy resources to match the v3 API spec.

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
      - value: |
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

Immediately after applying the v3 config, a rollout will occur to update the EnvoyDeployment to use v3:

```bash
▶ kubectl get pods
NAME                                             READY   STATUS    RESTARTS   AGE
marin3r-discoveryservice-7b4c6d4c7c-fftg2        1/1     Running   0          64m
marin3r-envoydeployment-envoy-7494674d9d-7nx9h   0/1     Running   0          28s
marin3r-envoydeployment-envoy-7fd98988c-9m9n2    1/1     Running   0          53m
```

After the Deployment rollout is complete, the new Envoy pods will be using v3.

```bash
▶ kubectl get pods
NAME                                             READY   STATUS    RESTARTS   AGE
marin3r-discoveryservice-7b4c6d4c7c-fftg2        1/1     Running   0          65m
marin3r-envoydeployment-envoy-7494674d9d-7nx9h   1/1     Running   0          77s

▶ kubectl get envoyconfig
NAME    NODE ID   ENVOY API   DESIRED VERSION   PUBLISHED VERSION   CACHE STATE
envoy   envoy     v3          76795866f         76795866f           InSync
```

It's important to note that during the rollout process, both the v2 and the v3 versions of the config are being served by the discovery service. This is designed this way because during the rollout both v2 and v3 clients will be requesting configurations to the discovery service. In fact, the v2 EnvoyConfigRevisions (the internal operator resource that the EnvoyConfig controller to keep track of config versioning) will be never deleted by the operator, as we can see using the following command:

```bash
▶ kubectl get envoyconfigrevisions
NAME                 NODE ID   ENVOY API   VERSION     PUBLISHED   CREATED AT             LAST PUBLISHED AT      TAINTED
envoy-696bd4bdc      envoy     v2          696bd4bdc   true        2021-07-07T09:39:42Z   2021-07-07T09:39:42Z
envoy-v3-76795866f   envoy     v3          76795866f   true        2021-07-07T10:33:17Z   2021-07-07T10:33:17Z
```

### Cleanup

Execute the following commands to delete the resources created in this walkthough:

```bash
kubectl delete envoydeployment envoy
kubectl delete envoyconfig envoy
```

## **Procedure for Envoy sidecars**

The process to upgrade Envoy sidecars from v2 to v3 involves an extra step since when using sidecars the operator has no control over the lifecycle of the pods themselves, it just injects sidecar containers whenever a new Pod is created.


### Create a Deployment with Envoy sidecars

Let's first create a Deployment with Envoy sidecars to showcase the upgrade process.

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: marin3r.3scale.net/v1alpha1
kind: EnvoyConfig
metadata:
  name: envoy
spec:
  nodeID: envoy
  serialization: yaml
  envoyAPI: v2
  envoyResources:
    listeners:
      - value: |
          name: http
          address:
            socket_address: { address: 0.0.0.0, port_value: 8080 }
          filter_chains:
            - filters:
                - name: envoy.http_connection_manager
                  typed_config:
                    "@type": type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager
                    stat_prefix: ingress_http
                    route_config:
                      name: route
                      virtual_hosts:
                        - name: any
                          domains: ["*"]
                          routes:
                            - {"match":{"prefix":"/"},"direct_response":{"status":200}}
                    http_filters:
                      - name: envoy.router
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
        marin3r.3scale.net/ports: envoy-http:8081
        marin3r.3scale.net/envoy-api-version: v2
        marin3r.3scale.net/shutdown-manager.enabled: "true"
        marin3r.3scale.net/envoy-image: envoyproxy/envoy:v1.16.4
    spec:
      containers:
        - name: kuard
          image: gcr.io/kuar-demo/kuard-amd64:blue
          ports:
            - containerPort: 8080
              name: http
              protocol: TCP
EOF
```

You should see the following pods:

```bash
▶ kubectl get pods
NAME                                        READY   STATUS    RESTARTS   AGE
kuard-7dfdbb5656-vhtxk                      3/3     Running   0          61s
marin3r-discoveryservice-7b4c6d4c7c-fftg2   1/1     Running   0          90m
```

### Update the EnvoyConfig to v3

In order to upgrade to v3, you just need to set the `spec.envoyAPI` field to `v3` and update the syntax of your envoy resources to match the v3 API spec.

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
      - value: |
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

Check that the EnvoyConfig has been updated to v3:

```bash
▶ kubectl get envoyconfig
NAME    NODE ID   ENVOY API   DESIRED VERSION   PUBLISHED VERSION   CACHE STATE
envoy   envoy     v3          76795866f         76795866f           InSync
```

### Update the Envoy sidecars to use v3

At this point, we have our EnvoyConfig updated to v3 but our Envoy sidecars still use v2. To do so, we update the `marin3r.3scale.net/envoy-api-version` annotation in the `kuard` Deployment. This will trigger a rollout of the Deployment and v3 sidecars will be injected into the new pods.

```bash
▶ kubectl patch deployment kuard --type merge --patch '{"spec":{"template":{"metadata":{"annotations":{"marin3r.3scale.net/envoy-api-version":"v3"}}}}}'
deployment.apps/kuard patched
```

A Deployment rollout will happen immediately after applying the patch:

```bash
▶ kubectl get pods
NAME                                        READY   STATUS    RESTARTS   AGE
kuard-6bcdfcd9df-thg4r                      2/3     Running   0          31s
kuard-7dfdbb5656-vhtxk                      3/3     Running   0          8m11s
marin3r-discoveryservice-7b4c6d4c7c-fftg2   1/1     Running   0          107m
```

Once the rollout is complete all pods will be using the updated v3 EnvoyConfig.

### Cleanup

Execute the following commands to delete the resources created in this walkthough:

```bash
kubectl delete deployment kuard
kubectl delete envoyconfig envoy
```
