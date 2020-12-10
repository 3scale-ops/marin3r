<!-- omit in toc -->
# Development

The development process for MARIN3R is based on running locally the operator against a local [Kind (Kubernetes In Docker)](https://kind.sigs.k8s.io/docs/) cluster. The following sections describe the steps to take to accomplish different local testing scenarios.

- [Image generation](#image-generation)
- [Starting a Kind kubernetes cluster](#starting-a-kind-kubernetes-cluster)
- [Deleting a Kind kubernetes cluster](#deleting-a-kind-kubernetes-cluster)
- [Running the operator out-of-cluster](#running-the-operator-out-of-cluster)
- [Running the operator in-cluster](#running-the-operator-in-cluster)
- [Running a standalone discovery service locally](#running-a-standalone-discovery-service-locally)
- [Testing envoy configurations locally](#testing-envoy-configurations-locally)
- [Testing](#testing)
  - [Unit testing](#unit-testing)
  - [Integration testing](#integration-testing)
  - [e2e testing](#e2e-testing)

## Image generation

Generate a new image locally by issuing `make docker-build`. Even if the operator can be run locally without building the image, MARIN3R is a single binary that contains the code for the operator, the discovery service server and the mutating webhook, so you need to generate the image at least for the former two.

## Starting a Kind kubernetes cluster

Run a Kind cluster locally by using the following command. The Makefile will take care of downloading the proper `kind` binary:

```bash
make kind-create
```

The make target will also load the latest locally generated image into the cluster, which will be always tagged as `quay.io/3scale/marin3r:test`. This tag is updated each time you issue a `make docker-build` command.

Once the Kind cluster is up and running export the kubeconfig file into your shell:

```bash
export KUBECONFIG=${PWD}/kubeconfig
```

**NOTE**: remember to export the `KUBECONFIG` variable each time you open a new shell and want to interact with the Kind cluster.

## Deleting a Kind kubernetes cluster

Use `make kind-delete` to destroy the current Kind cluster.

## Running the operator out-of-cluster

With the [local Kind cluster](#starting-a-kind-kubernetes-cluster) running, install the CRDs in the cluster.

```bash
make install
```

Start tart the operator out-of-cluster. The operator will run with the admin priviledges given by the kubeconfig file.

```bash
make run
```

Deploy a DiscoveryService instance. There are several samples in the `examples` directory of the repo.

```bash
kubectl apply -f examples/e2e/deployment/discoveryservice.yaml
```

## Running the operator in-cluster

With the [local Kind cluster](#starting-a-kind-kubernetes-cluster) running, use kustomize to deploy everything into the cluster:

```bash
make kind-deploy
```

The operator will be deployed in the `marin3r-system` namespace and will run with the rbac permissions assigned to it.

## Running a standalone discovery service locally

In the case that you are developing functionality specific to the discovery service you can locally run the discovery service server and and envoy container running inside your local docker that connects to it.

Start by running the discovery service server. The discovery service server run the EnvoyConfig and EnvoyConfigRevision controllers itself, so you need to have the [local Kind cluster](#starting-a-kind-kubernetes-cluster) up and running beforehand. This discovery service will read resources from the `default` namespace.

```bash
make run-ds
```

In a different shell run an envoy pod that is already configured to talk to this discovery service server.

```bash
make run-envoy
```

You will see in the logs of the discovery service server that the envoy process connects to it as soon as it starts to request configurations:

```bash
[...]
urce": "kind source: /, Kind="}
2020-10-29T17:48:29.767+0100    INFO    controller      Starting workers        {"reconcilerGroup": "marin3r.3scale.net", "reconcilerKind": "EnvoyConfigRevision", "controller": "envoyconfigrevision", "worker count": 1}
2020-10-29T17:48:29.767+0100    INFO    controller      Starting Controller     {"reconcilerGroup": "marin3r.3scale.net", "reconcilerKind": "EnvoyConfig", "controller": "envoyconfig"}
2020-10-29T17:48:29.767+0100    INFO    controller      Starting workers        {"reconcilerGroup": "marin3r.3scale.net", "reconcilerKind": "EnvoyConfig", "controller": "envoyconfig", "worker count": 1}




2020-10-29T17:48:37.532+0100    DEBUG   envoy_control_plane     Stream opened   {"StreamId": 1}
2020-10-29T17:48:37.533+0100    DEBUG   envoy_control_plane     Received request        {"ResourceNames": [], "Version": "", "TypeURL": "type.googleapis.com/envoy.api.v2.Cluster", "NodeID": "envoy1", "StreamID": 1}
2020-10-29T17:48:52.282+0100    DEBUG   envoy_control_plane     Received request        {"ResourceNames": [], "Version": "", "TypeURL": "type.googleapis.com/envoy.api.v2.Listener", "NodeID": "envoy1", "StreamID": 1}
[...]
```

You can now deploy EnvoyConfig resources in the `default` namespace using the `nodeID=envoy1` as it is the one configured for the envoy process running in the docker container.

## Testing envoy configurations locally

The process of developing envoy configurations can be cumbersome. It is faster to first test the configurations locally to check for syntax errors or deprecation warnings.

First, create a static envoy config file. You can take `examples/local/static-config.yaml` as an example and add your resources under `static_resources`. Then use the following target to run the container with the given config:

```bash
make test-envoy-config CONFIG=<path to your config>
```

For example:

```bash
▶ make test-envoy-config CONFIG=examples/local/static-config.yaml
docker run -ti --rm \
        --network=host \
        -v $(pwd)/examples/local/static-config.yaml:/config.yaml \
        envoyproxy/envoy:v1.14.1 \
        envoy -c /config.yaml
[2020-10-29 17:04:12.663][1][info][main] [source/server/server.cc:255] initializing epoch 0 (hot restart version=11.104)
[2020-10-29 17:04:12.663][1][info][main] [source/server/server.cc:257] statically linked extensions:
[2020-10-29 17:04:12.663][1][info][main] [source/server/server.cc:259]   envoy.tracers: envoy.dynamic.ot, envoy.lightstep, envoy.tracers.datadog, envoy.tracers.dynamic_ot, envoy.tracers.lightstep, envoy.tracers.opencensus, envoy.tracers.xray, envoy.tracers.zipkin, envoy.zipkin
[...]
```

You can use a specific version of envoy using the following command:

```bash
make test-envoy-config CONFIG=<path to your config> ENVOY_VERSION=<version>
```

You can add additional arguments to the envoy command line, for example to increase logs levels:

```bash
make test-envoy-config CONFIG=examples/local/static-config.yaml ARGS="--component-log-level http:debug"
```

## Testing

You can run the whole test suite using `make test`. There are also targets to run each of the test suites.

### Unit testing

Unit tests can be run with:

```bash
make unit-test
```

### Integration testing

Integration tests can be run with:

```bash
make integration-test
```

### e2e testing

The e2e tests can be run with:

```bash
make e2e-test
```
