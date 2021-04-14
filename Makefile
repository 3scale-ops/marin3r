SHELL := /bin/bash
# Project name
NAME := marin3r
# Current Operator version
VERSION ?= 0.8.0-alpha.6
# Default bundle image tag
BUNDLE_IMG ?= quay.io/3scale/marin3r-bundle:v$(VERSION)
INDEX_IMG ?= quay.io/3scale/marin3r-catalog:latest
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG_NAME ?= quay.io/3scale/marin3r
IMG ?= $(IMG_NAME):v$(VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

#############################
### Makefile requirements ###
#############################

OS=$(shell uname | awk '{print tolower($$0)}')
ARCH = $(shell arch)
ifeq ($(shell arch),x86_64)
ARCH := amd64
endif
ifeq ($(shell arch),aarch64)
ARCH := arm64
endif

$(shell pwd)/bin:
	mkdir -p $(shell pwd)/bin

# Download operator-sdk binary if necesasry
OPERATOR_SDK_RELEASE = v1.5.0
OPERATOR_SDK = $(shell pwd)/bin/operator-sdk-$(OPERATOR_SDK_RELEASE)
OPERATOR_SDK_DL_URL = https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_RELEASE)/operator-sdk_$(OS)_$(ARCH)
$(OPERATOR_SDK): $(shell pwd)/bin
	curl -sL -o $(OPERATOR_SDK) $(OPERATOR_SDK_DL_URL)
	chmod +x $(OPERATOR_SDK)

# Download operator package manager if necessary
OPM_RELEASE = v1.16.1
OPM = $(shell pwd)/bin/opm-$(OPM_RELEASE)
OPM_DL_URL = https://github.com/operator-framework/operator-registry/releases/download/$(OPM_RELEASE)/$(OS)-$(ARCH)-opm
$(OPM): $(shell pwd)/bin
	curl -sL -o $(OPM) $(OPM_DL_URL)
	chmod +x $(OPM)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

# Download controller-gen locally if necessary
CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen:
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

# Download kustomize locally if necessary
KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize:
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# Download ginkgo locally if necessary
GINKGO = $(shell pwd)/bin/ginkgo
ginkgo:
	$(call go-get-tool,$(GINKGO),github.com/onsi/ginkgo/ginkgo)


# Download gocovmerge locally if necessary
GOCOVMERGE = $(shell pwd)/bin/gocovmerge
gocovmerge:
	$(call go-get-tool,$(GOCOVMERGE),github.com/wadey/gocovmerge)

# Download kind locally if necessary
KIND = $(shell pwd)/bin/kind
kind:
	$(call go-get-tool,$(KIND),sigs.k8s.io/kind@v0.9.0)

# Download crd-ref-docs locally if necessary
CRD_REFDOCS = $(shell pwd)/bin/crd-ref-docs
crd-ref-docs:
	$(call go-get-tool,$(CRD_REFDOCS),github.com/elastic/crd-ref-docs@v0.0.6)

#######################
### General targets ###
#######################

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

tmp:
	mkdir -p $@

.PHONY: clean
clean: ## remove temporary resources from the repo
	rm -rf certs tmp bin kubeconfig cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go operator --debug

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG_NAME):v$(VERSION)
	cd config/webhook && $(KUSTOMIZE) edit set image controller=$(IMG_NAME):v$(VERSION)
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# Undeploy controller from the configured Kubernetes cluster in ~/.kube/config
undeploy: manifests kustomize
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# Deploy controller (test configuration) in the configured Kubernetes cluster in ~/.kube/config
deploy-test: manifests kustomize
	$(KUSTOMIZE) build config/test | kubectl apply -f -

# Undeploy controller (test configuration) in the configured Kubernetes cluster in ~/.kube/config
undeploy-test: manifests kustomize
	$(KUSTOMIZE) build config/test | kubectl delete -f -

deploy-cert-manager:
	kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.1.0/cert-manager.yaml
	while [[ $$(kubectl -n cert-manager get deployment cert-manager-webhook -o 'jsonpath={.status.readyReplicas}') != "1" ]]; \
		do echo "waiting for cert-manager webhook" && sleep 3; \
	done

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	gofmt -s -w ./
# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build:
	docker build -t $(IMG) .
	docker tag $(IMG) $(IMG_NAME):test

# Push the docker image
docker-push:
	docker push $(IMG)

