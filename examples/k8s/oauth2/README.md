# OAUTH EXAMPLE

## How to use

* Set the values for your oauth account and redirect url in `oauth-config.env` and `oauth-secrets.env`
* Deploy the resources

```bash
❯ kustomize build examples/k8s/oauth2 | kubectl apply -f -
configmap/oauth-config created
secret/oauth-secrets created
service/kuard created
deployment.apps/kuard created
envoyconfig.marin3r.3scale.net/kuard created
discoveryservice.operator.marin3r.3scale.net/instance created
discoveryservicecertificate.operator.marin3r.3scale.net/kuard created
envoydeployment.operator.marin3r.3scale.net/kuard created
```

* Execute port-forward to access the EnvoyDeployment pod in localhost

```bash
❯ kubectl port-forward svc/oauth-proxy 8443:8443
Forwarding from 127.0.0.1:8443 -> 8443
Forwarding from [::1]:8443 -> 8443
```

* Open a browser and access `https://127.0.0.1.nip.io:8443/`, this will initiate the oauth flow.
