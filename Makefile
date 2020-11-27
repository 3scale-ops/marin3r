SHELL := /bin/bash
# Project name
NAME := marin3r
# Current Operator version
VERSION ?= 0.7.0-alpha2
# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)
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
IMG ?= $(IMG_NAME):latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

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
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG_NAME}:v${VERSION}
	cd config/webhook && $(KUSTOMIZE) edit set image controller=${IMG_NAME}:v${VERSION}
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
	# kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.1.0/cert-manager.crds.yaml
	kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.1.0/cert-manager.yaml

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

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG_NAME):$(VERSION)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

bump-release:
	sed -i 's/version string = "v\(.*\)"/version string = "v$(VERSION)"/g' pkg/version/version.go

prepare-release: bump-release generate fmt vet manifests bundle refdocs

#########################
#### General targets ####
#########################

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

tmp:
	mkdir -p $@

EASYRSA_VERSION ?= v3.0.6
certs:
	hack/gen-certs.sh $(EASYRSA_VERSION)

.PHONY: clean
clean: ## remove temporary resources from the repo
	rm -rf certs build tmp bin

#######################
#### Build targets ####
#######################

CURRENT_GIT_REF := $(shell git describe --always --dirty)
RELEASE := $(CURRENT_GIT_REF)

build: ## builds $(RELEASE) or HEAD of the current branch when $(RELEASE) is unset
build: build/bin/$(NAME)_amd64_$(RELEASE)

build/bin/$(NAME)_amd64_$(RELEASE):
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o build/bin/$(NAME)_amd64_$(RELEASE) main.go

clean-dirty-builds:
	rm -rf build/bin/*-dirty

docker-build: ## builds the docker image for $(RELEASE) or for HEAD of the current branch when $(RELEASE) is unset
docker-build: generate
	docker build . -t ${IMG_NAME}:$(RELEASE)
	docker tag ${IMG_NAME}:$(RELEASE) ${IMG_NAME}:test

######################
#### test targets ####
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
unit-test: fmt vet
	mkdir -p $(shell dirname $(COVERPROFILE))
	go test -p $(TEST_CPUS) ./controllers/... ./apis/... ./pkg/... -race -coverpkg="$(COVERPKGS)" -coverprofile=$(COVERPROFILE)

# Run integration tests
ENVTEST_ASSETS_DIR ?= $(shell pwd)/tmp
OPERATOR_COVERPROFILE = operator.marin3r.coverprofile
MARIN3R_COVERPROFILE = marin3r.coverprofile
integration-test: generate fmt vet manifests ginkgo
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || \
		curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; \
		fetch_envtest_tools $(ENVTEST_ASSETS_DIR); \
		setup_envtest_env $(ENVTEST_ASSETS_DIR); \
		ginkgo -p -r -cover -race -coverpkg=$(COVERPKGS) -outputdir=$(COVER_OUTPUT_DIR) ./controllers

coverprofile: gocovmerge
	gocovmerge $(COVER_OUTPUT_DIR)/$(UNIT_COVERPROFILE) $(COVER_OUTPUT_DIR)/$(OPERATOR_COVERPROFILE) $(COVER_OUTPUT_DIR)/$(MARIN3R_COVERPROFILE) > $(COVER_OUTPUT_DIR)/$(COVERPROFILE)
	$(MAKE) fix-cover COVERPROFILE=$(COVER_OUTPUT_DIR)/$(COVERPROFILE)
	go tool cover -func=$(COVER_OUTPUT_DIR)/$(COVERPROFILE) | awk '/total/{print $$3}'


e2e-test: export KUBECONFIG = ${PWD}/kubeconfig
e2e-test: kind-create
	$(MAKE) e2e-envtest-suite
	$(MAKE) kind-delete

e2e-envtest-suite: docker-build kind-load-image manifests ginkgo deploy-test
	ginkgo -r -nodes=1 ./test/e2e/operator
	ginkgo -r -p ./test/e2e/marin3r

test: unit-test integration-test e2e-test coverprofile

ginkgo:
	@which ginkgo > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		GO111MODULE=off go get -u github.com/onsi/ginkgo/ginkgo; \
	fi

gocovmerge:
	@which gocovmerge > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		GO111MODULE=off go get -u github.com/wadey/gocovmerge; \
	fi

############################################
#### Targets to manually test with Kind ####
############################################

KIND_VERSION ?= v0.9.0
KIND ?= bin/kind

$(KIND):
	mkdir -p $$(dirname $@)
	curl -sLo $(KIND) https://github.com/kubernetes-sigs/kind/releases/download/$(KIND_VERSION)/kind-$$(uname)-amd64
	chmod +x $(KIND)

kind-create: ## runs a k8s kind cluster with a local registry in "localhost:5000" and ports 1080 and 1443 exposed to the host
kind-create: export KUBECONFIG = ${PWD}/kubeconfig
kind-create: tmp $(KIND)
	$(KIND) create cluster --wait 5m --config test/kind.yaml
	$(KIND) load docker-image quay.io/3scale/marin3r:test --name kind

kind-deploy: export KUBECONFIG = ${PWD}/kubeconfig
kind-deploy: manifests kustomize
	$(KUSTOMIZE) build config/test | kubectl apply -f -

kind-refresh-discoveryservice: ## rebuilds the marin3r image, pushes it to the kind registry and recycles the marin3r pod
kind-refresh-discoveryservice: export KUBECONFIG = ${PWD}/kubeconfig
kind-refresh-discoveryservice: docker-build
	$(KIND) load docker-image quay.io/3scale/marin3r:test --name kind
	kubectl delete pods -A -l app.kubernetes.io/name=marin3r --force --grace-period=0

kind-load-image: export KUBECONFIG = ${PWD}/kubeconfig
kind-load-image:
	$(KIND) load docker-image quay.io/3scale/marin3r:test --name kind

kind-delete: ## deletes the kind cluster and the registry
kind-delete: $(KIND)
	$(KIND) delete cluster

###########################################
#### Targets to run components locally ####
###########################################

ENVOY_VERSION ?= v1.14.1

run-ds: ## locally starts marin3r's discovery service
run-ds: certs
	WATCH_NAMESPACE="" go run main.go \
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

CRD_REFDOCS_VERSION := v0.0.5
CRD_REFDOCS := bin/crd-ref-docs
$(CRD_REFDOCS):
		mkdir -p $$(dirname $@)
		curl -sLo $(CRD_REFDOCS) https://github.com/elastic/crd-ref-docs/releases/download/$(CRD_REFDOCS_VERSION)/crd-ref-docs
		chmod +x $(CRD_REFDOCS)

refdocs: $(CRD_REFDOCS) ## Generates api reference documentation from code
	crd-ref-docs \
		--source-path=apis \
		--config=docs/api-reference/config.yaml \
		--templates-dir=docs/api-reference/templates/asciidoctor \
		--renderer=asciidoctor \
		--output-path=docs/api-reference/reference.asciidoc

.FORCE:
