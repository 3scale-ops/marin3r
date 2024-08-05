# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.13.1

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# quay.io/3scale/marin3r-bundle:$VERSION and quay.io/3scale/marin3r-catalog:$VERSION.
IMAGE_TAG_BASE ?= quay.io/3scale/marin3r

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
    BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Image URL to use all building/pushing image targets
IMG ?= $(IMAGE_TAG_BASE):v$(VERSION)

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.27

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

go-generate: gen-pkg-version gen-pkg-image gen-pkg-envoy-proto
	IMAGE=$(IMG) VERSION=$(VERSION) PATH=$$PATH:$$PWD/bin go generate ./...

##@ Test

test: generate fmt vet manifests go-generate unit-test integration-test e2e-test coverprofile ## Run tests and coverage


COVERPKGS = ./controllers/...,./apis/...,./pkg/...
COVER_OUTPUT_DIR = tmp/coverage
COVERPROFILE = total.coverprofile
TEST_CPUS ?= $(shell nproc)

$(COVER_OUTPUT_DIR):
	mkdir -p $(COVER_OUTPUT_DIR)

fix-cover:
	tmpfile=$$(mktemp) && grep -v "_generated.deepcopy.go" $(COVERPROFILE) > $${tmpfile} && cat $${tmpfile} > $(COVERPROFILE) && rm -f $${tmpfile}


UNIT_COVERPROFILE = unit.coverprofile
unit-test: export COVERPROFILE=$(COVER_OUTPUT_DIR)/$(UNIT_COVERPROFILE)
unit-test: export RUN_ENVTEST=0
unit-test: $(COVER_OUTPUT_DIR) ## Run unit tests
	mkdir -p $(shell dirname $(COVERPROFILE))
	go test -p $(TEST_CPUS) ./controllers/... ./apis/... ./pkg/... -race -coverpkg="$(COVERPKGS)" -coverprofile=$(COVERPROFILE)

OPERATOR_COVERPROFILE = operator.coverprofile
MARIN3R_COVERPROFILE = marin3r.coverprofile
OPERATOR_WEBHOOK_COVERPROFILE = operator.webhook.coverprofile
integration-test: export ACK_GINKGO_DEPRECATIONS=1.16.4
integration-test: envtest ginkgo $(COVER_OUTPUT_DIR) ## Run integration tests
		KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" $(GINKGO) -p -r -race -cover -coverpkg=$(COVERPKGS) -output-dir=$(COVER_OUTPUT_DIR) -coverprofile=$(OPERATOR_COVERPROFILE) ./controllers/operator.marin3r
		KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" $(GINKGO) -p -r -race -cover -coverpkg=$(COVERPKGS) -output-dir=$(COVER_OUTPUT_DIR) -coverprofile=$(OPERATOR_WEBHOOK_COVERPROFILE) ./apis/operator.marin3r
		KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" $(GINKGO) -p -r -race -cover -coverpkg=$(COVERPKGS) -output-dir=$(COVER_OUTPUT_DIR) -coverprofile=$(MARIN3R_COVERPROFILE) ./controllers/marin3r

coverprofile: unit-test integration-test gocovmerge ## Calculates test  coverage from unit and integration tests
	$(GOCOVMERGE) \
		$(COVER_OUTPUT_DIR)/$(UNIT_COVERPROFILE) \
		$(COVER_OUTPUT_DIR)/$(OPERATOR_COVERPROFILE) \
		$(COVER_OUTPUT_DIR)/$(MARIN3R_COVERPROFILE) \
		$(COVER_OUTPUT_DIR)/$(OPERATOR_WEBHOOK_COVERPROFILE) \
		> $(COVER_OUTPUT_DIR)/$(COVERPROFILE)
	$(MAKE) fix-cover COVERPROFILE=$(COVER_OUTPUT_DIR)/$(COVERPROFILE)
	go tool cover -func=$(COVER_OUTPUT_DIR)/$(COVERPROFILE) | awk '/total/{print $$3}'


e2e-test: export KUBECONFIG = $(PWD)/kubeconfig
e2e-test: kind-create ## Runs e2e test suite
	$(MAKE) e2e-envtest-suite
	$(MAKE) kind-delete

e2e-envtest-suite: export KUBECONFIG = $(PWD)/kubeconfig
e2e-envtest-suite: docker-build kind-load-image manifests ginkgo deploy-test
	$(GINKGO) -r -p ./test/e2e

##@ Build

build: manifests generate fmt vet go-generate ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet go-generate ## Run a controller from your host.
	go run ./main.go operator

docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} .
	docker tag $(IMG) $(IMAGE_TAG_BASE):test

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	cd config/webhook && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

deploy-test: manifests kustomize ## Deploy controller (test configuration) to the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/test | kubectl apply -f -

