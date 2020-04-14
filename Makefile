NAME := marin3r
EASYRSA_VERSION := v3.0.6
ENVOY_VERSION := v1.13.1
CURRENT_GIT_REF := $(shell git describe --always --dirty)
RELEASE := $(CURRENT_GIT_REF)
KIND_VERSION := v0.7.0
KIND := bin/kind
export KUBECONFIG = tmp/kubeconfig
.PHONY: clean kind-create kind-delete docker-build compose envoy start

$(KIND):
	mkdir -p $$(dirname $@)
	curl -sLo $(KIND) https://github.com/kubernetes-sigs/kind/releases/download/$(KIND_VERSION)/kind-$$(uname)-amd64
	chmod +x $(KIND)

tmp:
	mkdir -p $@

kind-create: export KIND_BIN=$(KIND)
kind-create: tmp $(KIND)
	script/kind-with-registry.sh

kind-docker-build: ## Builds the docker image for HEAD of the current branch
kind-docker-build: build/$(NAME)_amd64_$(RELEASE)
	docker build . -t ${IMAGE_NAME}:$(RELEASE) --build-arg RELEASE=$(RELEASE)
	docker tag ${IMAGE_NAME}:$(RELEASE) ${IMAGE_NAME}/$(NAME):test
	docker push ${IMAGE_NAME}:$(RELEASE)
	docker push ${IMAGE_NAME}:test

kind-start-marin3r: export IMAGE_NAME = localhost:5000/${NAME}
kind-start-marin3r: certs kind-docker-build
	kubectl label namespace/default marin3r.3scale.net/status="enabled"
	kubectl create secret tls marin3r-server-cert --cert=certs/marin3r.default.svc.crt --key=certs/marin3r.default.svc.key
	kubectl create secret tls marin3r-ca-cert --cert=certs/ca.crt --key=certs/ca.key
	kubectl create secret tls envoy-sidecar-client-cert --cert=certs/envoy-client.crt --key=certs/envoy-client.key
	kubectl apply -f deploy/marin3r.yaml

kind-start-envoy1: certs
	kubectl create secret tls envoy1-cert --cert=certs/envoy-server1.crt --key=certs/envoy-server1.key
	kubectl annotate secret envoy1-cert cert-manager.io/common-name=envoy-server1
	kubectl apply -f deploy/envoy-config-cm.yaml
	kubectl apply -f deploy/envoy-pod.yaml

kind-refresh-marin3r: docker-build
	kubectl delete pod -l app=marin3r

kind-logs-marin3r:
	kubectl logs -f -l app=marin3r

kind-logs-envoy1:
	kubectl logs -f envoy1

kind-delete: $(KIND)
	$(KIND) delete cluster
	docker rm -f kind-registry


build: ## Builds $(RELEASE)
build: build/$(NAME)_amd64_$(RELEASE)

build/$(NAME)_amd64_$(RELEASE):
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o build/$(NAME)_amd64_$(RELEASE) cmd/main.go


docker-build: ## Builds the docker image for HEAD of the current branch
docker-build: build/$(NAME)_amd64_$(RELEASE)
	docker build . -t ${IMAGE_NAME}:$(RELEASE) --build-arg RELEASE=$(RELEASE)

docker-push: ## Pushes the image to quay.io
docker-push:
	docker push ${IMAGE_NAME}:$(RELEASE)

certs: ## use easyrsa to create testing CA and certificates for mTLS
	script/gen-certs.sh $(EASYRSA_VERSION)

compose: ## Compose file to test marin3r
compose: export RELEASE=$(CURRENT_GIT_REF)
compose: certs build/$(NAME)_amd64_$(RELEASE)
	docker-compose up

clean: ## Remove temporary resources from the repo
	rm -rf certs build tmp bin

envoy: ## execute an envoy process in a container that will try to connect to a local marin3r control plane
envoy: certs
	docker run -ti --rm \
		--network=host \
		--add-host marin3r:127.0.0.1 \
		-v $$(pwd)/certs:/etc/envoy/tls \
		-v $$(pwd)/example:/config \
		envoyproxy/envoy:$(ENVOY_VERSION) \
		envoy -c /config/envoy-bootstrap.yaml $(ARGS)

start: ## starts the marin3r contrl plane
start: certs
	KUBECONFIG=tmp/kubeconfig go run cmd/main.go \
		--certificate certs/marin3r-server.crt \
		--private-key certs/marin3r-server.key \
		--ca certs/ca.crt \
		--log-level debug \
		--namespace default \
		--out-of-cluster

TEST_RESULTS = ./coverage.txt

test-unit:
	go test ./... -race -coverprofile=$(TEST_RESULTS) -covermode=atomic

test: test-unit


