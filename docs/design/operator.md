# Operator

The operator part of MARIN3R is a group of controllers that manage the deployment and lifecycle of all the other components. It is composed by three different controllers:

* DiscoveryService controller
* DiscoveryServiceCertificate controller

## DiscoveryService controller

The DiscoveryService controller watches for DiscoveryService resources. The DiscoveryService resource is a namespace scoped resource and only watches for resources in its same namespace.

The DiscoveryService controller deploys the discovery service and sets up all the requirements for sidecar injection to work in a given namespace. It is also in charge of creating the certificates required for all components.

### Certificates

When a new DiscoveryService instance is created, a PKI is created to issue all the required certificates. To generate certificates, the DiscoveryService controller creates DiscoveryServiceCertificate resources. This is a list of all the certificates that are created:

* A self signed CA certificate that will be used as the root certificate for the DiscoveryService PKI. All other certificates are signed with this CA.
* A server certificate for both the discovery service server and the mutating webhook server.
* Client certificates for the envoy clients to authenticate against the discovery service server. One client certificate is issued per namespace enabled in the `spec.enabledNamespaces` field of the DiscoveryService resource.

When certificates change they need to be reloaded by the applications that are using them. There are currently two mechanisms to reload certificates.

#### Discovery service server certificate reload

The Pod where the discovery service server run has a label with the hash of the current certificate. Whenever that hash changes a new rollout of the Deployment is triggered, causing a restart and reload of the certificate from disk. This will cause some seconds of unavailability for the xDS server while the new Pod is being started. Running envoy pods won't be affected by this, but any new Pod created with sidecar injection enabled will fail to load resources from the discovery service until the latter is available again.

#### Envoy proxy client certificate reload

The sidecar envoy proxies consume the client certificate by mounting the Secret holding the certificate into the container's filesystem. Envoy watches the path of the certificate for changes in its contents and automatically reloads the certificate because MARIN3R uses filesystem service discovery in the envoy proxy container's configuration. This configuration looks like the following:

```yaml
      name: envoy.transport_sockets.tls
      typedConfig:
        "@type": type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext
        commonTlsContext:
          tlsCertificateSdsSecretConfigs:
          - sdsConfig:
              path: "/etc/envoy/bootstrap/tls_certificate_sds_secret.yaml"
```

## DiscoveryServiceCertificate controller

This controller is responsible for creating certificates matching the spec defined in DiscoveryServiceCertificate resources. Both self-signed and ca-signed certificates are supported.

Certificates are stored as kubernetes Secrets of type `kubernetes.io/tls`.

### Certificate renewal

The DiscoveryServiceCertificate controller starts trying to reissue a given certificate when the 80% of the certificate's duration has passed. Certificate renewal can be disabled setting `spec.certificateRenewalConfig.enabled: false` in the DiscoveryServiceCertificate resource.
