NAME := marin3r
EASYRSA_VERSION := v3.0.6
ENVOY_VERSION := v1.14.1
CURRENT_GIT_REF := $(shell git describe --always --dirty)
RELEASE := $(CURRENT_GIT_REF)
KIND_VERSION := v0.9.0
KIND := bin/kind
export KUBECONFIG = ${PWD}/tmp/kubeconfig
.PHONY: help clean kind-create kind-delete docker-build envoy start build

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

tmp:
	mkdir -p $@

certs:
	hack/gen-certs.sh $(EASYRSA_VERSION)

clean: ## remove temporary resources from the repo
	rm -rf certs build tmp bin

#######################
#### Build targets ####
#######################
build: ## builds $(RELEASE) or HEAD of the current branch when $(RELEASE) is unset
build: build/bin/$(NAME)_amd64_$(RELEASE)

build/bin/$(NAME)_amd64_$(RELEASE):
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o build/bin/$(NAME)_amd64_$(RELEASE) main.go

clean-dirty-builds:
	rm -rf build/bin/*-dirty

# docker-build: ## builds the docker image for $(RELEASE) or for HEAD of the current branch when $(RELEASE) is unset
# docker-build: build/bin/$(NAME)_amd64_$(RELEASE)
# 	cd build && docker build . -t ${IMAGE_NAME}:$(RELEASE) --build-arg RELEASE=$(RELEASE)

# docker-push: ## pushes the image built from $(RELEASE) to quay.io
# 	docker push ${IMAGE_NAME}:$(RELEASE)

######################
#### Test targets ####
######################

# TEST_RESULTS = ./coverage.txt

# test-unit: ## runs unit tests
# 	go test ./pkg/... -race -coverprofile=$(TEST_RESULTS) -covermode=atomic

# test-e2e: ## run e2e tests
# test-e2e: kind-create kind-docker-build
# 	operator-sdk --verbose test local ./test/e2e --watch-namespace="" --operator-namespace="default" --up-local --local-operator-flags "--operator"
# 	$(MAKE) kind-delete

# test: ## runs all tests
# test: test-unit

###########################################
#### Targets to run components locally ####
###########################################

run-envoy: ## executes an envoy process in a container that will try to connect to a local marin3r control plane
run-envoy: certs
	docker run -ti --rm \
		--network=host \
		--add-host marin3r.default.svc:127.0.0.1 \
		-v $$(pwd)/certs:/etc/envoy/tls \
		-v $$(pwd)/examples/local:/config \
		envoyproxy/envoy:$(ENVOY_VERSION) \
		envoy -c /config/envoy-client-bootstrap.yaml $(ARGS)

grpc-proxy: ## executes an envoy process in a container that will try to connect to a local marin3r control plane
grpc-proxy: certs
	docker run -ti --rm \
		--network=host \
		--add-host marin3r.default.svc:127.0.0.1 \
		-v $$(pwd)/certs:/etc/envoy/tls \
		-v $$(pwd)/examples/local:/config \
		envoyproxy/envoy:$(ENVOY_VERSION) \
		envoy -c /config/discovery-service-proxy.yaml $(ARGS)

run-ds: ## locally starts marin3r's discovery service
run-ds: export KUBECONFIG=tmp/kubeconfig
run-ds: certs
	WATCH_NAMESPACE="" go run cmd/manager/main.go \
		--discovery-service \
		--certificate certs/marin3r.default.svc.crt \
		--private-key certs/marin3r.default.svc.key \
		--ca certs/ca.crt \
		--zap-devel

test-envoy-config: ## Run a local envoy container with the configuration passed in var CONFIG: "make test-envoy-config CONFIG=example/config.yaml". To debug problems with configs, increase envoy components log levels: make test-envoy-config CONFIG=example/envoy-ratelimit.yaml ARGS="--component-log-level http:debug"
test-envoy-config:
	docker run -ti --rm \
		--network=host \
		-v $$(pwd)/$(CONFIG):/config.yaml \
		envoyproxy/envoy:$(ENVOY_VERSION) \
		envoy -c /config.yaml $(ARGS)

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
	hack/kind-with-registry.sh

kind-docker-build: ## builds the docker image  $(RELEASE) or HEAD of the current branch when unset and pushes it to the kind local registry in "localhost:5000"
kind-docker-build: export IMAGE_NAME = localhost:5000/${NAME}
kind-docker-build:
	docker build . -t ${IMAGE_NAME}:$(RELEASE)
	docker tag ${IMAGE_NAME}:$(RELEASE) ${IMAGE_NAME}:latest
	docker push ${IMAGE_NAME}:$(RELEASE)
	docker push ${IMAGE_NAME}:latest

kind-install-certmanager:
	kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v0.14.3/cert-manager.crds.yaml
	kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v0.14.3/cert-manager.yaml

kind-start-marin3r: ## deploys marin3r inside the kind k8s cluster
kind-start-marin3r: certs kind-docker-build kind-apply-crds
	kubectl label namespace/default marin3r.3scale.net/status="enabled" || true
	kubectl create secret tls marin3r-server-cert --cert=certs/marin3r.default.svc.crt --key=certs/marin3r.default.svc.key || true
	kubectl create secret tls marin3r-ca-cert --cert=certs/ca.crt --key=certs/ca.key || true
	kubectl create secret tls envoy-sidecar-client-cert --cert=certs/envoy-client.crt --key=certs/envoy-client.key || true
	kubectl apply -f deploy/kind/marin3r.yaml
	while [[ $$(kubectl get pods -l app=marin3r -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]]; do sleep 5; done
	kubectl logs -f -l app=marin3r

kind-start-envoy: ## runs an envoy pod inside the k8s kind cluster that connects to the marin3r control plane
kind-start-envoy: certs
	kubectl create secret tls envoy1-cert --cert=certs/envoy-server1.crt --key=certs/envoy-server1.key || true
	kubectl annotate secret envoy1-cert cert-manager.io/common-name=envoy-server1 || true
	kubectl apply -f deploy/kind/envoy1-pod.yaml
	while [[ $$(kubectl get pods envoy1 -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]]; do sleep 5; done
	kubectl logs -f envoy1


kind-refresh-marin3r: ## rebuilds the marin3r image, pushes it to the kind registry and recycles the marin3r pod
kind-refresh-marin3r: export IMAGE_NAME = localhost:5000/${NAME}
kind-refresh-marin3r: kind-docker-build kind-apply-crds
	find deploy/crds -name "*_crd.yaml" -exec kubectl apply -f {} \;
	kubectl delete pod -l app=marin3r --force --grace-period=0

kind-delete: ## deletes the kind cluster and the registry
kind-delete: $(KIND)
	$(KIND) delete cluster
	docker rm -f kind-registry



AUTH_TOKEN = $(shell curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '{"user": {"username": "${QUAY_USERNAME}", "password": "${QUAY_PASSWORD}"}}' | jq -r '.token')

olm-install:
	operator-sdk olm install --timeout=5m

olm-generate-package-manifests: ## OPERATOR OLM CSV - Generate CSV Manifests
	operator-sdk generate csv --csv-version=$(RELEASE) --update-crds --make-manifests=false --csv-channel=alpha

olm-test-package-manifests:
	operator-sdk run packagemanifests --operator-version=$(RELEASE) --timeout=5m --install-mode='AllNamespaces=""'

olm-verify-package-manifests:
		operator-courier --verbose verify --ui_validate_io deploy/olm-catalog/marin3r

olm-push-package-manifests: ## OPERATOR OLM CSV - Push CSV manifests to remote application registry
olm-push-package-manifests: olm-verify-package-manifests
	operator-courier --verbose push deploy/olm-catalog/marin3r/ 3scaleops marin3r $(RELEASE) "$(AUTH_TOKEN)"

refdocs: ## Generates api reference documentation from code. Requires https://github.com/elastic/crd-ref-docs binary to be present in the PATH
	crd-ref-docs \
		--source-path=pkg/apis \
		--config=docs/api-reference/config.yaml \
		--templates-dir=docs/api-reference/templates/asciidoctor \
		--renderer=asciidoctor \
		--output-path=docs/api-reference/reference.asciidoc