undeploy-test: manifests kustomize ## Undeploy controller (test configuration) from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/test | kubectl delete -f -

deploy-cert-manager: ## Deployes cert-manager in the K8s cluster specified in ~/.kube/config.
	kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.7.3/cert-manager.yaml
	kubectl -n cert-manager wait --timeout=300s --for=condition=Available deployments --all

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GINKGO ?= $(LOCALBIN)/ginkgo
GOCOVMERGE ?= $(LOCALBIN)/gocovmerge
CRD_REFDOCS ?= $(LOCALBIN)/crd-ref-docs
KIND ?= $(LOCALBIN)/kind

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.11.3
GINKGO_VERSION ?= v2.14.0
CRD_REFDOCS_VERSION ?= v0.0.8
KIND_VERSION ?= v0.16.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(KUSTOMIZE) || curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(CONTROLLER_GEN) || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(ENVTEST) || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.17

.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download ginkgo locally if necessary
$(GINKGO):
	test -s $(GINKGO) || GOBIN=$(LOCALBIN) go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION)

.PHONY: gocovmerge
gocovmerge: $(GOCOVMERGE) ## Download gocovmerge locally if necessary
$(GOCOVMERGE):
	test -s $(GOCOVMERGE) || GOBIN=$(LOCALBIN) go install github.com/wadey/gocovmerge@latest

.PHONY: crd-ref-docs
crd-ref-docs: ## Download crd-ref-docs locally if necessary
	test -s $(CRD_REFDOCS) || GOBIN=$(LOCALBIN) go install github.com/elastic/crd-ref-docs@$(CRD_REFDOCS_VERSION)

.PHONY: kind
KIND = $(shell pwd)/bin/kind
kind: $(KIND) ## Download kind locally if necessary
$(KIND):
	test -s $(KIND) || GOBIN=$(LOCALBIN) go install sigs.k8s.io/kind@$(KIND_VERSION)

##@ OLM related targets

.PHONY: bundle
bundle: operator-sdk manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	cd config/webhook && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS)
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.23.0/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Default catalog base image to append bundles to
CATALOG_BASE_IMG ?= $(IMAGE_TAG_BASE)-catalog:latest

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

.PHONY: catalog-add-bundle-to-alpha
catalog-add-bundle-to-alpha: opm ## Adds a bundle to a file based catalog
	$(OPM) render $(BUNDLE_IMGS) -oyaml > catalog/marin3r/objects/marin3r.v$(VERSION).clusterserviceversion.yaml
	yq -i '.entries += {"name": "marin3r.v$(VERSION)","replaces":"$(shell yq '.entries[-1].name' catalog/marin3r/alpha-channel.yaml)"}' catalog/marin3r/alpha-channel.yaml

.PHONY: catalog-add-bundle-to-stable
catalog-add-bundle-to-stable: opm ## Adds a bundle to a file based catalog
	$(OPM) render $(BUNDLE_IMGS) -oyaml > catalog/marin3r/objects/marin3r.v$(VERSION).clusterserviceversion.yaml
	yq -i '.entries += {"name": "marin3r.v$(VERSION)","replaces":"$(shell yq '.entries[-1].name' catalog/marin3r/alpha-channel.yaml)"}' catalog/marin3r/alpha-channel.yaml
	yq -i '.entries += {"name": "marin3r.v$(VERSION)","replaces":"$(shell yq '.entries[-1].name' catalog/marin3r/stable-channel.yaml)"}' catalog/marin3r/stable-channel.yaml

# Validate the catalog.
.PHONY: catalog-validate
catalog-validate: ## Push a catalog image.
	$(OPM) validate catalog/marin3r

.PHONY: catalog-build
catalog-build: opm catalog-validate ## Build a catalog image.
	docker build -f catalog/marin3r.Dockerfile -t $(CATALOG_IMG) catalog/

.PHONY: catalog-run
catalog-run: catalog-build
	docker run --rm -p 50051:50051 $(CATALOG_IMG)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)


##@ Kind Deployment

kind-create: export KUBECONFIG = $(PWD)/kubeconfig
kind-create: tmp docker-build kind ## Runs a k8s kind cluster with a local registry in "localhost:5000" and ports 1080 and 1443 exposed to the host
	$(KIND) create cluster --wait 5m --config test/kind.yaml --image kindest/node:v1.27.10
	$(MAKE) deploy-cert-manager
	$(KIND) load docker-image quay.io/3scale/marin3r:test --name kind

kind-deploy: export KUBECONFIG = $(PWD)/kubeconfig
kind-deploy: manifests kustomize ## Deploy operator to the Kind K8s cluster
	$(KUSTOMIZE) build config/test | kubectl apply -f -

kind-undeploy: export KUBECONFIG = $(PWD)/kubeconfig
kind-undeploy: ## Undeploy controller from the Kind K8s cluster
	$(KUSTOMIZE) build config/test | kubectl delete -f -