.PHONY: bundle
bundle: $(OPERATOR_SDK) manifests kustomize
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	cd config/webhook && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	# (2021-03-19) Remove the services generated by kustomize due
	# to a bug in OLM where upgrading a CSV fails when providing
	# a Service object as part of the bundle manifests. This problem
	# affects Openshift <4.6
	rm -vf bundle/manifests/*_v1_service.yaml
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

#########################
#### Release targets ####
#########################

prepare-alpha-release: bump-release generate fmt vet manifests bundle

prepare-release: bump-release generate fmt vet manifests bundle refdocs
	$(MAKE) bundle CHANNELS=alpha,stable DEFAULT_CHANNEL=alpha

bump-release:
	sed -i 's/version string = "v\(.*\)"/version string = "v$(VERSION)"/g' pkg/version/version.go

bundle-push:
	docker push $(BUNDLE_IMG)

index-build: $(OPM)
	$(OPM) index add \
		--build-tool docker \
		--mode semver \
		--bundles $(BUNDLE_IMG) \
		--from-index $(INDEX_IMG) \
		--tag $(INDEX_IMG)

index-push:
	docker push $(INDEX_IMG)

bundle-publish: bundle-build bundle-push index-build index-push


get-new-release:
	@hack/new-release.sh v$(VERSION)

######################
#### Test targets ####
######################

COVERPKGS = ./controllers/...,./apis/...,./pkg/...
COVER_OUTPUT_DIR = tmp/coverage
COVERPROFILE = total.coverprofile
TEST_CPUS ?= $(shell nproc)

$(COVER_OUTPUT_DIR):
	mkdir -p $(COVER_OUTPUT_DIR)

fix-cover:
	tmpfile=$$(mktemp) && grep -v "_generated.deepcopy.go" $(COVERPROFILE) > $${tmpfile} && cat $${tmpfile} > $(COVERPROFILE) && rm -f $${tmpfile}

# Run unit tests
UNIT_COVERPROFILE = unit.coverprofile
unit-test: export COVERPROFILE=$(COVER_OUTPUT_DIR)/$(UNIT_COVERPROFILE)
unit-test: export RUN_ENVTEST=0
unit-test: fmt vet $(COVER_OUTPUT_DIR)
	mkdir -p $(shell dirname $(COVERPROFILE))
	go test -p $(TEST_CPUS) ./controllers/... ./apis/... ./pkg/... -race -coverpkg="$(COVERPKGS)" -coverprofile=$(COVERPROFILE)

# Run integration tests
ENVTEST_ASSETS_DIR ?= $(shell pwd)/tmp
OPERATOR_COVERPROFILE = operator.coverprofile
MARIN3R_COVERPROFILE = marin3r.coverprofile
MARIN3R_WEBHOOK_COVERPROFILE = marin3r.webhook.coverprofile
OPERATOR_WEBHOOK_COVERPROFILE = operator.webhook.coverprofile
integration-test: generate fmt vet manifests ginkgo $(COVER_OUTPUT_DIR)
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || \
		curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; \
		fetch_envtest_tools $(ENVTEST_ASSETS_DIR); \
		setup_envtest_env $(ENVTEST_ASSETS_DIR); \
		$(GINKGO) -p -r -cover -coverpkg=$(COVERPKGS) -outputdir=$(COVER_OUTPUT_DIR) -coverprofile=$(MARIN3R_COVERPROFILE) ./controllers/marin3r; \
		$(GINKGO) -p -r -cover -coverpkg=$(COVERPKGS) -outputdir=$(COVER_OUTPUT_DIR) -coverprofile=$(OPERATOR_COVERPROFILE) ./controllers/operator.marin3r; \
		$(GINKGO) -p -r -cover -coverpkg=$(COVERPKGS) -outputdir=$(COVER_OUTPUT_DIR) -coverprofile=$(MARIN3R_WEBHOOK_COVERPROFILE) ./apis/marin3r; \
		$(GINKGO) -p -r -cover -coverpkg=$(COVERPKGS) -outputdir=$(COVER_OUTPUT_DIR) -coverprofile=$(OPERATOR_WEBHOOK_COVERPROFILE) ./apis/operator.marin3r

		# $(GINKGO) -p -r -cover -coverpkg=$(COVERPKGS) -outputdir=$(COVER_OUTPUT_DIR) ./controllers

coverprofile: gocovmerge
	$(GOCOVMERGE) \
		$(COVER_OUTPUT_DIR)/$(UNIT_COVERPROFILE) \
		$(COVER_OUTPUT_DIR)/$(OPERATOR_COVERPROFILE) \
		$(COVER_OUTPUT_DIR)/$(MARIN3R_COVERPROFILE) \
		$(COVER_OUTPUT_DIR)/$(MARIN3R_WEBHOOK_COVERPROFILE) \
		$(COVER_OUTPUT_DIR)/$(OPERATOR_WEBHOOK_COVERPROFILE) \
		> $(COVER_OUTPUT_DIR)/$(COVERPROFILE)
	$(MAKE) fix-cover COVERPROFILE=$(COVER_OUTPUT_DIR)/$(COVERPROFILE)
	go tool cover -func=$(COVER_OUTPUT_DIR)/$(COVERPROFILE) | awk '/total/{print $$3}'


e2e-test: export KUBECONFIG = $(PWD)/kubeconfig
e2e-test: kind-create
	$(MAKE) e2e-envtest-suite
	$(MAKE) kind-delete

e2e-envtest-suite: export KUBECONFIG = $(PWD)/kubeconfig
e2e-envtest-suite: docker-build kind-load-image manifests ginkgo deploy-test
	$(GINKGO) -r -p ./test/e2e

test: unit-test integration-test e2e-test coverprofile

############################################
#### Targets to manually test with Kind ####
############################################

kind-create: ## runs a k8s kind cluster with a local registry in "localhost:5000" and ports 1080 and 1443 exposed to the host
kind-create: export KUBECONFIG = $(PWD)/kubeconfig
kind-create: tmp docker-build kind
	$(KIND) create cluster --wait 5m --config test/kind.yaml
	$(MAKE) deploy-cert-manager
	$(KIND) load docker-image quay.io/3scale/marin3r:test --name kind

kind-deploy: export KUBECONFIG = $(PWD)/kubeconfig
kind-deploy: manifests kustomize
	$(KUSTOMIZE) build config/test | kubectl apply -f -

kind-refresh-discoveryservice: ## rebuilds the marin3r image, pushes it to the kind registry and recycles the marin3r pod
kind-refresh-discoveryservice: export KUBECONFIG = $(PWD)/kubeconfig
kind-refresh-discoveryservice: kind docker-build
	$(KIND) load docker-image quay.io/3scale/marin3r:test --name kind
	kubectl delete pods -A -l app.kubernetes.io/name=marin3r --force --grace-period=0

kind-load-image: export KUBECONFIG = $(PWD)/kubeconfig
kind-load-image: kind
	$(KIND) load docker-image quay.io/3scale/marin3r:test --name kind

kind-delete: ## deletes the kind cluster and the registry
kind-delete: kind
	$(KIND) delete cluster

###########################################
#### Targets to run components locally ####
###########################################

EASYRSA_VERSION ?= v3.0.6
certs:
	hack/gen-certs.sh $(EASYRSA_VERSION)

ENVOY_VERSION ?= v1.16.0

run-ds: ## locally starts marin3r's discovery service
run-ds: certs
	WATCH_NAMESPACE="default" go run main.go \
		discovery-service \
		--server-certificate-path certs/server \
		--ca-certificate-path certs/ca \
		--debug

run-envoy: ## executes an envoy process in a container that will try to connect to the local marin3r's discovery service
run-envoy: certs
	docker run -ti --rm \
		--network=host \
		--add-host marin3r.default.svc:127.0.0.1 \
		-v $$(pwd)/certs:/etc/envoy/tls \
		-v $$(pwd)/examples/local:/config \
		envoyproxy/envoy:$(ENVOY_VERSION) \
		envoy -c /config/envoy-client-bootstrap.yaml $(ARGS)



test-envoy-config: ## Run a local envoy container with the configuration passed in var CONFIG: "make test-envoy-config CONFIG=example/config.yaml". To debug problems with configs, increase envoy components log levels: make test-envoy-config CONFIG=example/envoy-ratelimit.yaml ARGS="--component-log-level http:debug"
test-envoy-config:
	@test -f $$(pwd)/$(CONFIG)
	docker run -ti --rm \
		--network=host \
		-v $$(pwd)/$(CONFIG):/config.yaml \
		envoyproxy/envoy:$(ENVOY_VERSION) \
		envoy -c /config.yaml $(ARGS)

grpc-proxy: ## executes an envoy process in a container that will try to connect to a local marin3r control plane
grpc-proxy: certs
	docker run -ti --rm \
		--network=host \
		--add-host marin3r.default.svc:127.0.0.1 \
		-v $$(pwd)/certs:/etc/envoy/tls \
		-v $$(pwd)/examples/local:/config \
		envoyproxy/envoy:$(ENVOY_VERSION) \
		envoy -c /config/discovery-service-proxy.yaml $(ARGS)

############################
#### refdocs generation ####
############################

refdocs: ## Generates api reference documentation from code
refdocs: crd-ref-docs
	$(CRD_REFDOCS) \
		--source-path=apis \
		--config=docs/api-reference/config.yaml \
		--templates-dir=docs/api-reference/templates/asciidoctor \
		--renderer=asciidoctor \
		--output-path=docs/api-reference/reference.asciidoc

.FORCE:
