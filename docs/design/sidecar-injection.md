# Sidecar injection

MARIN3R supports transparent injection of envoy sidecar containers into pods via a [kubernetes mutating admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook).

## Enable sidecar injection

For sidecar injection of envoy containers into Pods to work in a given namespace, the following prerequisites need to be met:

1. The label `marin3r.3scale.net/status=enabled` needs to be present in the Pod metadata
2. The annotation `marin3r.3scale.net=<nodeID>` needs to be present in the Pod metadata. The annotation's value must match the `spec.nodeID` field of the EnvoyConfig revision holding the config for the envoy proxy.
3. A client certificate for the envoy proxy to authenticate against the discovery service is required. This needs to be provided through a kubernetes Secret created in the namespace.
4. A static envoy config file or bootstrap configuration needs to be provided to the envoy proxy sidecar with the minimal configuration required to contact the discovery service server.

Points 1, 3 and 4 are managed automatically by MARIN3R's DiscoveryService controller when a DiscoveryService resource is created in a given namespace. For example, with the following DiscoveryService resource, the `default` namespace would get the required resources defined in 1, 3 and 4 created.

```yaml
apiVersion: operator.marin3r.3scale.net/v1alpha1
kind: DiscoveryService
metadata:
  name: discoveryservice
  namespace: default
spec: {}
```

Sidecar injection can also be manually configured for a namespace by creating the required resources manually. The names of the ConfigMap for the bootstrap envoy config and the Secret for the client certificate can be modified using Pod annotations as described in [this table](https://github.com/3scale-ops/marin3r#sidecar-injection-configuration).

The following diagram depicts the sidecar injection.

![Sidecar injection](sidecar-injection.svg)
