# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.9.1

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

# Image URL to use all building/pushing image targets
IMG ?= $(IMAGE_TAG_BASE):v$(VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.21

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
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

go-generate: gen-pkg-version gen-pkg-envoy-proto
	VERSION=$(VERSION) PATH=$$PATH:$$PWD/bin go generate ./...

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

ENVTEST_ASSETS_DIR ?= $(shell pwd)/tmp
OPERATOR_COVERPROFILE = operator.coverprofile
MARIN3R_COVERPROFILE = marin3r.coverprofile
MARIN3R_WEBHOOK_COVERPROFILE = marin3r.webhook.coverprofile
OPERATOR_WEBHOOK_COVERPROFILE = operator.webhook.coverprofile
export ACK_GINKGO_DEPRECATIONS=1.16.4
integration-test: ginkgo $(COVER_OUTPUT_DIR) ## Run integration tests
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || \
		curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; \
		fetch_envtest_tools $(ENVTEST_ASSETS_DIR); \
		setup_envtest_env $(ENVTEST_ASSETS_DIR); \
		$(GINKGO) -p -r -cover -coverpkg=$(COVERPKGS) -outputdir=$(COVER_OUTPUT_DIR) -coverprofile=$(MARIN3R_COVERPROFILE) ./controllers/marin3r; \
		$(GINKGO) -p -r -cover -coverpkg=$(COVERPKGS) -outputdir=$(COVER_OUTPUT_DIR) -coverprofile=$(OPERATOR_COVERPROFILE) ./controllers/operator.marin3r; \
		$(GINKGO) -p -r -cover -coverpkg=$(COVERPKGS) -outputdir=$(COVER_OUTPUT_DIR) -coverprofile=$(MARIN3R_WEBHOOK_COVERPROFILE) ./apis/marin3r; \
		$(GINKGO) -p -r -cover -coverpkg=$(COVERPKGS) -outputdir=$(COVER_OUTPUT_DIR) -coverprofile=$(OPERATOR_WEBHOOK_COVERPROFILE) ./apis/operator.marin3r

coverprofile: gocovmerge ## Calculates test  coverage from unit and integration tests
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
e2e-test: kind-create ## Runs e2e test suite
	$(MAKE) e2e-envtest-suite
	$(MAKE) kind-delete

e2e-envtest-suite: export KUBECONFIG = $(PWD)/kubeconfig
e2e-envtest-suite: docker-build kind-load-image manifests ginkgo deploy-test
	$(GINKGO) -r -p ./test/e2e

.PHONY: gocovmerge
GOCOVMERGE = $(shell pwd)/bin/gocovmerge
gocovmerge: ## Download gocovmerge locally if necessary
	$(call go-get-tool,$(GOCOVMERGE),github.com/wadey/gocovmerge)

.PHONY: ginkgo
GINKGO = $(shell pwd)/bin/ginkgo
ginkgo: ## Download ginkgo locally if necessary
	$(call go-get-tool,$(GINKGO),github.com/onsi/ginkgo/ginkgo)

##@ Build

build: generate fmt vet go-generate ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet go-generate ## Run a controller from your host.
	go run ./main.go

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

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

ENVTEST = $(shell pwd)/bin/setup-envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

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

.PHONY: bundle
bundle: operator-sdk manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	cd config/webhook && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
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
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.15.1/$${OS}-$${ARCH}-opm ;\
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


deploy-cert-manager: ## Deployes cert-manager in the K8s cluster specified in ~/.kube/config.
	kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.1.0/cert-manager.yaml
	while [[ $$(kubectl -n cert-manager get deployment cert-manager-webhook -o 'jsonpath={.status.readyReplicas}') != "1" ]]; \
		do echo "waiting for cert-manager webhook" && sleep 3; \
	done

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

##@ Kind Deployment

kind-create: export KUBECONFIG = $(PWD)/kubeconfig
kind-create: tmp docker-build kind ## Runs a k8s kind cluster with a local registry in "localhost:5000" and ports 1080 and 1443 exposed to the host
	$(KIND) create cluster --wait 5m --config test/kind.yaml
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

kind-delete: ## Deletes the kind cluster and the registry
kind-delete: kind
	$(KIND) delete cluster

.PHONY: kind
KIND = $(shell pwd)/bin/kind
kind: ## Download kind locally if necessary
	$(call go-get-tool,$(KIND),sigs.k8s.io/kind@v0.11.1)

##@ Release

prepare-alpha-release: generate fmt vet manifests go-generate bundle ## Generates bundle manifests for alpha channel release

prepare-stable-release: generate fmt vet manifests go-generate bundle refdocs ## Generates bundle manifests for stable channel release
	$(MAKE) bundle CHANNELS=alpha,stable DEFAULT_CHANNEL=stable

bundle-publish: docker-build docker-push bundle-build bundle-push catalog-build catalog-push catalog-retag-latest ## Generates and pushes all required images for a release

get-new-release: ## Checks if a release with the name $(VERSION) already exists in https://github.com/3scale-ops/marin3r/releases
	@hack/new-release.sh v$(VERSION)

catalog-retag-latest:
	docker tag $(CATALOG_IMG) $(IMAGE_TAG_BASE)-catalog:latest
	$(MAKE) docker-push IMG=$(IMAGE_TAG_BASE)-catalog:latest

##@ Run components locally

EASYRSA_VERSION ?= v3.0.6
certs:
	hack/gen-certs.sh $(EASYRSA_VERSION)

ENVOY_VERSION ?= v1.18.3

run-ds: ## locally starts a discovery service
run-ds: manifests generate fmt vet go-generate certs
	WATCH_NAMESPACE="default" go run main.go \
		discovery-service \
		--server-certificate-path certs/server \
		--ca-certificate-path certs/ca \
		--debug

run-envoy: ## runs an envoy process in a container that will try to connect to a local discovery service
run-envoy: certs
	docker run -ti --rm \
		--network=host \
		--add-host marin3r.default.svc:127.0.0.1 \
		-v $$(pwd)/certs:/etc/envoy/tls \
		-v $$(pwd)/examples/local:/config \
		envoyproxy/envoy:$(ENVOY_VERSION) \
		envoy -c /config/envoy-client-bootstrap.yaml $(ARGS)

##@ Other

$(shell pwd)/bin:
	mkdir -p $(shell pwd)/bin

.PHONY: operator-sdk
OPERATOR_SDK_RELEASE = v1.13.1
OPERATOR_SDK = bin/operator-sdk-$(OPERATOR_SDK_RELEASE)
operator-sdk: ## Download operator-sdk locally if necessary.
ifeq (,$(wildcard $(OPERATOR_SDK)))
ifeq (,$(shell which $(OPERATOR_SDK) 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPERATOR_SDK)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/v1.13.1/operator-sdk_$${OS}_$${ARCH};\
	chmod +x $(OPERATOR_SDK) ;\
	}
else
OPERATOR_SDK = $(shell which $(OPERATOR_SDK))
endif
endif

.PHONY: crd-ref-docs
CRD_REFDOCS = $(shell pwd)/bin/crd-ref-docs
crd-ref-docs: ## Download crd-ref-docs locally if necessary
	$(call go-get-tool,$(CRD_REFDOCS),github.com/elastic/crd-ref-docs@v0.0.7)

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