kind-load-image: export KUBECONFIG = $(PWD)/kubeconfig
kind-load-image: kind ## Reload the marin3r:test image into the cluster
	$(KIND) load docker-image quay.io/3scale/marin3r:test --name kind

kind-refresh-image: export KUBECONFIG = ${PWD}/kubeconfig
kind-refresh-image: manifests kind docker-build ## Reloads the image into the K8s cluster and deletes the old pods
	$(MAKE) kind-load-image
	kubectl -n marin3r-system delete pod -l control-plane=controller-manager
	kubectl -n marin3r-system delete pod -l control-plane=controller-webhook
	kubectl -n default delete pod -l app.kubernetes.io/component=discovery-service

kind-delete: ## Deletes the kind cluster and the registry
kind-delete: kind
	$(KIND) delete cluster

##@ Release

prepare-alpha-release: generate fmt vet manifests go-generate bundle ## Generates bundle manifests for alpha channel release

prepare-stable-release: generate fmt vet manifests go-generate bundle refdocs ## Generates bundle manifests for stable channel release
	$(MAKE) bundle CHANNELS=alpha,stable DEFAULT_CHANNEL=stable

bundle-publish: docker-build docker-push bundle-build bundle-push ## Builds and pushes operator and bundle images

catalog-publish: catalog-build catalog-push catalog-retag-latest ## Builds and pushes the catalog image

get-new-release: ## Checks if a release with the name $(VERSION) already exists in https://github.com/3scale-ops/marin3r/releases
	@hack/new-release.sh v$(VERSION)

catalog-retag-latest:
	docker tag $(CATALOG_IMG) $(IMAGE_TAG_BASE)-catalog:latest
	$(MAKE) docker-push IMG=$(IMAGE_TAG_BASE)-catalog:latest

##@ Run components locally
tmp/certs:
	hack/gen-certs.sh

ENVOY_VERSION ?= v1.23.2

run-ds: ## locally starts a discovery service
run-ds: manifests generate fmt vet go-generate tmp/certs
	WATCH_NAMESPACE="default" go run main.go \
		discovery-service \
		--server-certificate-path tmp/certs/server \
		--ca-certificate-path tmp/certs/ca \
		--client-certificate-path tmp/certs/client \
		--debug

run-envoy: ## runs an envoy process in a container that will try to connect to a local discovery service
run-envoy: tmp/certs
	docker run -ti --rm \
		--network=host \
		--add-host marin3r.default.svc:127.0.0.1 \
		-v $$(pwd)/tmp/certs/client:/etc/envoy/tls \
		-v $$(pwd)/examples/local:/config \
		envoyproxy/envoy:$(ENVOY_VERSION) \
		envoy -c /config/envoy-client-bootstrap.yaml $(ARGS)

##@ Other

.PHONY: operator-sdk
OPERATOR_SDK_RELEASE = v1.28.0
OPERATOR_SDK = bin/operator-sdk-$(OPERATOR_SDK_RELEASE)
operator-sdk: ## Download operator-sdk locally if necessary.
ifeq (,$(wildcard $(OPERATOR_SDK)))
ifeq (,$(shell which $(OPERATOR_SDK) 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPERATOR_SDK)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_RELEASE}/operator-sdk_$${OS}_$${ARCH};\
	chmod +x $(OPERATOR_SDK) ;\
	}
else
OPERATOR_SDK = $(shell which $(OPERATOR_SDK))
endif
endif

refdocs: ## Generates api reference documentation from code
refdocs: crd-ref-docs
	$(CRD_REFDOCS) \
		--source-path=apis \
		--config=docs/api-reference/config.yaml \
		--templates-dir=docs/api-reference/templates/asciidoctor \
		--renderer=asciidoctor \
		--output-path=docs/api-reference/reference.asciidoc

gen-pkg-envoy-proto: export TARGET_PATH = $(PWD)/bin
gen-pkg-envoy-proto: ## builds the gen-pkg-envoy-proto binary
	 cd generators/pkg-envoy-proto && go build -o $${TARGET_PATH}/gen-pkg-envoy-proto main.go

gen-pkg-version: export TARGET_PATH = $(PWD)/bin
gen-pkg-version: ## builds the gen-pkg-version binary
	 cd generators/pkg-version && go build -o $${TARGET_PATH}/gen-pkg-version main.go

tmp: ## Create project tmp directory
	mkdir tmp

gen-pkg-image: export TARGET_PATH = $(PWD)/bin
gen-pkg-image: ## builds the gen-pkg-image binary
	 cd generators/pkg-image && go build -o $${TARGET_PATH}/gen-pkg-image main.go

clean: ## Clean project directory
	rm -rf tmp bin kubeconfig