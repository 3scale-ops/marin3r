NAME := marin3r
EASYRSA_VERSION := v3.0.6
ENVOY_VERSION := v1.14.1
CURRENT_GIT_REF := $(shell git describe --always --dirty)
RELEASE := $(CURRENT_GIT_REF)
KIND_VERSION := v0.7.0
KIND := bin/kind
export KUBECONFIG = tmp/kubeconfig
.PHONY: help clean kind-create kind-delete docker-build envoy start

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

tmp:
	mkdir -p $@

certs:
	script/gen-certs.sh $(EASYRSA_VERSION)

clean: ## remove temporary resources from the repo
	rm -rf certs build tmp bin

#######################
#### Build targets ####
#######################

build: ## builds $(RELEASE) or HEAD of the current branch when $(RELEASE) is unset
build: build/$(NAME)_amd64_$(RELEASE)

build/$(NAME)_amd64_$(RELEASE):
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o build/$(NAME)_amd64_$(RELEASE) cmd/main.go


docker-build: ## builds the docker image for $(RELEASE) or for HEAD of the current branch when $(RELEASE) is unset
docker-build: build/$(NAME)_amd64_$(RELEASE)
	docker build . -t ${IMAGE_NAME}:$(RELEASE) --build-arg RELEASE=$(RELEASE)

docker-push: ## pushes the image built from $(RELEASE) to quay.io
	docker push ${IMAGE_NAME}:$(RELEASE)

######################
#### Test targets ####
######################

TEST_RESULTS = ./coverage.txt

test-unit: ## runs unit tests
	go test ./... -race -coverprofile=$(TEST_RESULTS) -covermode=atomic

test: ## runs all tests
test: test-unit

#################################
#### Targets to test locally ####
#################################

envoy: ## executes an envoy process in a container that will try to connect to a local marin3r control plane
envoy: certs
	docker run -ti --rm \
		--network=host \
		--add-host marin3r:127.0.0.1 \
		-v $$(pwd)/certs:/etc/envoy/tls \
		-v $$(pwd)/example:/config \
		envoyproxy/envoy:$(ENVOY_VERSION) \
		envoy -c /config/envoy-bootstrap.yaml $(ARGS)

start: ## locally starts the marin3r control plane
start: certs
	KUBECONFIG=tmp/kubeconfig go run cmd/main.go \
		--certificate certs/marin3r.default.svc.crt \
		--private-key certs/marin3r.default.svc.key \
		--ca certs/ca.crt \
		--log-level debug \
		--namespace default \
		--out-of-cluster

###################################
#### Targets to test with Kind ####
###################################

$(KIND):
	mkdir -p $$(dirname $@)
	curl -sLo $(KIND) https://github.com/kubernetes-sigs/kind/releases/download/$(KIND_VERSION)/kind-$$(uname)-amd64
	chmod +x $(KIND)

kind-create: ## runs a k8s kind cluster with a local registry in "localhost:5000" and ports 1080 and 1443 exposed to the host
kind-create: export KIND_BIN=$(KIND)
kind-create: tmp $(KIND)
	script/kind-with-registry.sh

kind-docker-build: ## builds the docker image  $(RELEASE) or HEAD of the current branch when unset and pushes it to the kind local registry in "localhost:5000"
kind-docker-build: build/$(NAME)_amd64_$(RELEASE)
	docker build . -t ${IMAGE_NAME}:$(RELEASE) --build-arg RELEASE=$(RELEASE)
	docker tag ${IMAGE_NAME}:$(RELEASE) ${IMAGE_NAME}:test
	docker push ${IMAGE_NAME}:$(RELEASE)
	docker push ${IMAGE_NAME}:test

kind-start-marin3r: ## deploys marin3r inside the kind k8s cluster
kind-start-marin3r: export IMAGE_NAME = localhost:5000/${NAME}
kind-start-marin3r: certs kind-docker-build
	kubectl label namespace/default marin3r.3scale.net/status="enabled"
	kubectl create secret tls marin3r-server-cert --cert=certs/marin3r.default.svc.crt --key=certs/marin3r.default.svc.key
	kubectl create secret tls marin3r-ca-cert --cert=certs/ca.crt --key=certs/ca.key
	kubectl create secret tls envoy-sidecar-client-cert --cert=certs/envoy-client.crt --key=certs/envoy-client.key
	kubectl apply -f deploy/marin3r.yaml

kind-start-envoy: ## runs an envoy pod inside the k8s kind cluster that connects to the marin3r control plane
kind-start-envoy: certs
	kubectl create secret tls envoy1-cert --cert=certs/envoy-server1.crt --key=certs/envoy-server1.key
	kubectl annotate secret envoy1-cert cert-manager.io/common-name=envoy-server1
	kubectl apply -f deploy/envoy-config-cm.yaml
	kubectl apply -f deploy/envoy-pod.yaml

kind-refresh-marin3r: ## rebuilds the marin3r image, pushes it to the kind registry and recycles the marin3r pod
kind-refresh-marin3r: export IMAGE_NAME = localhost:5000/${NAME}
kind-refresh-marin3r: kind-docker-build
	kubectl delete pod -l app=marin3r

kind-logs-marin3r: ## shows the marin3r logs for the kind marin3r pod
	kubectl logs -f -l app=marin3r

kind-logs-envoy: ## shows the envoy logs for the kind envoy pod
	kubectl logs -f envoy1

kind-delete: ## deletes the kind cluster and the registry
kind-delete: $(KIND)
	$(KIND) delete cluster
	docker rm -f kind-registry

test-envoy-config: ## Run a local envoy container with the configuration passed in var CONFIG: make test-envoy-config CONFIG=example/config.yaml. To debug problems with configs, increase envoy components log levels: make test-envoy-config CONFIG=example/envoy-ratelimit.yaml ARGS="--component-log-level http:debug"
test-envoy-config:
	docker run -ti --rm \
		--network=host \
		-v $$(pwd)/$(CONFIG):/config.yaml \
		envoyproxy/envoy:$(ENVOY_VERSION) \
		envoy -c /config.yaml $(ARGS)

