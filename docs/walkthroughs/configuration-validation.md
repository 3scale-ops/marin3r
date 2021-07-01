# **Configuration Validation**

When MARIN3R is installed in a cluster, an [admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) is installed alongside the operator. One of the tasks of this admission webhook is to validate that the configurations a user includes in an EnvoyConfig resource are syntactically correct and adhere to the appropriate Envoy API spec.

## **Preparation**

You need to have MARIN3R operator installed in the cluster and a DiscoveryService within the namespace you will be using. Follow the [installation instructions](../../README.md#installation) to do so if you haven't already.

## **Deploy an EnvoyConfig**

Let's see what happens when we try to create an EnvoyConfig resource that contains an envoy cluster with a wrong configuration (in this case the error is in the connect_timeout units used):

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
          connect_timeout: 10 miliseconds
          type: STRICT_DNS
          lb_policy: ROUND_ROBIN
          load_assignment:
            cluster_name: kuard
            endpoints:
              - lb_endpoints:
                  - endpoint:
                      address:
                        socket_address: { address: 127.0.0.1, port_value: 8080 }
EOF
```

We get an error specifying that the units we are trying to use are not correct and the operation is rejected so the resource never gets created in the kubernetes api server. This provides quick feedback to the user, very useful when developing new configurations, and avoids having to troubleshoot problems by inspecting the Envoy logs.

```bash
Error from server ({"validationErrors":["Error deserializing resource: 'bad Duration: time: unknown unit \" miliseconds\" in duration \"10 miliseconds\"'"]}): error when creating "STDIN": admission webhook "envoyconfig.marin3r.3scale.net" denied the request: {"validationErrors":["Error deserializing resource: 'bad Duration: time: unknown unit \" miliseconds\" in duration \"10 miliseconds\"'"]}
```

Beware though, that even with the webhook performing this validation, there are times that even if the config is perfectly right from an API spec standpoint, not all versions of envoy support a given API spec exactly, as there may be deprecations and additions to the API between different versions of Envoy.

It's specially important that you check the [Envoy release notes](https://www.envoyproxy.io/docs/envoy/latest/version_history/version_history) when you are switching between Envoy versions in order to validate that all your EnvoyConfigs will still work after the change.
