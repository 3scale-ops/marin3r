NAME := marin3r
EASYRSA_VERSION := v3.0.6
ENVOY_VERSION := v1.13.1
RELEASE := 0.0.0
CURRENT_GIT_REF := $(shell git describe --always --dirty)

build: ## Builds HEAD of the current branch
build: export RELEASE=$(CURRENT_GIT_REF)
build: build/$(NAME)_amd64_$(RELEASE)

# build/$(NAME)_amd64_$(CURRENT_GIT_REF):
# 	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o build/$(NAME)_amd64_$(RELEASE) cmd/main.go

build/$(NAME)_amd64_$(RELEASE):
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o build/$(NAME)_amd64_$(RELEASE) cmd/main.go

docker-build: ## Builds the docker image for $(RELEASE)
docker-build: build/$(NAME)_amd64_$(RELEASE)
	docker buil## Compose file to test marin3rd . -t quay.io/roivaz/$(NAME):v$(RELEASE) --build-arg release=$(RELEASE)

certs: ## use easyrsa to create testing CA and certificates for mTLS
	example/gen-certs.sh $(EASYRSA_VERSION)

compose: ## Compose file to test marin3r
compose: export RELEASE=$(CURRENT_GIT_REF)
compose: certs build/$(NAME)_amd64_$(RELEASE)
	docker-compose up

clean: ## Remove temporary resources from the repo
	rm -rf certs
	rm -rf build

envoy: ## execute an envoy process in a container that will try to connect to a local marin3r control plane
envoy: certs
	docker run -ti --rm \
		--network=host \
		--add-host marin3r:127.0.0.1 \
		-v $$(pwd)/certs:/etc/envoy/tls \
		-v $$(pwd)/example:/config \
		envoyproxy/envoy:$(ENVOY_VERSION) \
		envoy -c /config/envoy-config.yaml $(ARGS)


start: ## starts the marin3r contrl plane
start: certs
	KUBECONFIG=$(HOME)/.kube/config go run cmd/main.go