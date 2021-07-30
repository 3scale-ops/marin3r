# **TLS offloading with Envoy sidecars**

In this walkthrough, we will be deploying the [Kubernetes Up And Running Demo app](https://github.com/kubernetes-up-and-running/kuard) and setting TLS offloading for it using Envoy sidecar containers that will be automatically injected by MARIN3R in the pods. We will also see how automatic certificate updates occur, providing an easy way to automate the process of certificate renewal.

## **Preparation**

You need to have MARIN3R operator installed in the cluster and a DiscoveryService within the namespace you will be using. Follow the [installation instructions](../../README.md#installation) to do so if you haven't already.

## **Deploy the KUAR demo app**

First of all, we will be deploying the KUAR demo app.

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
        marin3r.3scale.net/ports: envoy-https:8443
        marin3r.3scale.net/envoy-api-version: v3
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

Note that we have added some labels and annotations to the Pod's metadata to activate sidecar injection for the Deployment, specifically the label `marin3r.3scale.net/status=enabled` and the annotation `marin3r.3scale.net/node-id=kuard`. The presence of these in a Pod makes the Pod creation request to be modified by MARIN3R's mutating webhook, which will inject an Envoy sidecar in the Pod's spec. The sidecar injection can be further configured with several other annotations, like the `marin3r.3scale.net/envoy-api-version` we are using to specify the Envoy API version we want to use. See the [configuration section](../../README.md#sidecar-injection-configuration) for a comprehensive list of the sidecar configuration annotations.

You should see that a new Pod is running, but it has 2 containers instead of the one we declared. An Envoy container just got added to the pod by MARIN3R:

```bash
kubectl get pods -l app=kuard
NAME                       READY   STATUS    RESTARTS   AGE
kuard-6bd9456d55-xbs7m     2/2     Running   0          32s
```

We need now to provide the Envoy sidecar with the appropriate config to publish the kuard application through https.

## **Create a certificate**

Use OpenSSL to create a self-signed certificate that we will be using for this example.

NOTE: you could also generate a certificate with cert-manager if you have it available in your cluster. This would be a typical case for a production environment. Check [cert-manager documentation](https://cert-manager.io/docs/).

```bash
openssl req -x509 -newkey rsa:4096 -keyout /tmp/key.pem -out /tmp/cert.pem -days 365 -nodes -subj '/CN=localhost'
```

Generate a kubernetes Secret from the certificate.

```bash
kubectl create secret tls kuard-certificate --cert=/tmp/cert.pem --key=/tmp/key.pem
```

## **Add an EnvoyConfig to publish the KUAR Demo app through https**

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
  envoyAPI: v3
  envoyResources:
    secrets:
      - name: kuard-certificate
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
      - name: https
        value: |
          name: https
          address: { socket_address: { address: 0.0.0.0, port_value: 8443 } }
          filter_chains:
            - transport_socket:
                name: envoy.transport_sockets.tls
                typed_config:
                  "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext
                  common_tls_context:
                    tls_certificate_sds_secret_configs:
                      - name: kuard-certificate
                        sds_config: { ads: {}, resource_api_version: "V3" }
              filters:
                - name: envoy.filters.network.http_connection_manager
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                    stat_prefix: ingress_https
                    route_config:
                      name: local_route
                      virtual_hosts:
                        - name: kuard
                          domains: ["*"]
                          routes:
                            [{ match: { prefix: "/" }, route: { cluster: "kuard" } }]
                    http_filters:
                      - name: envoy.filters.http.router
EOF
```

If you now check the status of the EnvoyConfig object, `CacheState` should show `InSync`.

```bash
$ kubectl get envoyconfig
NAME    NODEID   DESIRED VERSION   PUBLISHED VERSION   CACHE STATE
kuard   kuard    99d577784         99d577784           InSync
```

## **Expose the application for testing**

You can expose the kuard Deployment by running `kubectl port-forward` in your local environment. You could also expose the service through an Ingress or with a Service of the LoadBalancer type if your cluster provider supports it.

Run kubectl port-forward in a different shell:

```bash
$ kubectl port-forward deployment/kuard 8443:8443
Forwarding from 127.0.0.1:8443 -> 8443
Forwarding from [::1]:8443 -> 8443```
```

The service be can now accessed at `https://localhost:8443` via a browser or any other client, like curl. Due to the certificate used being self-signed, you will get a warning in the browser, and the `-k` flag must be used in curl:

```bash
▶ curl -kvs -o /dev/null https://localhost:8443
*   Trying ::1:8443...
* Connected to localhost (::1) port 8443 (#0)
* ALPN, offering h2
* ALPN, offering http/1.1
* successfully set certificate verify locations:
*  CAfile: /etc/pki/tls/certs/ca-bundle.crt
*  CApath: none
} [5 bytes data]
* TLSv1.3 (OUT), TLS handshake, Client hello (1):
} [512 bytes data]
* TLSv1.3 (IN), TLS handshake, Server hello (2):
{ [122 bytes data]
* TLSv1.3 (IN), TLS handshake, Encrypted Extensions (8):
{ [6 bytes data]
* TLSv1.3 (IN), TLS handshake, Certificate (11):
{ [1306 bytes data]
* TLSv1.3 (IN), TLS handshake, CERT verify (15):
{ [520 bytes data]
* TLSv1.3 (IN), TLS handshake, Finished (20):
{ [52 bytes data]
* TLSv1.3 (OUT), TLS change cipher, Change cipher spec (1):
} [1 bytes data]
* TLSv1.3 (OUT), TLS handshake, Finished (20):
} [52 bytes data]
* SSL connection using TLSv1.3 / TLS_AES_256_GCM_SHA384
* ALPN, server did not agree to a protocol
* Server certificate:
*  subject: CN=localhost
*  start date: Jul  1 11:28:36 2021 GMT
*  expire date: Jul  1 11:28:36 2022 GMT
*  issuer: CN=localhost
*  SSL certificate verify result: self signed certificate (18), continuing anyway.
} [5 bytes data]
> GET / HTTP/1.1
> Host: localhost:8443
> User-Agent: curl/7.76.1
> Accept: */*
>
{ [5 bytes data]
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
{ [230 bytes data]
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
{ [230 bytes data]
* old SSL session ID is stale, removing
{ [5 bytes data]
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< content-length: 1593
< content-type: text/html
< date: Thu, 01 Jul 2021 12:52:13 GMT
< x-envoy-upstream-service-time: 1
< server: envoy
<
{ [1593 bytes data]
* Connection #0 to host localhost left intact
```

Note the `Server: envoy` header we received, stating that the Envoy sidecar is proxying our request to the kuard container.

## **Test automatic reload of certificates**

Now that the KUAR Demo app is configured to be accessible through https, we will update the self-signed certificate being used and check that within seconds, MARIN3R will send the new certificate to the Envoy sidecars and these will automatically load the new one.

Let's reissue the certificate with a different common name, so we can easily identify which certificate is being returned. Note that in the previous section, the curl returned a certificate with `subject: CN=localhost`.

```bash
openssl req -x509 -newkey rsa:4096 -keyout /tmp/new-key.pem -out /tmp/new-cert.pem -days 365 -nodes -subj '/CN=new-certificate'
```

```bash
kubectl delete secret kuard-certificate
kubectl create secret tls kuard-certificate --cert=/tmp/new-cert.pem --key=/tmp/new-key.pem
```

If we execute the same curl again we will see that we get the new certificate instead of the one in the previous section. You can identify the new certificate by the change in the common name `subject: CN=new-certificate` of the server certificate.

```bash
▶ curl -kvs -o /dev/null https://localhost:8443
*   Trying ::1:8443...
* Connected to localhost (::1) port 8443 (#0)
* ALPN, offering h2
* ALPN, offering http/1.1
* successfully set certificate verify locations:
*  CAfile: /etc/pki/tls/certs/ca-bundle.crt
*  CApath: none
} [5 bytes data]
* TLSv1.3 (OUT), TLS handshake, Client hello (1):
} [512 bytes data]
* TLSv1.3 (IN), TLS handshake, Server hello (2):
{ [122 bytes data]
* TLSv1.3 (IN), TLS handshake, Encrypted Extensions (8):
{ [6 bytes data]
* TLSv1.3 (IN), TLS handshake, Certificate (11):
{ [1318 bytes data]
* TLSv1.3 (IN), TLS handshake, CERT verify (15):
{ [520 bytes data]
* TLSv1.3 (IN), TLS handshake, Finished (20):
{ [52 bytes data]
* TLSv1.3 (OUT), TLS change cipher, Change cipher spec (1):
} [1 bytes data]
* TLSv1.3 (OUT), TLS handshake, Finished (20):
} [52 bytes data]
* SSL connection using TLSv1.3 / TLS_AES_256_GCM_SHA384
* ALPN, server did not agree to a protocol
* Server certificate:
*  subject: CN=new-certificate
*  start date: Jul  1 13:10:18 2021 GMT
*  expire date: Jul  1 13:10:18 2022 GMT
*  issuer: CN=new-certificate
*  SSL certificate verify result: self signed certificate (18), continuing anyway.
} [5 bytes data]
> GET / HTTP/1.1
> Host: localhost:8443
> User-Agent: curl/7.76.1
> Accept: */*
>
{ [5 bytes data]
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
{ [230 bytes data]
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
{ [230 bytes data]
* old SSL session ID is stale, removing
{ [5 bytes data]
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< content-length: 1593
< content-type: text/html
< date: Thu, 01 Jul 2021 13:12:03 GMT
< x-envoy-upstream-service-time: 1
< server: envoy
<
{ [1593 bytes data]
* Connection #0 to host localhost left intact
```

## **Conclusion**

We have seen how we can easily configure TLS offloading using MARIN3R with Envoy sidecar containers. This idea can be extended to support a fully GitOps certificate management solution in a production environment, using [cert-manager](https://cert-manager.io) as the certificate provider. In this scenario, you [declaratively create certificates using cert-manager custom resources](https://cert-manager.io/docs/usage/certificate/), using any of the supported certificate providers like Let's Encrypt. If configured properly, cert-manager can renew certificates when required (certificates are stored as kubernetes Secrets) and certificate changes will be picked up by MARIN3R and distributed to the Envoy proxies for automatic reload of certificates in the servers.

## **Cleanup**

Execute the following commands to delete the resources created in this walkthough:

```bash
kubectl delete deployment kuard
kubectl delete envoyconfig kuard
kubectl delete secret kuard-certificate
```
